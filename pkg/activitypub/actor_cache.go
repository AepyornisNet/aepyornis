package activitypub

import (
	"context"
	"strings"
	"sync"
	"time"

	vocab "github.com/go-ap/activitypub"
)

type cachedActorProfile struct {
	Name      string
	AvatarURL string
	ExpiresAt time.Time
}

var actorProfileCache = struct {
	sync.RWMutex
	items map[string]cachedActorProfile
}{
	items: map[string]cachedActorProfile{},
}

const actorProfileCacheTTL = 24 * time.Hour

func CacheActorProfile(actorIRI, name, avatarURL string) {
	actorIRI = strings.TrimSpace(actorIRI)
	if actorIRI == "" {
		return
	}

	entry := cachedActorProfile{
		Name:      strings.TrimSpace(name),
		AvatarURL: strings.TrimSpace(avatarURL),
		ExpiresAt: time.Now().UTC().Add(actorProfileCacheTTL),
	}

	actorProfileCache.Lock()
	actorProfileCache.items[actorIRI] = entry
	actorProfileCache.Unlock()
}

func GetCachedActorProfile(actorIRI string) (name, avatarURL string, ok bool) {
	actorIRI = strings.TrimSpace(actorIRI)
	if actorIRI == "" {
		return "", "", false
	}

	actorProfileCache.RLock()
	entry, exists := actorProfileCache.items[actorIRI]
	actorProfileCache.RUnlock()
	if !exists {
		return "", "", false
	}

	if time.Now().UTC().After(entry.ExpiresAt) {
		actorProfileCache.Lock()
		delete(actorProfileCache.items, actorIRI)
		actorProfileCache.Unlock()
		return "", "", false
	}

	return entry.Name, entry.AvatarURL, true
}

func ResolveAndCacheActorProfile(ctx context.Context, actorIRI string) (name, avatarURL string, ok bool) {
	if cachedName, cachedAvatarURL, found := GetCachedActorProfile(actorIRI); found {
		return cachedName, cachedAvatarURL, true
	}

	actor, err := LoadRemoteActor(ctx, actorIRI)
	if err != nil || actor == nil {
		return "", "", false
	}

	resolvedName := ""
	if actor.Name != nil {
		resolvedName = strings.TrimSpace(actor.Name.String())
	}

	resolvedAvatar := ActorIconIRI(actor)
	CacheActorProfile(actorIRI, resolvedName, resolvedAvatar)

	return resolvedName, resolvedAvatar, true
}

func ActorIconIRI(actor *vocab.Actor) string {
	if actor == nil || vocab.IsNil(actor.Icon) {
		return ""
	}

	if vocab.IsIRI(actor.Icon) {
		return actor.Icon.GetLink().String()
	}

	iconIRI := ""
	_ = vocab.OnLink(actor.Icon, func(link *vocab.Link) error {
		iconIRI = link.Href.String()
		return nil
	})
	if iconIRI != "" {
		return iconIRI
	}

	_ = vocab.OnObject(actor.Icon, func(obj *vocab.Object) error {
		if !vocab.IsNil(obj.URL) {
			iconIRI = itemIRIString(obj.URL)
			if iconIRI != "" {
				return nil
			}
		}

		if obj.ID != "" {
			iconIRI = obj.ID.String()
		}

		return nil
	})

	return iconIRI
}
