package repository

import (
	"testing"

	"github.com/fsouza/slognil"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
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

func createRepositoryUser(t *testing.T, db *gorm.DB, username, name, apiKey string) *model.User {
	t.Helper()

	u := &model.User{
		UserData: model.UserData{
			Username: username,
			Name:     name,
			Active:   true,
		},
		UserSecrets: model.UserSecrets{
			Password: "my-password",
			APIKey:   apiKey,
		},
	}

	require.NoError(t, u.Create(db))

	return u
}

func TestUserRepository_GetByUsername(t *testing.T) {
	db := createRepositoryMemoryDB(t)
	created := createRepositoryUser(t, db, "repo-user", "Repo User", "repo-api-key")

	repo := NewUser(db)
	loaded, err := repo.GetByUsername(created.Username)

	require.NoError(t, err)
	assert.Equal(t, created.ID, loaded.ID)
	assert.Equal(t, created.Username, loaded.Username)
	assert.Equal(t, created.Name, loaded.Name)
}

func TestUserRepository_GetByID(t *testing.T) {
	db := createRepositoryMemoryDB(t)
	created := createRepositoryUser(t, db, "repo-id-user", "Repo ID User", "repo-id-key")

	repo := NewUser(db)
	loaded, err := repo.GetByID(created.ID)

	require.NoError(t, err)
	assert.Equal(t, created.ID, loaded.ID)
	assert.Equal(t, created.Username, loaded.Username)
	assert.Equal(t, created.Name, loaded.Name)
	assert.NotNil(t, loaded.PreferredUnits())
}

func TestUserRepository_GetByAPIKey(t *testing.T) {
	db := createRepositoryMemoryDB(t)
	created := createRepositoryUser(t, db, "repo-key-user", "Repo Key User", "repo-key")

	repo := NewUser(db)
	loaded, err := repo.GetByAPIKey(created.APIKey)

	require.NoError(t, err)
	assert.Equal(t, created.ID, loaded.ID)
	assert.Equal(t, created.Username, loaded.Username)
	assert.Equal(t, created.APIKey, loaded.APIKey)
}

func TestUserRepository_GetAll(t *testing.T) {
	db := createRepositoryMemoryDB(t)
	createRepositoryUser(t, db, "repo-list-1", "Repo List 1", "repo-list-key-1")
	createRepositoryUser(t, db, "repo-list-2", "Repo List 2", "repo-list-key-2")

	repo := NewUser(db)
	users, err := repo.GetAll()

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(users), 2)
}
