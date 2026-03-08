package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWantsActivityPub(t *testing.T) {
	tests := []struct {
		name   string
		accept string
		want   bool
	}{
		{
			name:   "empty accept",
			accept: "",
			want:   false,
		},
		{
			name:   "browser html",
			accept: "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			want:   false,
		},
		{
			name:   "activity+json",
			accept: "application/activity+json",
			want:   true,
		},
		{
			name:   "ld+json",
			accept: "application/ld+json",
			want:   true,
		},
		{
			name:   "ld+json with profile",
			accept: `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`,
			want:   true,
		},
		{
			name:   "multiple types including AP",
			accept: "text/html, application/activity+json",
			want:   true,
		},
		{
			name:   "wildcard only",
			accept: "*/*",
			want:   false,
		},
		{
			name:   "json only",
			accept: "application/json",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.accept != "" {
				req.Header.Set("Accept", tt.accept)
			}
			got := wantsActivityPub(req)
			assert.Equal(t, tt.want, got)
		})
	}
}
