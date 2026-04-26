package service

import (
	"context"
	"errors"
	"strings"

	"github.com/AepyornisNet/aepyornis/pkg/aputil"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/samber/do/v2"
	"gorm.io/gorm"
)

type ActivityPubProfileService interface {
	GetByActorIRI(ctx context.Context, actorIRI string) (*model.Profile, error)
}

type activityPubProfileService struct {
	db *gorm.DB
}

func NewActivityPubProfileService(injector do.Injector) (ActivityPubProfileService, error) {
	return &activityPubProfileService{
		db: do.MustInvoke[*gorm.DB](injector),
	}, nil
}

func (s *activityPubProfileService) GetByActorIRI(ctx context.Context, actorIRI string) (*model.Profile, error) {
	actorIRI = strings.TrimSpace(actorIRI)
	if actorIRI == "" {
		return nil, errors.New("actor iri is empty")
	}

	existing, err := s.findByActorIRI(actorIRI)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	actor, err := aputil.LoadRemoteActor(ctx, actorIRI)
	if err != nil {
		return nil, err
	}

	profile, err := aputil.RemoteProfileFromActor(actor)
	if err != nil {
		return nil, err
	}

	saved, err := profile.UpsertRemote(s.db)
	if err != nil {
		return nil, err
	}

	return saved, nil
}

func (s *activityPubProfileService) findByActorIRI(actorIRI string) (*model.Profile, error) {
	profile := &model.Profile{}
	if err := s.db.Where("url = ?", actorIRI).First(profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return profile, nil
}
