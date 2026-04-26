package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AepyornisNet/aepyornis/pkg/config"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/fsouza/slognil"
	"github.com/labstack/echo/v4"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestActivityPubProfileService_GetByActorIRIReturnsExistingProfile(t *testing.T) {
	db := createActivityPubProfileTestDB(t)

	actorURL := "https://remote.example/ap/users/alice"
	avatarURL := actorURL + "/avatar.png"
	profile, err := (&model.Profile{
		Username:        "alice",
		DisplayName:     "Alice",
		URL:             &actorURL,
		AvatarRemoteURL: &avatarURL,
	}).UpsertRemote(db)
	require.NoError(t, err)

	svc := createActivityPubProfileService(t, db)
	resolved, err := svc.GetByActorIRI(context.Background(), actorURL)

	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, profile.ID, resolved.ID)
	assert.Equal(t, "Alice", resolved.DisplayName)
	require.NotNil(t, resolved.AvatarRemoteURL)
	assert.Equal(t, avatarURL, *resolved.AvatarRemoteURL)
}

func TestActivityPubProfileService_GetByActorIRIFetchesAndPersistsProfile(t *testing.T) {
	db := createActivityPubProfileTestDB(t)

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ap/users/alice" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/activity+json")
		_, _ = w.Write([]byte(`{
			"@context": "https://www.w3.org/ns/activitystreams",
			"id": "` + server.URL + `/ap/users/alice",
			"type": "Person",
			"preferredUsername": "alice",
			"name": "Alice Remote",
			"inbox": "` + server.URL + `/ap/users/alice/inbox",
			"outbox": "` + server.URL + `/ap/users/alice/outbox",
			"followers": "` + server.URL + `/ap/users/alice/followers",
			"icon": {
				"type": "Image",
				"url": "` + server.URL + `/media/alice.png"
			}
		}`))
	}))
	defer server.Close()

	svc := createActivityPubProfileService(t, db)
	actorURL := server.URL + "/ap/users/alice"
	resolved, err := svc.GetByActorIRI(context.Background(), actorURL)

	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, "alice", resolved.Username)
	assert.Equal(t, "Alice Remote", resolved.DisplayName)
	require.NotNil(t, resolved.AvatarRemoteURL)
	assert.Equal(t, server.URL+"/media/alice.png", *resolved.AvatarRemoteURL)

	stored := &model.Profile{}
	require.NoError(t, db.Where("url = ?", actorURL).First(stored).Error)
	assert.Equal(t, resolved.ID, stored.ID)
}

func createActivityPubProfileService(t *testing.T, db *gorm.DB) ActivityPubProfileService {
	t.Helper()

	injector := do.New(Package)
	do.ProvideValue(injector, db)
	do.ProvideValue(injector, &config.Config{})
	do.ProvideValue(injector, echo.New())

	svc, err := NewActivityPubProfileService(injector)
	require.NoError(t, err)

	return svc
}

func createActivityPubProfileTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := model.Connect("memory", "", false, slognil.NewLogger())
	require.NoError(t, err)

	return db
}
