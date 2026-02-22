package activitypub

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
)

type LocalActorURLConfig struct {
	Host           string
	WebRoot        string
	FallbackHost   string
	FallbackScheme string
}

func LocalActorURL(cfg LocalActorURLConfig, username string) string {
	host := cfg.Host
	scheme := cfg.FallbackScheme
	if scheme == "" {
		scheme = "https"
	}

	if host == "" {
		host = cfg.FallbackHost
	} else {
		if parsedHost, err := url.Parse(host); err == nil && parsedHost.Host != "" {
			host = parsedHost.Host
			if parsedHost.Scheme != "" {
				scheme = parsedHost.Scheme
			}
		} else {
			scheme = "https"
		}
	}

	root := path.Join("/", cfg.WebRoot)
	root = path.Clean(root)
	if root == "/" || root == "." {
		root = ""
	}

	return fmt.Sprintf("%s://%s%s/ap/users/%s", scheme, host, root, username)
}

func SendSignedActivity(ctx context.Context, actorURL, privateKeyPEM, inbox string, payload []byte) error {
	if inbox == "" {
		return errors.New("inbox is empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, inbox, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", ContentType)
	req.Header.Set("Accept", ContentType)

	if err := SignRequest(req, privateKeyPEM, actorURL+"#main-key"); err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("remote inbox rejected activity: %s", resp.Status)
	}

	return nil
}

func SendFollowAccept(ctx context.Context, actorURL, privateKeyPEM string, follower model.Follower) error {
	if follower.ActorInbox == "" {
		return errors.New("follower inbox is empty")
	}

	follow := vocab.Activity{
		Type:   vocab.FollowType,
		Actor:  vocab.IRI(follower.ActorIRI),
		Object: vocab.IRI(actorURL),
	}

	accept := vocab.Activity{
		ID:     vocab.ID(fmt.Sprintf("%s#accept-follow-%d", actorURL, follower.ID)),
		Type:   vocab.AcceptType,
		Actor:  vocab.IRI(actorURL),
		Object: follow,
	}

	payload, err := jsonld.WithContext(
		jsonld.IRI(vocab.ActivityBaseURI),
	).Marshal(accept)
	if err != nil {
		return err
	}

	return SendSignedActivity(ctx, actorURL, privateKeyPEM, follower.ActorInbox, payload)
}

func SendFollow(ctx context.Context, actorURL, privateKeyPEM, inbox, targetActorIRI string) error {
	follow := vocab.Activity{
		ID:     vocab.ID(fmt.Sprintf("%s#follow-%d", actorURL, time.Now().UTC().UnixNano())),
		Type:   vocab.FollowType,
		Actor:  vocab.IRI(actorURL),
		Object: vocab.IRI(targetActorIRI),
	}

	payload, err := jsonld.WithContext(
		jsonld.IRI(vocab.ActivityBaseURI),
	).Marshal(follow)
	if err != nil {
		return err
	}

	return SendSignedActivity(ctx, actorURL, privateKeyPEM, inbox, payload)
}

func SendUndoFollow(ctx context.Context, actorURL, privateKeyPEM, inbox, targetActorIRI string) error {
	follow := vocab.Activity{
		Type:   vocab.FollowType,
		Actor:  vocab.IRI(actorURL),
		Object: vocab.IRI(targetActorIRI),
	}

	undo := vocab.Activity{
		ID:     vocab.ID(fmt.Sprintf("%s#undo-follow-%d", actorURL, time.Now().UTC().UnixNano())),
		Type:   vocab.UndoType,
		Actor:  vocab.IRI(actorURL),
		Object: follow,
	}

	payload, err := jsonld.WithContext(
		jsonld.IRI(vocab.ActivityBaseURI),
	).Marshal(undo)
	if err != nil {
		return err
	}

	return SendSignedActivity(ctx, actorURL, privateKeyPEM, inbox, payload)
}
