package activitypub

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalActorURL(t *testing.T) {
	tests := []struct {
		name     string
		cfg      LocalActorURLConfig
		username string
		expected string
	}{
		{
			name:     "plain host and username",
			cfg:      LocalActorURLConfig{Host: "example.com"},
			username: "alice",
			expected: "https://example.com/ap/users/alice",
		},
		{
			name:     "host with web root",
			cfg:      LocalActorURLConfig{Host: "example.com", WebRoot: "/app"},
			username: "alice",
			expected: "https://example.com/app/ap/users/alice",
		},
		{
			name:     "fallback host used when host is empty",
			cfg:      LocalActorURLConfig{FallbackHost: "fallback.example.com"},
			username: "bob",
			expected: "https://fallback.example.com/ap/users/bob",
		},
		{
			name:     "host with explicit https scheme",
			cfg:      LocalActorURLConfig{Host: "https://example.com"},
			username: "carol",
			expected: "https://example.com/ap/users/carol",
		},
		{
			name:     "host with explicit http scheme",
			cfg:      LocalActorURLConfig{Host: "http://example.com"},
			username: "dave",
			expected: "http://example.com/ap/users/dave",
		},
		{
			name:     "fallback scheme applied to fallback host",
			cfg:      LocalActorURLConfig{FallbackHost: "example.com", FallbackScheme: "http"},
			username: "eve",
			expected: "http://example.com/ap/users/eve",
		},
		{
			name:     "web root slash is collapsed",
			cfg:      LocalActorURLConfig{Host: "example.com", WebRoot: "/"},
			username: "frank",
			expected: "https://example.com/ap/users/frank",
		},
		{
			name:     "web root without leading slash",
			cfg:      LocalActorURLConfig{Host: "example.com", WebRoot: "app"},
			username: "grace",
			expected: "https://example.com/app/ap/users/grace",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := LocalActorURL(tc.cfg, tc.username)
			assert.Equal(t, tc.expected, result)
		})
	}
}
