package repository

import (
	"testing"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/fsouza/slognil"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func createRepositoryMemoryDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := model.Connect("memory", "", false, slognil.NewLogger())
	require.NoError(t, err)

	return db
}

func createRepositoryInjector(db *gorm.DB) do.Injector {
	injector := do.New()
	do.ProvideValue(injector, db)

	return injector
}

func createRepositoryUser(t *testing.T, db *gorm.DB, username, name, apiKey string) *model.User {
	t.Helper()

	u := &model.User{
		UserData: model.UserData{
			Active: true,
		},
		UserSecrets: model.UserSecrets{
			Email:    username + "@example.com",
			Password: "my-password",
			APIKey:   apiKey,
		},
		Profile: model.Profile{Username: username, DisplayName: name},
	}

	require.NoError(t, u.Create(db))

	return u
}

func TestUserRepository_GetByUsername(t *testing.T) {
	db := createRepositoryMemoryDB(t)
	created := createRepositoryUser(t, db, "repo-user", "Repo User", "repo-api-key")

	repo, err := NewUser(createRepositoryInjector(db))
	require.NoError(t, err)
	loaded, err := repo.GetByUsername(created.Profile.Username)

	require.NoError(t, err)
	assert.Equal(t, created.ID, loaded.ID)
	assert.Equal(t, created.Profile.Username, loaded.Profile.Username)
	assert.Equal(t, created.Profile.DisplayName, loaded.Profile.DisplayName)
}

func TestUserRepository_GetByHandle(t *testing.T) {
	db := createRepositoryMemoryDB(t)
	created := createRepositoryUser(t, db, "repo-handle-user", "Repo Handle User", "repo-handle-key")

	repo, err := NewUser(createRepositoryInjector(db))
	require.NoError(t, err)

	t.Run("plain username", func(t *testing.T) {
		loaded, err := repo.GetByHandle(created.Profile.Username, "example.com")
		require.NoError(t, err)
		assert.Equal(t, created.ID, loaded.ID)
	})

	t.Run("acct handle", func(t *testing.T) {
		loaded, err := repo.GetByHandle("@"+created.Profile.Username+"@example.com", "example.com")
		require.NoError(t, err)
		assert.Equal(t, created.ID, loaded.ID)
	})

	t.Run("actor url", func(t *testing.T) {
		loaded, err := repo.GetByHandle("https://example.com/ap/users/"+created.Profile.Username, "example.com")
		require.NoError(t, err)
		assert.Equal(t, created.ID, loaded.ID)
	})

	t.Run("remote host rejected", func(t *testing.T) {
		_, err := repo.GetByHandle("@"+created.Profile.Username+"@remote.example", "example.com")
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}

func TestUserRepository_GetByID(t *testing.T) {
	db := createRepositoryMemoryDB(t)
	created := createRepositoryUser(t, db, "repo-id-user", "Repo ID User", "repo-id-key")

	repo, err := NewUser(createRepositoryInjector(db))
	require.NoError(t, err)
	loaded, err := repo.GetByID(created.ID)

	require.NoError(t, err)
	assert.Equal(t, created.ID, loaded.ID)
	assert.Equal(t, created.Profile.Username, loaded.Profile.Username)
	assert.Equal(t, created.Profile.DisplayName, loaded.Profile.DisplayName)
	assert.NotEmpty(t, loaded.PreferredUnits.Distance())
}

func TestUserRepository_GetByAPIKey(t *testing.T) {
	db := createRepositoryMemoryDB(t)
	created := createRepositoryUser(t, db, "repo-key-user", "Repo Key User", "repo-key")

	repo, err := NewUser(createRepositoryInjector(db))
	require.NoError(t, err)
	loaded, err := repo.GetByAPIKey(created.APIKey)

	require.NoError(t, err)
	assert.Equal(t, created.ID, loaded.ID)
	assert.Equal(t, created.Profile.Username, loaded.Profile.Username)
	assert.Equal(t, created.APIKey, loaded.APIKey)
}

func TestUserRepository_GetAll(t *testing.T) {
	db := createRepositoryMemoryDB(t)
	createRepositoryUser(t, db, "repo-list-1", "Repo List 1", "repo-list-key-1")
	createRepositoryUser(t, db, "repo-list-2", "Repo List 2", "repo-list-key-2")

	repo, err := NewUser(createRepositoryInjector(db))
	require.NoError(t, err)
	users, err := repo.GetAll()

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(users), 2)
}
