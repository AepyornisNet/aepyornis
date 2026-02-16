package model

import (
	"time"

	"gorm.io/gorm"
)

type Follower struct {
	Model

	UserID uint64 `gorm:"index:idx_followers_user_id;not null" json:"user_id"`
	User   *User  `json:"-"`

	ActorIRI   string     `gorm:"type:text;index:idx_followers_user_actor,unique;not null" json:"actor_iri"`
	ActorInbox string     `gorm:"type:text" json:"actor_inbox"`
	Approved   bool       `gorm:"default:false;index" json:"approved"`
	ApprovedAt *time.Time `json:"approved_at,omitempty"`
}

func UpsertFollowerRequest(db *gorm.DB, userID uint64, actorIRI, actorInbox string) (*Follower, error) {
	f := &Follower{}
	if err := db.Where(&Follower{UserID: userID, ActorIRI: actorIRI}).First(f).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			f.UserID = userID
			f.ActorIRI = actorIRI
			f.ActorInbox = actorInbox
			f.Approved = false
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
	if err := db.Save(f).Error; err != nil {
		return nil, err
	}

	return f, nil
}

func ListFollowerRequests(db *gorm.DB, userID uint64) ([]Follower, error) {
	followers := make([]Follower, 0)
	if err := db.Where("user_id = ? AND approved = ?", userID, false).Order("created_at DESC").Find(&followers).Error; err != nil {
		return nil, err
	}
	return followers, nil
}

func ListApprovedFollowers(db *gorm.DB, userID uint64) ([]Follower, error) {
	followers := make([]Follower, 0)
	if err := db.Where("user_id = ? AND approved = ?", userID, true).Order("created_at DESC").Find(&followers).Error; err != nil {
		return nil, err
	}
	return followers, nil
}

func ApproveFollowerRequest(db *gorm.DB, userID uint64, requestID uint64) (*Follower, error) {
	f := &Follower{}
	if err := db.Where("id = ? AND user_id = ?", requestID, userID).First(f).Error; err != nil {
		return nil, err
	}

	n := time.Now().UTC()
	f.Approved = true
	f.ApprovedAt = &n

	if err := db.Save(f).Error; err != nil {
		return nil, err
	}

	return f, nil
}
