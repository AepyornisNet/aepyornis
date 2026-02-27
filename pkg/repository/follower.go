package repository

import (
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"gorm.io/gorm"
)

type Follower interface {
	UpsertFollowerRequest(userID uint64, actorIRI, actorInbox string) (*model.Follower, error)
	UpsertFollowingRequest(userID uint64, actorIRI, actorInbox string) (*model.Follower, error)
	ListFollowerRequests(userID uint64) ([]model.Follower, error)
	ListApprovedFollowers(userID uint64) ([]model.Follower, error)
	ListApprovedFollowing(userID uint64) ([]model.Follower, error)
	ApproveFollowerRequest(userID uint64, requestID uint64) (*model.Follower, error)
	DeleteFollowerByActorIRI(userID uint64, actorIRI string) error
	DeleteFollowingByActorIRI(userID uint64, actorIRI string) error
	CountApprovedFollowers(userID uint64) (int64, error)
	CountApprovedFollowingByActorIRI(actorIRI string) (int64, error)
	IsActorFollowingUser(userID uint64, actorIRI string) (bool, error)
	IsFollowingApprovedByActorIRI(userID uint64, actorIRI string) (bool, error)
	IsFollowingActiveByActorIRI(userID uint64, actorIRI string) (bool, error)
	MarkFollowingApprovedByActorIRI(userID uint64, actorIRI string) (*model.Follower, error)
	MarkFollowingRejectedByActorIRI(userID uint64, actorIRI string) (*model.Follower, error)
}

type followerRepository struct {
	db *gorm.DB
}

func NewFollower(db *gorm.DB) Follower {
	return &followerRepository{db: db}
}

func (r *followerRepository) UpsertFollowerRequest(userID uint64, actorIRI, actorInbox string) (*model.Follower, error) {
	return r.upsertFollowEntry(userID, actorIRI, actorInbox, model.FollowerDirectionIncoming)
}

func (r *followerRepository) UpsertFollowingRequest(userID uint64, actorIRI, actorInbox string) (*model.Follower, error) {
	return r.upsertFollowEntry(userID, actorIRI, actorInbox, model.FollowerDirectionOutgoing)
}

func (r *followerRepository) ListFollowerRequests(userID uint64) ([]model.Follower, error) {
	followers := make([]model.Follower, 0)
	if err := r.db.Where("user_id = ? AND direction = ? AND approved = ?", userID, model.FollowerDirectionIncoming, false).Order("created_at DESC").Find(&followers).Error; err != nil {
		return nil, err
	}
	return followers, nil
}

func (r *followerRepository) ListApprovedFollowers(userID uint64) ([]model.Follower, error) {
	followers := make([]model.Follower, 0)
	if err := r.db.Where("user_id = ? AND direction = ? AND approved = ?", userID, model.FollowerDirectionIncoming, true).Order("created_at DESC").Find(&followers).Error; err != nil {
		return nil, err
	}
	return followers, nil
}

func (r *followerRepository) ListApprovedFollowing(userID uint64) ([]model.Follower, error) {
	following := make([]model.Follower, 0)
	if err := r.db.Where("user_id = ? AND direction = ? AND approved = ?", userID, model.FollowerDirectionOutgoing, true).Order("created_at DESC").Find(&following).Error; err != nil {
		return nil, err
	}
	return following, nil
}

func (r *followerRepository) ApproveFollowerRequest(userID uint64, requestID uint64) (*model.Follower, error) {
	f := &model.Follower{}
	if err := r.db.Where("id = ? AND user_id = ? AND direction = ?", requestID, userID, model.FollowerDirectionIncoming).First(f).Error; err != nil {
		return nil, err
	}

	n := time.Now().UTC()
	f.Approved = true
	f.ApprovedAt = &n
	f.RejectedAt = nil

	if err := r.db.Save(f).Error; err != nil {
		return nil, err
	}

	return f, nil
}

