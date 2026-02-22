package activitypub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
)

type webFingerResponse struct {
	Links []struct {
		Rel  string `json:"rel"`
		Type string `json:"type"`
		Href string `json:"href"`
	} `json:"links"`
}

func ResolveActorIRIFromWebFinger(ctx context.Context, username, host string) (string, error) {
	username = strings.TrimSpace(strings.TrimPrefix(username, "@"))
	host = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(host, "https://"), "http://"))
	if username == "" || host == "" {
		return "", errors.New("invalid handle")
	}

	resource := url.QueryEscape(fmt.Sprintf("acct:%s@%s", username, host))
	candidates := []string{
		fmt.Sprintf("https://%s/.well-known/webfinger?resource=%s", host, resource),
	}

	var lastErr error
	for _, endpoint := range candidates {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Accept", "application/jrd+json, application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}

		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			lastErr = fmt.Errorf("webfinger rejected: %s", resp.Status)
			continue
		}

		parsed := webFingerResponse{}
		if err := json.Unmarshal(body, &parsed); err != nil {
			lastErr = err
			continue
		}

		for _, link := range parsed.Links {
			if link.Rel != "self" || link.Href == "" {
				continue
			}

			typ := strings.TrimSpace(strings.ToLower(link.Type))
			if typ == "" || typ == "application/activity+json" || strings.HasPrefix(typ, "application/ld+json") {
				return link.Href, nil
			}
		}

		lastErr = errors.New("no ActivityPub self link found")
	}

	if lastErr == nil {
		lastErr = errors.New("could not resolve actor via webfinger")
	}

	return "", lastErr
}

func LoadRemoteActor(ctx context.Context, actorIRI string) (*vocab.Actor, error) {
	client := actorHTTPClient{client: http.DefaultClient}
	return client.LoadActor(ctx, actorIRI)
}

func LoadCollectionTotalItems(ctx context.Context, collectionIRI string) (int64, error) {
	if strings.TrimSpace(collectionIRI) == "" {
		return 0, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, collectionIRI, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", ContentType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return 0, fmt.Errorf("collection fetch rejected: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	ordered := vocab.OrderedCollection{}
	if err := jsonld.Unmarshal(body, &ordered); err == nil {
		return int64(ordered.TotalItems), nil
	}

	plain := vocab.Collection{}
	if err := jsonld.Unmarshal(body, &plain); err == nil {
		return int64(plain.TotalItems), nil
	}

	return 0, errors.New("could not parse collection")
}
