package repository

import (
	"errors"
	"strings"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/samber/do/v2"
	"gorm.io/gorm"
)

type Follower interface {
	UpsertFollowerRequest(profileID uint64, followerProfile *model.Profile) (*model.Follower, error)
	UpsertFollowingRequest(profileID uint64, followingProfile *model.Profile) (*model.Follower, error)
	ListFollowerRequests(profileID uint64) ([]model.Follower, error)
	ListApprovedFollowers(profileID uint64) ([]model.Follower, error)
	ListApprovedFollowing(profileID uint64) ([]model.Follower, error)
	ApproveFollowerRequest(profileID uint64, requestID uint64) (*model.Follower, error)
	DeleteFollower(profileID, followingProfileID uint64) error
	DeleteFollowerByURL(profileID uint64, followingProfileURL string) error
	DeleteFollowing(profileID, followingProfileID uint64) error
	DeleteFollowingByURL(profileID uint64, followingProfileURL string) error
	CountApprovedFollowers(profileID uint64) (int64, error)
	CountApprovedFollowing(profileID uint64) (int64, error)
	IsFollowingApproved(profileID, followingProfileID uint64) (bool, error)
	IsFollowingActive(profileID, followingProfileID uint64) (bool, error)
	IsFollowingActiveByURL(profileID uint64, followingProfileURL string) (bool, error)
	MarkFollowingApprovedByURL(profileID uint64, followingProfileURL string) (*model.Follower, error)
	MarkFollowingRejectedByURL(profileID uint64, followingProfileURL string) (*model.Follower, error)
}

type followerRepository struct {
	db *gorm.DB
}

func NewFollower(injector do.Injector) (Follower, error) {
	return &followerRepository{db: do.MustInvoke[*gorm.DB](injector)}, nil
}

func (r *followerRepository) UpsertFollowerRequest(profileID uint64, followerProfile *model.Profile) (*model.Follower, error) {
	resolvedFollowerProfile, err := r.ensureRelatedProfile(followerProfile)
	if err != nil {
		return nil, err
	}

	return r.upsertFollowEntry(resolvedFollowerProfile.ID, profileID, resolvedFollowerProfile, nil)
}

func (r *followerRepository) UpsertFollowingRequest(profileID uint64, followingProfile *model.Profile) (*model.Follower, error) {
	resolvedFollowingProfile, err := r.ensureRelatedProfile(followingProfile)
	if err != nil {
		return nil, err
	}

	return r.upsertFollowEntry(profileID, resolvedFollowingProfile.ID, nil, resolvedFollowingProfile)
}

func (r *followerRepository) ListFollowerRequests(profileID uint64) ([]model.Follower, error) {
	followers := make([]model.Follower, 0)
	if err := r.db.Preload("Profile").Where("following_profile_id = ? AND approved = ?", profileID, false).Order("created_at DESC").Find(&followers).Error; err != nil {
		return nil, err
	}
	return followers, nil
}

func (r *followerRepository) ListApprovedFollowers(profileID uint64) ([]model.Follower, error) {
	followers := make([]model.Follower, 0)
	if err := r.db.Preload("Profile").Where("following_profile_id = ? AND approved = ?", profileID, true).Order("created_at DESC").Find(&followers).Error; err != nil {
		return nil, err
	}
	return followers, nil
}

func (r *followerRepository) ListApprovedFollowing(profileID uint64) ([]model.Follower, error) {
	following := make([]model.Follower, 0)
	if err := r.db.Preload("FollowingProfile").Where("profile_id = ? AND approved = ?", profileID, true).Order("created_at DESC").Find(&following).Error; err != nil {
		return nil, err
	}
	return following, nil
}

