package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/aputil"
	"github.com/AepyornisNet/aepyornis/pkg/config"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
	"github.com/samber/do/v2"
)

type ActivityPubActorService interface {
	ActorURL(profile *model.Profile) (string, error)
	MainKeyID(profile *model.Profile) (string, error)
	SendActivity(ctx context.Context, profile *model.Profile, inbox string, payload []byte) error
	SendFollowAccept(ctx context.Context, profile *model.Profile, follower model.Follower) error
	SendFollow(ctx context.Context, profile *model.Profile, inbox, targetActorIRI string) error
	SendUndoFollow(ctx context.Context, profile *model.Profile, inbox, targetActorIRI string) error
	SendLike(ctx context.Context, profile *model.Profile, inbox, objectIRI string) error
}

type activityPubActorService struct {
	cfg        *config.Config
	requestSvc ActivityPubRequestService
}

func NewActivityPubActorService(injector do.Injector) (ActivityPubActorService, error) {
	return &activityPubActorService{
		cfg:        do.MustInvoke[*config.Config](injector),
		requestSvc: do.MustInvoke[ActivityPubRequestService](injector),
	}, nil
}

func (s *activityPubActorService) ActorURL(profile *model.Profile) (string, error) {
	if profile == nil {
		return "", errors.New("profile is nil")
	}

	if actorURL := strings.TrimSpace(profile.ActorURL()); actorURL != "" {
		return actorURL, nil
	}

	username := strings.TrimSpace(profile.Username)
	if username == "" {
		return "", errors.New("profile username is empty")
	}

	if profile.Domain != nil && strings.TrimSpace(*profile.Domain) != "" {
		return fmt.Sprintf("https://%s/ap/users/%s", strings.TrimSpace(*profile.Domain), username), nil
	}

	if s.cfg == nil || strings.TrimSpace(s.cfg.Host) == "" {
		return "", errors.New("local host is not configured")
	}

	return aputil.LocalActorURL(aputil.LocalActorURLConfig{
		Host:           s.cfg.Host,
		WebRoot:        s.cfg.WebRoot,
		FallbackScheme: "https",
	}, username), nil
}

func (s *activityPubActorService) MainKeyID(profile *model.Profile) (string, error) {
	actorURL, err := s.ActorURL(profile)
	if err != nil {
		return "", err
	}

	return actorURL + "#main-key", nil
}

func (s *activityPubActorService) SendActivity(ctx context.Context, profile *model.Profile, inbox string, payload []byte) error {
	if profile == nil {
		return errors.New("profile is nil")
	}

	privateKey := strings.TrimSpace(profile.PrivateKey)
	if privateKey == "" {
		return errors.New("profile private key is empty")
	}

	keyID, err := s.MainKeyID(profile)
	if err != nil {
		return err
	}

	return s.requestSvc.SendSignedActivity(ctx, keyID, privateKey, inbox, payload)
}

func (s *activityPubActorService) SendFollowAccept(ctx context.Context, profile *model.Profile, follower model.Follower) error {
	if follower.Profile == nil {
		return errors.New("follower profile is empty")
	}

	if follower.Profile.UserID != nil || follower.Profile.Local {
		return nil
	}

	if follower.Profile.InboxURL == nil || strings.TrimSpace(*follower.Profile.InboxURL) == "" {
		return errors.New("follower inbox is empty")
	}

	actorURL, err := s.ActorURL(profile)
	if err != nil {
		return err
	}

	follow := vocab.Activity{
		Type:   vocab.FollowType,
		Actor:  vocab.IRI(follower.Profile.ActorURL()),
		Object: vocab.IRI(actorURL),
	}

	accept := vocab.Activity{
		ID:     vocab.ID(fmt.Sprintf("%s#accept-follow-%d", actorURL, follower.ID)),
		Type:   vocab.AcceptType,
		Actor:  vocab.IRI(actorURL),
		Object: follow,
	}

	payload, err := marshalActivity(accept)
	if err != nil {
		return err
	}

	return s.SendActivity(ctx, profile, *follower.Profile.InboxURL, payload)
}

func (s *activityPubActorService) SendFollow(ctx context.Context, profile *model.Profile, inbox, targetActorIRI string) error {
	actorURL, err := s.ActorURL(profile)
	if err != nil {
		return err
	}

	follow := vocab.Activity{
		ID:     vocab.ID(fmt.Sprintf("%s#follow-%d", actorURL, time.Now().UTC().UnixNano())),
		Type:   vocab.FollowType,
		Actor:  vocab.IRI(actorURL),
		Object: vocab.IRI(targetActorIRI),
	}

	payload, err := marshalActivity(follow)
	if err != nil {
		return err
	}

	return s.SendActivity(ctx, profile, inbox, payload)
}

func (s *activityPubActorService) SendUndoFollow(ctx context.Context, profile *model.Profile, inbox, targetActorIRI string) error {
	actorURL, err := s.ActorURL(profile)
	if err != nil {
		return err
	}

	follow := vocab.Activity{
		Type:   vocab.FollowType,
		Actor:  vocab.IRI(actorURL),
		Object: vocab.IRI(targetActorIRI),
	}

	undo := vocab.Activity{
		ID:     vocab.ID(fmt.Sprintf("%s#undo-follow-%d", actorURL, time.Now().UTC().UnixNano())),
		Type:   vocab.UndoType,
		Actor:  vocab.IRI(actorURL),
		Object: follow,
	}

	payload, err := marshalActivity(undo)
	if err != nil {
		return err
	}

	return s.SendActivity(ctx, profile, inbox, payload)
}

func (s *activityPubActorService) SendLike(ctx context.Context, profile *model.Profile, inbox, objectIRI string) error {
	actorURL, err := s.ActorURL(profile)
	if err != nil {
		return err
	}

	like := vocab.Activity{
		ID:     vocab.ID(fmt.Sprintf("%s#like-%d", actorURL, time.Now().UTC().UnixNano())),
		Type:   vocab.LikeType,
		Actor:  vocab.IRI(actorURL),
		Object: vocab.IRI(objectIRI),
	}

	payload, err := marshalActivity(like)
	if err != nil {
		return err
	}

	return s.SendActivity(ctx, profile, inbox, payload)
}

func marshalActivity(activity vocab.Activity) ([]byte, error) {
	return jsonld.WithContext(
		jsonld.IRI(vocab.ActivityBaseURI),
	).Marshal(activity)
}
