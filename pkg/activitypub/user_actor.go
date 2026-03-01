package activitypub

import (
	"context"
	"errors"
	"fmt"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
)

type UserActor struct {
	ActorURL      string
	PrivateKeyPEM string
}

func NewUserActor(actorURL, privateKeyPEM string) *UserActor {
	return &UserActor{ActorURL: actorURL, PrivateKeyPEM: privateKeyPEM}
}

func (u *UserActor) SendFollowAccept(ctx context.Context, follower model.Follower) error {
	if follower.ActorInbox == "" {
		return errors.New("follower inbox is empty")
	}

	follow := vocab.Activity{
		Type:   vocab.FollowType,
		Actor:  vocab.IRI(follower.ActorIRI),
		Object: vocab.IRI(u.ActorURL),
	}

	accept := vocab.Activity{
		ID:     vocab.ID(fmt.Sprintf("%s#accept-follow-%d", u.ActorURL, follower.ID)),
		Type:   vocab.AcceptType,
		Actor:  vocab.IRI(u.ActorURL),
		Object: follow,
	}

	payload, err := jsonld.WithContext(
		jsonld.IRI(vocab.ActivityBaseURI),
	).Marshal(accept)
	if err != nil {
		return err
	}

	return SendSignedActivity(ctx, u.ActorURL, u.PrivateKeyPEM, follower.ActorInbox, payload)
}

func (u *UserActor) SendFollow(ctx context.Context, inbox, targetActorIRI string) error {
	follow := vocab.Activity{
		ID:     vocab.ID(fmt.Sprintf("%s#follow-%d", u.ActorURL, time.Now().UTC().UnixNano())),
		Type:   vocab.FollowType,
		Actor:  vocab.IRI(u.ActorURL),
		Object: vocab.IRI(targetActorIRI),
	}

	payload, err := jsonld.WithContext(
		jsonld.IRI(vocab.ActivityBaseURI),
	).Marshal(follow)
	if err != nil {
		return err
	}

	return SendSignedActivity(ctx, u.ActorURL, u.PrivateKeyPEM, inbox, payload)
}

func (u *UserActor) SendUndoFollow(ctx context.Context, inbox, targetActorIRI string) error {
	follow := vocab.Activity{
		Type:   vocab.FollowType,
		Actor:  vocab.IRI(u.ActorURL),
		Object: vocab.IRI(targetActorIRI),
	}

	undo := vocab.Activity{
		ID:     vocab.ID(fmt.Sprintf("%s#undo-follow-%d", u.ActorURL, time.Now().UTC().UnixNano())),
		Type:   vocab.UndoType,
		Actor:  vocab.IRI(u.ActorURL),
		Object: follow,
	}

	payload, err := jsonld.WithContext(
		jsonld.IRI(vocab.ActivityBaseURI),
	).Marshal(undo)
	if err != nil {
		return err
	}

	return SendSignedActivity(ctx, u.ActorURL, u.PrivateKeyPEM, inbox, payload)
}

func (u *UserActor) SendLike(ctx context.Context, inbox, objectIRI string) error {
	like := vocab.Activity{
		ID:     vocab.ID(fmt.Sprintf("%s#like-%d", u.ActorURL, time.Now().UTC().UnixNano())),
		Type:   vocab.LikeType,
		Actor:  vocab.IRI(u.ActorURL),
		Object: vocab.IRI(objectIRI),
	}

	payload, err := jsonld.WithContext(
		jsonld.IRI(vocab.ActivityBaseURI),
	).Marshal(like)
	if err != nil {
		return err
	}

	return SendSignedActivity(ctx, u.ActorURL, u.PrivateKeyPEM, inbox, payload)
}
