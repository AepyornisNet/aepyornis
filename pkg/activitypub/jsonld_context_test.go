package activitypub

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkoutJSONLDContext_ContainsBaseURI(t *testing.T) {
	ctx := WorkoutJSONLDContext()
	require.NotEmpty(t, ctx)

	// First element must map to the ActivityStreams base URI.
	found := false
	for _, elem := range ctx {
		if string(elem.IRI) == "https://www.w3.org/ns/activitystreams" {
			found = true
			break
		}
	}
	assert.True(t, found, "context should include the ActivityStreams base URI")
}

func TestWorkoutJSONLDContext_ContainsAEPYNamespace(t *testing.T) {
	ctx := WorkoutJSONLDContext()

	found := false
	for _, elem := range ctx {
		if string(elem.IRI) == AEPYNamespaceURL {
			found = true
			break
		}
	}
	assert.True(t, found, "context should include the AEPY namespace")
}

func TestWorkoutJSONLDContext_ContainsWorkoutTerms(t *testing.T) {
	ctx := WorkoutJSONLDContext()

	// Build a set of all registered terms.
	terms := make(map[string]bool, len(ctx))
	for _, elem := range ctx {
		terms[string(elem.Term)] = true
	}

	for _, expectedTerm := range workoutExtensionTerms {
		assert.True(t, terms[expectedTerm], "context should include term %q", expectedTerm)
	}
}