func (r *followerRepository) ApproveFollowerRequest(profileID uint64, requestID uint64) (*model.Follower, error) {
	f := &model.Follower{}
	if err := r.db.Preload("Profile").Preload("FollowingProfile").Where("id = ? AND following_profile_id = ?", requestID, profileID).First(f).Error; err != nil {
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

func (r *followerRepository) DeleteFollower(profileID, followingProfileID uint64) error {
	return r.db.Where("profile_id = ? AND following_profile_id = ?", followingProfileID, profileID).Delete(&model.Follower{}).Error
}

func (r *followerRepository) DeleteFollowerByURL(profileID uint64, followingProfileURL string) error {
	followingProfile, err := r.profileByURL(followingProfileURL)
	if err != nil {
		return err
	}
	return r.DeleteFollower(profileID, followingProfile.ID)
}

func (r *followerRepository) DeleteFollowing(profileID, followingProfileID uint64) error {
	return r.db.Where("profile_id = ? AND following_profile_id = ?", profileID, followingProfileID).Delete(&model.Follower{}).Error
}

func (r *followerRepository) DeleteFollowingByURL(profileID uint64, followingProfileURL string) error {
	followingProfile, err := r.profileByURL(followingProfileURL)
	if err != nil {
		return err
	}
	return r.DeleteFollowing(profileID, followingProfile.ID)
}

func (r *followerRepository) CountApprovedFollowers(profileID uint64) (int64, error) {
	var count int64
	if err := r.db.Model(&model.Follower{}).Where("following_profile_id = ? AND approved = ?", profileID, true).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *followerRepository) CountApprovedFollowing(profileID uint64) (int64, error) {
	var count int64
	if err := r.db.Model(&model.Follower{}).Where("profile_id = ? AND approved = ?", profileID, true).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *followerRepository) IsFollowingApproved(profileID, followingProfileID uint64) (bool, error) {
	var count int64
	if err := r.db.Model(&model.Follower{}).Where("profile_id = ? AND following_profile_id = ? AND approved = ?", profileID, followingProfileID, true).Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *followerRepository) IsFollowingActive(profileID, followingProfileID uint64) (bool, error) {
	var count int64
	if err := r.db.Model(&model.Follower{}).
		Where("profile_id = ? AND following_profile_id = ? AND rejected_at IS NULL", profileID, followingProfileID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *followerRepository) IsFollowingActiveByURL(profileID uint64, followingProfileURL string) (bool, error) {
	followingProfile, err := r.profileByURL(followingProfileURL)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	return r.IsFollowingActive(profileID, followingProfile.ID)
}

func (r *followerRepository) MarkFollowingApprovedByURL(profileID uint64, followingProfileURL string) (*model.Follower, error) {
	followingProfile, err := r.profileByURL(followingProfileURL)
	if err != nil {
		return nil, err
	}

	f := &model.Follower{}
	if err := r.db.Preload("Profile").Preload("FollowingProfile").Where(&model.Follower{ProfileID: profileID, FollowingProfileID: followingProfile.ID}).First(f).Error; err != nil {
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

func (r *followerRepository) MarkFollowingRejectedByURL(profileID uint64, followingProfileURL string) (*model.Follower, error) {
	followingProfile, err := r.profileByURL(followingProfileURL)
	if err != nil {
		return nil, err
	}

	f := &model.Follower{}
	if err := r.db.Preload("Profile").Preload("FollowingProfile").Where(&model.Follower{ProfileID: profileID, FollowingProfileID: followingProfile.ID}).First(f).Error; err != nil {
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

func (r *followerRepository) upsertFollowEntry(profileID, followingProfileID uint64, profile, followingProfile *model.Profile) (*model.Follower, error) {
	if profileID == 0 {
		return nil, errors.New("profile id is required")
	}
	if followingProfileID == 0 {
		return nil, errors.New("following profile id is required")
	}

	f := &model.Follower{}
	if err := r.db.Preload("Profile").Preload("FollowingProfile").Where(&model.Follower{ProfileID: profileID, FollowingProfileID: followingProfileID}).First(f).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			f.ProfileID = profileID
			f.FollowingProfileID = followingProfileID
			f.Profile = profile
			f.FollowingProfile = followingProfile
			f.Approved = false
			f.RejectedAt = nil
			if err := r.db.Create(f).Error; err != nil {
				return nil, err
			}
			return f, nil
		}

		return nil, err
	}

	if profile != nil {
		f.Profile = profile
	}
	if followingProfile != nil {
		f.FollowingProfile = followingProfile
	}
	if !f.Approved {
		f.ApprovedAt = nil
	}
	f.RejectedAt = nil
	if err := r.db.Save(f).Error; err != nil {
		return nil, err
	}

	return f, nil
}

func (r *followerRepository) ensureRelatedProfile(profile *model.Profile) (*model.Profile, error) {
	if profile == nil {
		return nil, errors.New("profile is nil")
	}
	if profile.ID != 0 {
		return profile, nil
	}
	if profile.UserID != nil {
		existing := &model.Profile{}
		if err := r.db.Where("user_id = ?", *profile.UserID).First(existing).Error; err != nil {
			return nil, err
		}
		return existing, nil
	}

	return profile.UpsertRemote(r.db)
}

func (r *followerRepository) profileByURL(profileURL string) (*model.Profile, error) {
	trimmed := strings.TrimSpace(profileURL)
	if trimmed == "" {
		return nil, gorm.ErrRecordNotFound
	}

	p := &model.Profile{}
	if err := r.db.Where("url = ?", trimmed).First(p).Error; err != nil {
		return nil, err
	}

	return p, nil
}
