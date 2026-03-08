package activitypub

import (
	"testing"

	vocab "github.com/go-ap/activitypub"
	"github.com/stretchr/testify/assert"
)

func TestCacheAndGetActorProfile(t *testing.T) {
	iri := "https://cache-test.example.com/users/alice"
	CacheActorProfile(iri, "Alice", "https://cache-test.example.com/avatar.png")

	name, avatar, ok := GetCachedActorProfile(iri)
	assert.True(t, ok)
	assert.Equal(t, "Alice", name)
	assert.Equal(t, "https://cache-test.example.com/avatar.png", avatar)
}

func TestGetCachedActorProfile_Miss(t *testing.T) {
	_, _, ok := GetCachedActorProfile("https://cache-test.example.com/users/nobody")
	assert.False(t, ok)
}

func TestGetCachedActorProfile_EmptyIRI(t *testing.T) {
	_, _, ok := GetCachedActorProfile("")
	assert.False(t, ok)
}

func TestCacheActorProfile_EmptyIRI(t *testing.T) {
	// Must not panic and must not store anything.
	CacheActorProfile("", "Alice", "https://cache-test.example.com/avatar.png")
	_, _, ok := GetCachedActorProfile("")
	assert.False(t, ok)
}

func TestCacheActorProfile_Overwrite(t *testing.T) {
	iri := "https://cache-test.example.com/users/bob"
	CacheActorProfile(iri, "Bob v1", "https://cache-test.example.com/bob1.png")
	CacheActorProfile(iri, "Bob v2", "https://cache-test.example.com/bob2.png")

	name, avatar, ok := GetCachedActorProfile(iri)
	assert.True(t, ok)
	assert.Equal(t, "Bob v2", name)
	assert.Equal(t, "https://cache-test.example.com/bob2.png", avatar)
}

func TestActorIconIRI_NilActor(t *testing.T) {
	assert.Equal(t, "", ActorIconIRI(nil))
}

func TestActorIconIRI_NilIcon(t *testing.T) {
	actor := &vocab.Actor{}
	assert.Equal(t, "", ActorIconIRI(actor))
}

func TestActorIconIRI_IRIIcon(t *testing.T) {
	actor := &vocab.Actor{}
	actor.Icon = vocab.IRI("https://cache-test.example.com/icon.png")
	assert.Equal(t, "https://cache-test.example.com/icon.png", ActorIconIRI(actor))
}

func TestActorIconIRI_ObjectWithID(t *testing.T) {
	actor := &vocab.Actor{}
	img := vocab.ObjectNew(vocab.ImageType)
	img.ID = "https://cache-test.example.com/img-id.png"
	actor.Icon = img
	assert.Equal(t, "https://cache-test.example.com/img-id.png", ActorIconIRI(actor))
}

func TestActorIconIRI_ObjectWithURL(t *testing.T) {
	actor := &vocab.Actor{}
	img := vocab.ObjectNew(vocab.ImageType)
	img.URL = vocab.IRI("https://cache-test.example.com/img-url.png")
	actor.Icon = img
	assert.Equal(t, "https://cache-test.example.com/img-url.png", ActorIconIRI(actor))
}

func TestActorIconIRI_LinkIcon(t *testing.T) {
	actor := &vocab.Actor{}
	link := &vocab.Link{Href: "https://cache-test.example.com/link-icon.png"}
	actor.Icon = link
	assert.Equal(t, "https://cache-test.example.com/link-icon.png", ActorIconIRI(actor))
}
