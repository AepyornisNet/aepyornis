package model

import (
	"time"

	"gorm.io/gorm"
)

type FollowerDirection string

const (
	FollowerDirectionIncoming FollowerDirection = "incoming"
	FollowerDirectionOutgoing FollowerDirection = "outgoing"
)

type Follower struct {
	Model

	UserID uint64 `gorm:"index:idx_followers_user_id;uniqueIndex:idx_followers_user_actor_direction;not null" json:"user_id"`
	User   *User  `json:"-"`

	ActorIRI   string            `gorm:"type:text;uniqueIndex:idx_followers_user_actor_direction;not null" json:"actor_iri"`
	ActorInbox string            `gorm:"type:text" json:"actor_inbox"`
	Direction  FollowerDirection `gorm:"type:varchar(16);uniqueIndex:idx_followers_user_actor_direction;not null;default:incoming;index" json:"direction"`
	Approved   bool              `gorm:"default:false;index" json:"approved"`
	ApprovedAt *time.Time        `json:"approved_at,omitempty"`
	RejectedAt *time.Time        `json:"rejected_at,omitempty"`
}

func UpsertFollowerRequest(db *gorm.DB, userID uint64, actorIRI, actorInbox string) (*Follower, error) {
	return upsertFollowEntry(db, userID, actorIRI, actorInbox, FollowerDirectionIncoming)
}

func UpsertFollowingRequest(db *gorm.DB, userID uint64, actorIRI, actorInbox string) (*Follower, error) {
	return upsertFollowEntry(db, userID, actorIRI, actorInbox, FollowerDirectionOutgoing)
}

func upsertFollowEntry(db *gorm.DB, userID uint64, actorIRI, actorInbox string, direction FollowerDirection) (*Follower, error) {
	f := &Follower{}
	if err := db.Where(&Follower{UserID: userID, ActorIRI: actorIRI, Direction: direction}).First(f).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			f.UserID = userID
			f.ActorIRI = actorIRI
			f.ActorInbox = actorInbox
			f.Direction = direction
			f.Approved = false
			f.RejectedAt = nil
			if err := db.Create(f).Error; err != nil {
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
	if err := db.Save(f).Error; err != nil {
		return nil, err
	}

	return f, nil
}

func ListFollowerRequests(db *gorm.DB, userID uint64) ([]Follower, error) {
	followers := make([]Follower, 0)
	if err := db.Where("user_id = ? AND direction = ? AND approved = ?", userID, FollowerDirectionIncoming, false).Order("created_at DESC").Find(&followers).Error; err != nil {
		return nil, err
	}
	return followers, nil
}

func ListApprovedFollowers(db *gorm.DB, userID uint64) ([]Follower, error) {
	followers := make([]Follower, 0)
	if err := db.Where("user_id = ? AND direction = ? AND approved = ?", userID, FollowerDirectionIncoming, true).Order("created_at DESC").Find(&followers).Error; err != nil {
		return nil, err
	}
	return followers, nil
}

func ApproveFollowerRequest(db *gorm.DB, userID uint64, requestID uint64) (*Follower, error) {
	f := &Follower{}
	if err := db.Where("id = ? AND user_id = ? AND direction = ?", requestID, userID, FollowerDirectionIncoming).First(f).Error; err != nil {
		return nil, err
	}

	n := time.Now().UTC()
	f.Approved = true
	f.ApprovedAt = &n
	f.RejectedAt = nil

	if err := db.Save(f).Error; err != nil {
		return nil, err
	}

	return f, nil
}

func DeleteFollowerByActorIRI(db *gorm.DB, userID uint64, actorIRI string) error {
	return db.Where("user_id = ? AND actor_iri = ? AND direction = ?", userID, actorIRI, FollowerDirectionIncoming).Delete(&Follower{}).Error
}

func DeleteFollowingByActorIRI(db *gorm.DB, userID uint64, actorIRI string) error {
	return db.Where("user_id = ? AND actor_iri = ? AND direction = ?", userID, actorIRI, FollowerDirectionOutgoing).Delete(&Follower{}).Error
}

func CountApprovedFollowers(db *gorm.DB, userID uint64) (int64, error) {
	var count int64
	if err := db.Model(&Follower{}).Where("user_id = ? AND direction = ? AND approved = ?", userID, FollowerDirectionIncoming, true).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func CountApprovedFollowingByActorIRI(db *gorm.DB, actorIRI string) (int64, error) {
	var count int64
	if err := db.Model(&Follower{}).Where("actor_iri = ? AND direction = ? AND approved = ?", actorIRI, FollowerDirectionIncoming, true).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func IsActorFollowingUser(db *gorm.DB, userID uint64, actorIRI string) (bool, error) {
	if actorIRI == "" {
		return false, nil
	}

	var count int64
	if err := db.Model(&Follower{}).Where("user_id = ? AND actor_iri = ? AND direction = ? AND approved = ?", userID, actorIRI, FollowerDirectionIncoming, true).Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func IsFollowingApprovedByActorIRI(db *gorm.DB, userID uint64, actorIRI string) (bool, error) {
	var count int64
	if err := db.Model(&Follower{}).
		Where("user_id = ? AND actor_iri = ? AND direction = ? AND approved = ?", userID, actorIRI, FollowerDirectionOutgoing, true).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func IsFollowingActiveByActorIRI(db *gorm.DB, userID uint64, actorIRI string) (bool, error) {
	var count int64
	if err := db.Model(&Follower{}).
		Where("user_id = ? AND actor_iri = ? AND direction = ? AND rejected_at IS NULL", userID, actorIRI, FollowerDirectionOutgoing).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func MarkFollowingApprovedByActorIRI(db *gorm.DB, userID uint64, actorIRI string) (*Follower, error) {
	f := &Follower{}
	if err := db.Where(&Follower{UserID: userID, ActorIRI: actorIRI, Direction: FollowerDirectionOutgoing}).First(f).Error; err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	f.Approved = true
	f.ApprovedAt = &now
	f.RejectedAt = nil

	if err := db.Save(f).Error; err != nil {
		return nil, err
	}

	return f, nil
}

func MarkFollowingRejectedByActorIRI(db *gorm.DB, userID uint64, actorIRI string) (*Follower, error) {
	f := &Follower{}
	if err := db.Where(&Follower{UserID: userID, ActorIRI: actorIRI, Direction: FollowerDirectionOutgoing}).First(f).Error; err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	f.Approved = false
	f.ApprovedAt = nil
	f.RejectedAt = &now

	if err := db.Save(f).Error; err != nil {
		return nil, err
	}

	return f, nil
}
