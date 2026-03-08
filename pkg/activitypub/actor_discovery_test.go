package activitypub

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	vocab "github.com/go-ap/activitypub"
	"github.com/jovandeginste/workout-tracker/v2/pkg/activitypub/aptest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestItemIRIString verifies that item IRI values are extracted correctly from
// different vocab.Item representations.
func TestItemIRIString(t *testing.T) {
	t.Run("nil item returns empty string", func(t *testing.T) {
		assert.Equal(t, "", itemIRIString(nil))
	})

	t.Run("IRI item returns the IRI string", func(t *testing.T) {
		iri := vocab.IRI("https://example.com/users/alice")
		assert.Equal(t, "https://example.com/users/alice", itemIRIString(iri))
	})

	t.Run("Link item returns the href", func(t *testing.T) {
		link := &vocab.Link{Href: "https://example.com/users/bob"}
		assert.Equal(t, "https://example.com/users/bob", itemIRIString(link))
	})
}

// TestResolveActorIRIFromWebFinger_ValidationErrors verifies that invalid inputs
// are rejected before any network request is made.
func TestResolveActorIRIFromWebFinger_ValidationErrors(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		username string
		host     string
	}{
		{"empty username", "", "example.com"},
		{"empty host", "alice", ""},
		{"both empty", "", ""},
		{"whitespace username", "   ", "example.com"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ResolveActorIRIFromWebFinger(ctx, tc.username, tc.host)
			require.Error(t, err)
		})
	}
}

// TestResolveActorIRIFromWebFinger_Success verifies the happy path: a valid
// WebFinger response is parsed and the actor IRI is returned.
func TestResolveActorIRIFromWebFinger_Success(t *testing.T) {
	ctx := context.Background()

	mux := http.NewServeMux()
	server := httptest.NewTLSServer(mux)
	t.Cleanup(server.Close)
	aptest.UseTestTLSTransport(t, server)

	serverHost, _ := url.Parse(server.URL)
	actorIRI := server.URL + "/users/wfuser"

	mux.HandleFunc("/.well-known/webfinger", aptest.WebFingerHandler(actorIRI))

	result, err := ResolveActorIRIFromWebFinger(ctx, "wfuser", serverHost.Host)
	require.NoError(t, err)
	assert.Equal(t, actorIRI, result)
}

// TestResolveActorIRIFromWebFinger_AtPrefixStripped verifies that a leading @
// is stripped from the username before the request is built.
func TestResolveActorIRIFromWebFinger_AtPrefixStripped(t *testing.T) {
	ctx := context.Background()

	mux := http.NewServeMux()
	server := httptest.NewTLSServer(mux)
	t.Cleanup(server.Close)
	aptest.UseTestTLSTransport(t, server)

	serverHost, _ := url.Parse(server.URL)
	actorIRI := server.URL + "/users/atuser"

	mux.HandleFunc("/.well-known/webfinger", aptest.WebFingerHandler(actorIRI))

	result, err := ResolveActorIRIFromWebFinger(ctx, "@atuser", serverHost.Host)
	require.NoError(t, err)
	assert.Equal(t, actorIRI, result)
}

// TestResolveActorIRIFromWebFinger_NoActivityPubLink verifies that an error is
// returned when the WebFinger response contains no ActivityPub self link.
func TestResolveActorIRIFromWebFinger_NoActivityPubLink(t *testing.T) {
	ctx := context.Background()

	mux := http.NewServeMux()
	server := httptest.NewTLSServer(mux)
	t.Cleanup(server.Close)
	aptest.UseTestTLSTransport(t, server)

	serverHost, _ := url.Parse(server.URL)

	mux.HandleFunc("/.well-known/webfinger", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/jrd+json")
		_, _ = w.Write([]byte(`{"links":[{"rel":"http://webfinger.net/rel/profile-page","href":"https://example.com/users/nobody"}]}`))
	})

	_, err := ResolveActorIRIFromWebFinger(ctx, "nobody", serverHost.Host)
	require.Error(t, err)
}
