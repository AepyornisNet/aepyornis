package model

import (
	"errors"
	"testing"

	"github.com/fsouza/slognil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func init() { //nolint:gochecknoinits
	goOffline()
}

func defaultUser() *User {
	return &User{
		UserSecrets: UserSecrets{
			Email:    "my-email@example.com",
			Password: "my-password",
		},
		Profile: Profile{
			Username:    "my-username",
			DisplayName: "my-name",
		},
	}
}

func dummyMapData() *WorkoutGeoMeta {
	return &WorkoutGeoMeta{}
}

func createMemoryDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := Connect("memory", "", false, slognil.NewLogger())
	require.NoError(t, err)

	return db
}

func createDefaultUser(t *testing.T, db *gorm.DB) {
	t.Helper()

	require.NoError(t, defaultUser().Create(db))
}

func getUserByUsernameForTest(db *gorm.DB, username string) (*User, error) {
	var u User

	err := db.
		Preload("Profile").
		Joins("JOIN profiles ON profiles.user_id = users.id").
		Where("profiles.username = ?", username).
		First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if u.Profile.UserID != nil && u.ID == *u.Profile.UserID {
		u.Profile.User = &u
	}

	u.SetDB(db)

	return &u, nil
}

func getUserByIDForTest(db *gorm.DB, userID uint64) (*User, error) {
	var u User

	err := db.Preload("Profile").First(&u, userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	u.SetDB(db)

	return &u, nil
}

func listUserWorkoutsForTest(db *gorm.DB, user *User) ([]*Workout, error) {
	workouts := make([]*Workout, 0)

	if err := PreloadWorkoutData(db).Where(&Workout{ProfileID: user.Profile.ID}).Order("date DESC").Find(&workouts).Error; err != nil {
		return nil, err
	}

	return workouts, nil
}

func countUsersForTest(db *gorm.DB) (int64, error) {
	var count int64

	if err := db.Model(&User{}).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func TestUser_IsValid(t *testing.T) {
	u := defaultUser()
	u.Active = true

	require.NoError(t, u.IsValid())
	assert.True(t, u.IsActive())
}

func TestUser_PasswordIsValid(t *testing.T) {
	pwd := "my-password"
	u := defaultUser()
	u.Active = true
	u.Password = ""

	require.Error(t, u.IsValid())
	assert.Empty(t, u.Salt)
	assert.Empty(t, u.Password)

	require.NoError(t, u.SetPassword(pwd))

	require.NoError(t, u.IsValid())
	assert.NotEmpty(t, u.Salt)
	assert.NotEmpty(t, u.Password)
	require.NotEqual(t, u.Password, pwd)

	assert.True(t, u.ValidLogin(pwd))
	assert.False(t, u.ValidLogin(pwd+pwd))
}

func TestUser_IsNotActive(t *testing.T) {
	u := User{
		UserData: UserData{
			Active:   false,
		},
		UserSecrets: UserSecrets{
			Email:    "my-email@example.com",
			Password: "my-password",
		},
		Profile: Profile{Username: "my-username", DisplayName: "my-name"},
	}

	require.NoError(t, u.IsValid())
	assert.False(t, u.IsActive())
}

func TestUser_EmailAddressIsValid(t *testing.T) {
	u := User{
		UserSecrets: UserSecrets{
			Email:    "my-username@localhost",
			Password: "my-password",
		},
		Profile: Profile{Username: "my-username", DisplayName: "my-name"},
	}

	require.NoError(t, u.IsValid())
	assert.False(t, u.IsActive())
}

func TestUser_EmailIsNotValid(t *testing.T) {
	u := User{
		UserSecrets: UserSecrets{
			Email:    "not-an-email",
			Password: "my-password",
		},
		Profile: Profile{Username: "my-username", DisplayName: "my-name"},
	}

	require.ErrorIs(t, u.IsValid(), ErrEmailInvalid)
	assert.False(t, u.IsActive())
}

func TestUser_PasswordNotSet(t *testing.T) {
	u := User{
		UserSecrets: UserSecrets{
			Email:    "my-email@example.com",
			Password: "",
		},
		Profile: Profile{Username: "username", DisplayName: "my-name"},
	}

	require.ErrorIs(t, u.IsValid(), ErrPasswordInvalidLength)
	assert.False(t, u.IsActive())
}

func TestUser_BeforeCreateNoPassword(t *testing.T) {
	db := createMemoryDB(t)
	u := &User{
		UserSecrets: UserSecrets{
			Email:    "my-email@example.com",
			Password: "",
		},
		Profile: Profile{Username: "username", DisplayName: "my-name"},
	}

	require.Error(t, u.Create(db))
	assert.NotEmpty(t, u.Salt)

	u, err := getUserByUsernameForTest(db, "other-username")
	require.NoError(t, err)
	require.Nil(t, u)
}

func TestDatabaseUserCreate(t *testing.T) {
	db := createMemoryDB(t)
	u := &User{
		UserSecrets: UserSecrets{
			Email:    "username@example.com",
			Password: "my-password",
		},
		Profile: Profile{Username: "username", DisplayName: "my-name"},
	}

	require.NoError(t, u.Create(db))
	require.NoError(t, u.IsValid())
	assert.False(t, u.IsActive())
	assert.NotEmpty(t, u.Salt)
	assert.NotEmpty(t, u.ID)

	u, err := getUserByUsernameForTest(db, "username")
	require.NoError(t, err)
	assert.Equal(t, "my-name", u.Profile.DisplayName)

	u, err = getUserByIDForTest(db, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "my-name", u.Profile.DisplayName)
}

func TestDatabaseUsers(t *testing.T) {
	db := createMemoryDB(t)

	u1 := User{
		UserSecrets: UserSecrets{
			Email:    "username1@example.com",
			Password: "my-password",
		},
		Profile: Profile{Username: "username1", DisplayName: "username1"},
	}
	require.NoError(t, u1.Create(db))

	users, err := countUsersForTest(db)
	require.NoError(t, err)
	assert.EqualValues(t, 1, users)

	u2 := User{
		UserSecrets: UserSecrets{
			Email:    "username2@example.com",
			Password: "my-password",
		},
		Profile: Profile{Username: "username2", DisplayName: "username2"},
	}
	require.NoError(t, u2.Create(db))

	users, err = countUsersForTest(db)
	require.NoError(t, err)
	assert.EqualValues(t, 2, users)
}

func TestDatabaseUserSave(t *testing.T) {
	db := createMemoryDB(t)
	u := defaultUser()

	require.NoError(t, u.Create(db))

	u, err := getUserByUsernameForTest(db, "my-username")
	require.NoError(t, err)
	assert.Equal(t, "my-name", u.Profile.DisplayName)

	u.Profile.DisplayName = "other-name"
	require.NoError(t, u.Profile.Save(db))

	u, err = getUserByUsernameForTest(db, "my-username")
	require.NoError(t, err)
	assert.Equal(t, "other-name", u.Profile.DisplayName)
}

func TestDatabaseUserCreateDoubleUsername(t *testing.T) {
	db := createMemoryDB(t)
	createDefaultUser(t, db)

	err := defaultUser().Create(db)
	require.Error(t, err)
	assert.ErrorIs(t, err, gorm.ErrDuplicatedKey)

	users, err := countUsersForTest(db)
	require.NoError(t, err)
	assert.EqualValues(t, 1, users)
}

func TestDatabaseUserDeleteUser(t *testing.T) {
	db := createMemoryDB(t)

	u := defaultUser()
	require.NoError(t, u.Create(db))

	users, err := countUsersForTest(db)
	require.NoError(t, err)
	assert.EqualValues(t, 1, users)

	require.NoError(t, u.Delete(db))

	users, err = countUsersForTest(db)
	require.NoError(t, err)
	assert.Zero(t, users)
}

func TestDatabaseProfileSave(t *testing.T) {
	db := createMemoryDB(t)
	u := &User{
		UserSecrets: UserSecrets{
			Email:    "username@example.com",
			Password: "my-password",
		},
		Profile: Profile{Username: "username", DisplayName: "my-name"},
	}
	u.Profile.DisplayName = "en"

	require.NoError(t, u.Create(db))
	assert.NotEmpty(t, u.Profile.ID)

	u, err := getUserByUsernameForTest(db, "username")
	require.NoError(t, err)
	assert.Equal(t, "en", u.Profile.DisplayName)

	u.Profile.DisplayName = "de"
	require.NoError(t, u.Profile.Save(db))
	u, err = getUserByUsernameForTest(db, "username")
	require.NoError(t, err)
	assert.Equal(t, "de", u.Profile.DisplayName)
}

func TestDatabaseUserWorkouts(t *testing.T) {
	populateGPXFS()

	db := createMemoryDB(t)

	u := defaultUser()
	require.NoError(t, u.Create(db))

	workouts, err := listUserWorkoutsForTest(db, u)
	require.NoError(t, err)
	assert.Empty(t, workouts)

	u.Profile.User = u
	w1, addErr := u.Profile.AddWorkout(
		db,
		WorkoutTypeAutoDetect,
		"some notes",
		"file.gpx",
		[]byte("invalid content"),
	)
	require.NotEmpty(t, addErr)
	require.ErrorIs(t, addErr[0], ErrInvalidData)
	assert.Nil(t, w1)

	workouts, err = listUserWorkoutsForTest(db, u)
	require.NoError(t, err)
	assert.Empty(t, workouts)

	f1, err := gpxFS.ReadFile("sample1.gpx")
	require.NoError(t, err)

	w2, addErr := u.Profile.AddWorkout(
		db,
		WorkoutTypeAutoDetect,
		"some notes",
		"file.gpx",
		f1,
	)
	require.Empty(t, addErr)
	assert.Len(t, w2, 1)
	w2_1 := w2[0]

	workouts, err = listUserWorkoutsForTest(db, u)
	require.NoError(t, err)
	assert.Len(t, workouts, 1)

	w2_1.UpdateExtraMetrics()
	assert.True(t, w2_1.HasElevation())
	assert.True(t, w2_1.HasHeartRate())

	w2_1.Type = WorkoutTypeWalking
	require.NoError(t, w2_1.Save(db))
}
