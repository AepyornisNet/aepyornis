package dto

import (
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
)

// UserProfileResponse represents user profile info in API v2 responses
type UserProfileResponse struct {
	ID                       uint64                   `json:"id"`
	Username                 string                   `json:"username"`
	Name                     string                   `json:"name"`
	Birthdate                *time.Time               `json:"birthdate,omitempty"`
	ActivityPub              bool                     `json:"activity_pub"`
	Active                   bool                     `json:"active"`
	Admin                    bool                     `json:"admin"`
	LastVersion              string                   `json:"last_version"`
	CreatedAt                time.Time                `json:"created_at"`
	UpdatedAt                time.Time                `json:"updated_at"`
	PreferredUnits           model.UserPreferredUnits `json:"preferred_units"`
	Language                 string                   `json:"language"`
	Theme                    string                   `json:"theme"`
	Timezone                 string                   `json:"timezone"`
	DefaultWorkoutVisibility model.WorkoutVisibility  `json:"default_workout_visibility"`
	PreferFullDate           bool                     `json:"prefer_full_date"`
	Profile                  *ProfileSettings         `json:"profile,omitempty"`
}

// TODO: Remove duplicate fields between UserProfileResponse and ProfileSettings

// ProfileSettings contains the user's profile
type ProfileSettings struct {
	PreferredUnits           model.UserPreferredUnits `json:"preferred_units"`
	Language                 string                   `json:"language"`
	Theme                    string                   `json:"theme"`
	TotalsShow               string                   `json:"totals_show"`
	Timezone                 string                   `json:"timezone"`
	AutoImportDirectory      string                   `json:"auto_import_directory"`
	DefaultWorkoutVisibility model.WorkoutVisibility  `json:"default_workout_visibility"`
	APIActive                bool                     `json:"api_active"`
	APIKey                   string                   `json:"api_key,omitempty"` // #nosec G117 -- API response key is intentionally named api_key
	PreferFullDate           bool                     `json:"prefer_full_date"`
}

// AppInfoResponse represents application info in API v2 responses
type AppInfoResponse struct {
	Version              string `json:"version"`
	VersionSha           string `json:"version_sha"`
	RegistrationDisabled bool   `json:"registration_disabled"`
	SocialsDisabled      bool   `json:"socials_disabled"`
	AutoImportEnabled    bool   `json:"auto_import_enabled"`
}

type ActivityPubProfileSummaryResponse struct {
	ID             uint64    `json:"id"`
	Username       string    `json:"username"`
	Name           string    `json:"name"`
	Handle         string    `json:"handle"`
	ActorURL       string    `json:"actor_url"`
	IconURL        string    `json:"icon_url"`
	IsExternal     bool      `json:"is_external"`
	IsOwn          bool      `json:"is_own"`
	IsFollowing    bool      `json:"is_following"`
	PostsCount     int64     `json:"posts_count"`
	FollowersCount int64     `json:"followers_count"`
	FollowingCount int64     `json:"following_count"`
	MemberSince    time.Time `json:"member_since"`
}

// NewUserProfileResponse converts a database user to API response
func NewUserProfileResponse(u *model.User) UserProfileResponse {
	resp := UserProfileResponse{
		ID:                       u.ID,
		Username:                 u.Username,
		Name:                     u.Name,
		ActivityPub:              u.ActivityPub,
		Active:                   u.Active,
		Admin:                    u.Admin,
		LastVersion:              u.LastVersion,
		CreatedAt:                u.CreatedAt,
		UpdatedAt:                u.UpdatedAt,
		PreferredUnits:           u.Profile.PreferredUnits,
		Language:                 u.Profile.Language,
		Theme:                    u.Profile.Theme,
		Timezone:                 u.Profile.Timezone,
		DefaultWorkoutVisibility: u.Profile.EffectiveDefaultWorkoutVisibility(),
		PreferFullDate:           u.Profile.PreferFullDate,
		Profile: &ProfileSettings{
			PreferredUnits:           u.Profile.PreferredUnits,
			Language:                 u.Profile.Language,
			Theme:                    u.Profile.Theme,
			TotalsShow:               string(u.Profile.TotalsShow),
			Timezone:                 u.Profile.Timezone,
			AutoImportDirectory:      u.Profile.AutoImportDirectory,
			DefaultWorkoutVisibility: u.Profile.EffectiveDefaultWorkoutVisibility(),
			APIActive:                u.Profile.APIActive,
			PreferFullDate:           u.Profile.PreferFullDate,
		},
	}

	if u.Birthdate != nil {
		bd := time.Time(*u.Birthdate)
		resp.Birthdate = &bd
	}

	return resp
}
