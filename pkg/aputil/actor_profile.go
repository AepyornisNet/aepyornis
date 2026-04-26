package aputil

import (
	"errors"
	"net/url"
	"strings"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	vocab "github.com/go-ap/activitypub"
)

func RemoteProfileFromActor(actor *vocab.Actor) (*model.Profile, error) {
	if actor == nil {
		return nil, errors.New("remote actor is nil")
	}

	actorURL, username, domain, err := actorIdentity(actor)
	if err != nil {
		return nil, err
	}
	profile := &model.Profile{
		Local:       false,
		Username:    username,
		DisplayName: actorDisplayName(actor, username),
		URL:         &actorURL,
	}
	if domain != "" {
		profile.Domain = &domain
	}
	assignActorEndpoints(profile, actor)

	return profile, nil
}

func actorIdentity(actor *vocab.Actor) (string, string, string, error) {
	actorURL := strings.TrimSpace(actor.ID.String())
	if actorURL == "" {
		return "", "", "", errors.New("remote actor id is empty")
	}

	username := actorPreferredUsername(actor)
	domain := ""
	if parsed, err := url.Parse(actorURL); err == nil && parsed.Host != "" {
		domain = parsed.Host
		if username == "" {
			segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
			if len(segments) > 0 {
				username = strings.TrimSpace(segments[len(segments)-1])
			}
		}
	}
	if username == "" {
		return "", "", "", errors.New("remote actor username is empty")
	}

	return actorURL, username, domain, nil
}

func actorPreferredUsername(actor *vocab.Actor) string {
	if actor == nil || actor.PreferredUsername == nil {
		return ""
	}

	return strings.TrimSpace(actor.PreferredUsername.String())
}

func actorDisplayName(actor *vocab.Actor, username string) string {
	if actor != nil && actor.Name != nil && strings.TrimSpace(actor.Name.String()) != "" {
		return strings.TrimSpace(actor.Name.String())
	}

	return username
}

func assignActorEndpoints(profile *model.Profile, actor *vocab.Actor) {
	if profile == nil || actor == nil {
		return
	}

	assignOptionalActorString(&profile.InboxURL, actorItemString(actor.Inbox))
	assignOptionalActorString(&profile.OutboxURL, actorItemString(actor.Outbox))
	assignOptionalActorString(&profile.FollowersURL, actorItemString(actor.Followers))
	assignOptionalActorString(&profile.AvatarRemoteURL, actorIconURL(actor))
}

func assignOptionalActorString(dst **string, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}

	*dst = &trimmed
}

func actorItemString(item vocab.Item) string {
	if vocab.IsNil(item) {
		return ""
	}
	if vocab.IsIRI(item) {
		return item.GetLink().String()
	}

	iri := ""
	_ = vocab.OnLink(item, func(link *vocab.Link) error {
		iri = link.Href.String()
		return nil
	})

	return iri
}

func actorIconURL(actor *vocab.Actor) string {
	if actor == nil || vocab.IsNil(actor.Icon) {
		return ""
	}
	if vocab.IsIRI(actor.Icon) {
		return actor.Icon.GetLink().String()
	}

	iconURL := actorItemString(actor.Icon)
	if iconURL != "" {
		return iconURL
	}

	_ = vocab.OnObject(actor.Icon, func(object *vocab.Object) error {
		if object != nil && !vocab.IsNil(object.URL) {
			iconURL = actorItemString(object.URL)
		}
		return nil
	})

	return iconURL
}
