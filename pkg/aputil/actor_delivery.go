package aputil

import (
	"fmt"
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
