package aputil

import (
	"context"
	"fmt"
	"io"
	"net/http"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
)

type actorHTTPClient struct {
	client *http.Client
}

func (c actorHTTPClient) CtxGet(ctx context.Context, uri string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", ContentType)

	return c.client.Do(req)
}

func (c actorHTTPClient) LoadActor(ctx context.Context, actorIRI string) (*vocab.Actor, error) {
	resp, err := c.CtxGet(ctx, actorIRI)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("failed to load actor: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var actor vocab.Actor
	if err := jsonld.Unmarshal(body, &actor); err != nil {
		return nil, err
	}

	return &actor, nil
}
