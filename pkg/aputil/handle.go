package aputil

import (
	"errors"
	"net/url"
	"strings"
)

var ErrInvalidHandle = errors.New("invalid handle")

func ParseActorHandle(handle string) (string, string, error) {
	h := strings.TrimSpace(strings.TrimPrefix(handle, "@"))
	if h == "" {
		return "", "", ErrInvalidHandle
	}

	if parsedURL, err := url.Parse(h); err == nil && parsedURL.Host != "" {
		segments := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
		if len(segments) == 3 && segments[0] == "ap" && segments[1] == "users" && segments[2] != "" {
			return segments[2], parsedURL.Host, nil
		}
	}

	if strings.Contains(h, "@") {
		parts := strings.SplitN(h, "@", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", ErrInvalidHandle
		}

		return parts[0], parts[1], nil
	}

	return h, "", nil
}
