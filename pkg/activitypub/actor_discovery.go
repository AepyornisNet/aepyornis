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

func itemIRIString(it vocab.Item) string {
	if vocab.IsNil(it) {
		return ""
	}

	if vocab.IsIRI(it) {
		return it.GetLink().String()
	}

	iri := ""
	_ = vocab.OnLink(it, func(link *vocab.Link) error {
		iri = link.Href.String()
		return nil
	})

	return iri
}

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
	endpoint := fmt.Sprintf("https://%s/.well-known/webfinger?resource=%s", host, resource)

	client := actorHTTPClient{client: http.DefaultClient}
	resp, err := client.CtxGet(ctx, endpoint)
	if err != nil {
		return "", err
	}

	body, readErr := io.ReadAll(resp.Body)
	resp.Body.Close()
	if readErr != nil {
		return "", readErr
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("webfinger rejected: %s", resp.Status)
	}

	parsed := webFingerResponse{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
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

	return "", errors.New("no ActivityPub self link found")
}

func LoadRemoteActor(ctx context.Context, actorIRI string) (*vocab.Actor, error) {
	client := actorHTTPClient{client: http.DefaultClient}
	return client.LoadActor(ctx, actorIRI)
}

func LoadCollectionTotalItems(ctx context.Context, collectionIRI string) (int64, error) {
	if strings.TrimSpace(collectionIRI) == "" {
		return 0, nil
	}

	client := actorHTTPClient{client: http.DefaultClient}
	resp, err := client.CtxGet(ctx, collectionIRI)
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

func ResolveObjectActorAndInbox(ctx context.Context, objectIRI string) (string, string, error) {
	trimmed := strings.TrimSpace(objectIRI)
	if trimmed == "" {
		return "", "", errors.New("object IRI is required")
	}

	client := actorHTTPClient{client: http.DefaultClient}
	resp, err := client.CtxGet(ctx, trimmed)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", "", fmt.Errorf("object fetch rejected: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	actorIRI := ""
	activity := vocab.Activity{}
	if err := jsonld.Unmarshal(body, &activity); err == nil {
		actorIRI = itemIRIString(activity.Actor)
		if actorIRI == "" {
			_ = vocab.OnObject(activity.Object, func(object *vocab.Object) error {
				if object != nil {
					actorIRI = itemIRIString(object.AttributedTo)
				}
				return nil
			})
		}
	}

	if actorIRI == "" {
		object := vocab.Object{}
		if err := jsonld.Unmarshal(body, &object); err == nil {
			actorIRI = itemIRIString(object.AttributedTo)
		}
	}

	if actorIRI == "" {
		return "", "", errors.New("object actor not found")
	}

	actor, err := LoadRemoteActor(ctx, actorIRI)
	if err != nil {
		return "", "", err
	}

	inbox := itemIRIString(actor.Inbox)
	if inbox == "" {
		return "", "", errors.New("actor inbox not found")
	}

	return actorIRI, inbox, nil
}
