package activitypub

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
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
