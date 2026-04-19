package dto

import (
	"fmt"
	"strings"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/model"
)

// UserProfileResponse represents user profile info in API v2 responses
type UserProfileResponse struct {
	ID          uint64           `json:"id"`
	Email       string           `json:"email"`
	Username    string           `json:"username"`
	Domain      *string          `json:"domain,omitempty"`
	Name        string           `json:"name"`
	Birthdate   *time.Time       `json:"birthdate,omitempty"`
	ActivityPub bool             `json:"activity_pub"`
	Active      bool             `json:"active"`
	Admin       bool             `json:"admin"`
	LastVersion string           `json:"last_version"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	Profile     *ProfileSettings `json:"profile,omitempty"`
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
	ActivityPubActive    bool   `json:"activity_pub_active"`
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
	username := ""
	name := ""
	var domain *string
	var birthdate *time.Time
	if u.Profile.ID != 0 {
		username = u.Profile.Username
		name = strings.TrimSpace(u.Profile.DisplayName)
		if u.Profile.Domain != nil {
			d := strings.TrimSpace(*u.Profile.Domain)
			if d != "" {
				domain = &d
			}
		}
		if name == "" {
			name = username
			if domain != nil {
				name = fmt.Sprintf("%s@%s", username, *domain)
			}
		}
		if u.Profile.Birthdate != nil {
			bd := time.Time(*u.Profile.Birthdate)
			birthdate = &bd
		}
	}

	resp := UserProfileResponse{
		ID:          u.ID,
		Email:       u.Email,
		Username:    username,
		Domain:      domain,
		Name:        name,
		Birthdate:   birthdate,
		ActivityPub: u.ActivityPub,
		Active:      u.Active,
		Admin:       u.Admin,
		LastVersion: u.LastVersion,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
		Profile: &ProfileSettings{
			PreferredUnits:           u.PreferredUnits,
			Language:                 u.Language,
			Theme:                    u.Theme,
			TotalsShow:               string(u.TotalsShow),
			Timezone:                 u.TZ,
			AutoImportDirectory:      u.AutoImportDirectory,
			DefaultWorkoutVisibility: u.EffectiveDefaultWorkoutVisibility(),
			APIActive:                u.APIActive,
			PreferFullDate:           u.PreferFullDate,
		},
	}

	return resp
}
