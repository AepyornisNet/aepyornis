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

	actorURL := strings.TrimSpace(actor.ID.String())
	if actorURL == "" {
		return nil, errors.New("remote actor id is empty")
	}

	username := ""
	if actor.PreferredUsername != nil && strings.TrimSpace(actor.PreferredUsername.String()) != "" {
		username = strings.TrimSpace(actor.PreferredUsername.String())
	}

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
		return nil, errors.New("remote actor username is empty")
	}

	displayName := username
	if actor.Name != nil && strings.TrimSpace(actor.Name.String()) != "" {
		displayName = strings.TrimSpace(actor.Name.String())
	}

	profile := &model.Profile{
		Local:       false,
		Username:    username,
		DisplayName: displayName,
		URL:         &actorURL,
	}
	if domain != "" {
		profile.Domain = &domain
	}
	if inbox := strings.TrimSpace(actorItemString(actor.Inbox)); inbox != "" {
		profile.InboxURL = &inbox
	}
	if outbox := strings.TrimSpace(actorItemString(actor.Outbox)); outbox != "" {
		profile.OutboxURL = &outbox
	}
	if followers := strings.TrimSpace(actorItemString(actor.Followers)); followers != "" {
		profile.FollowersURL = &followers
	}
	if avatar := strings.TrimSpace(actorIconURL(actor)); avatar != "" {
		profile.AvatarRemoteURL = &avatar
	}

	return profile, nil
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