func (r *followerRepository) DeleteFollowerByActorIRI(userID uint64, actorIRI string) error {
	return r.db.Where("user_id = ? AND actor_iri = ? AND direction = ?", userID, actorIRI, model.FollowerDirectionIncoming).Delete(&model.Follower{}).Error
}

func (r *followerRepository) DeleteFollowingByActorIRI(userID uint64, actorIRI string) error {
	return r.db.Where("user_id = ? AND actor_iri = ? AND direction = ?", userID, actorIRI, model.FollowerDirectionOutgoing).Delete(&model.Follower{}).Error
}

func (r *followerRepository) CountApprovedFollowers(userID uint64) (int64, error) {
	var count int64
	if err := r.db.Model(&model.Follower{}).Where("user_id = ? AND direction = ? AND approved = ?", userID, model.FollowerDirectionIncoming, true).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *followerRepository) CountApprovedFollowingByActorIRI(actorIRI string) (int64, error) {
	var count int64
	if err := r.db.Model(&model.Follower{}).Where("actor_iri = ? AND direction = ? AND approved = ?", actorIRI, model.FollowerDirectionIncoming, true).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *followerRepository) IsActorFollowingUser(userID uint64, actorIRI string) (bool, error) {
	if actorIRI == "" {
		return false, nil
	}

	var count int64
	if err := r.db.Model(&model.Follower{}).Where("user_id = ? AND actor_iri = ? AND direction = ? AND approved = ?", userID, actorIRI, model.FollowerDirectionIncoming, true).Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *followerRepository) IsFollowingApprovedByActorIRI(userID uint64, actorIRI string) (bool, error) {
	var count int64
	if err := r.db.Model(&model.Follower{}).
		Where("user_id = ? AND actor_iri = ? AND direction = ? AND approved = ?", userID, actorIRI, model.FollowerDirectionOutgoing, true).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *followerRepository) IsFollowingActiveByActorIRI(userID uint64, actorIRI string) (bool, error) {
	var count int64
	if err := r.db.Model(&model.Follower{}).
		Where("user_id = ? AND actor_iri = ? AND direction = ? AND rejected_at IS NULL", userID, actorIRI, model.FollowerDirectionOutgoing).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *followerRepository) MarkFollowingApprovedByActorIRI(userID uint64, actorIRI string) (*model.Follower, error) {
	f := &model.Follower{}
	if err := r.db.Where(&model.Follower{UserID: userID, ActorIRI: actorIRI, Direction: model.FollowerDirectionOutgoing}).First(f).Error; err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	f.Approved = true
	f.ApprovedAt = &now
	f.RejectedAt = nil

	if err := r.db.Save(f).Error; err != nil {
		return nil, err
	}

	return f, nil
}

func (r *followerRepository) MarkFollowingRejectedByActorIRI(userID uint64, actorIRI string) (*model.Follower, error) {
	f := &model.Follower{}
	if err := r.db.Where(&model.Follower{UserID: userID, ActorIRI: actorIRI, Direction: model.FollowerDirectionOutgoing}).First(f).Error; err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	f.Approved = false
	f.ApprovedAt = nil
	f.RejectedAt = &now

	if err := r.db.Save(f).Error; err != nil {
		return nil, err
	}

	return f, nil
}

func (r *followerRepository) upsertFollowEntry(userID uint64, actorIRI, actorInbox string, direction model.FollowerDirection) (*model.Follower, error) {
	f := &model.Follower{}
	if err := r.db.Where(&model.Follower{UserID: userID, ActorIRI: actorIRI, Direction: direction}).First(f).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			f.UserID = userID
			f.ActorIRI = actorIRI
			f.ActorInbox = actorInbox
			f.Direction = direction
			f.Approved = false
			f.RejectedAt = nil
			if err := r.db.Create(f).Error; err != nil {
				return nil, err
			}
			return f, nil
		}

		return nil, err
	}

	f.ActorInbox = actorInbox
	if !f.Approved {
		f.ApprovedAt = nil
	}
	f.RejectedAt = nil
	if err := r.db.Save(f).Error; err != nil {
		return nil, err
	}

	return f, nil
}
