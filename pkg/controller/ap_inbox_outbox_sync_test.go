package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractOrderedItems(t *testing.T) {
	body := []byte(`{
		"@context": "https://www.w3.org/ns/activitystreams",
		"type": "OrderedCollectionPage",
		"orderedItems": [
			{"type": "Create", "id": "item1"},
			{"type": "Create", "id": "item2"}
		]
	}`)

	items := extractOrderedItems(body)
	assert.Len(t, items, 2)
}

func TestExtractOrderedItems_Empty(t *testing.T) {
	body := []byte(`{
		"@context": "https://www.w3.org/ns/activitystreams",
		"type": "OrderedCollection",
		"totalItems": 5,
		"first": "https://example.com/outbox?page=1"
	}`)

	items := extractOrderedItems(body)
	assert.Nil(t, items)
}

func TestExtractFirstPageIRI(t *testing.T) {
	t.Run("string first", func(t *testing.T) {
		body := []byte(`{
			"type": "OrderedCollection",
			"first": "https://example.com/outbox?page=1"
		}`)

		iri := extractFirstPageIRI(body)
		assert.Equal(t, "https://example.com/outbox?page=1", iri)
	})

	t.Run("object first", func(t *testing.T) {
		body := []byte(`{
			"type": "OrderedCollection",
			"first": {"id": "https://example.com/outbox?page=1"}
		}`)

		iri := extractFirstPageIRI(body)
		assert.Equal(t, "https://example.com/outbox?page=1", iri)
	})

	t.Run("no first", func(t *testing.T) {
		body := []byte(`{
			"type": "OrderedCollection"
		}`)

		iri := extractFirstPageIRI(body)
		assert.Equal(t, "", iri)
	})
}
