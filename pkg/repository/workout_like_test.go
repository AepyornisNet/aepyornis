package repository

import (
	"testing"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func createRepositoryWorkout(t *testing.T, db *gorm.DB, dbUser *model.User) *model.Workout {
	t.Helper()

	w := &model.Workout{
		Date:      time.Now().UTC(),
		Name:      "Repository Workout",
		Type:      model.WorkoutTypeRunning,
		Creator:   "repository-test",
		Profile:   &dbUser.Profile,
		ProfileID: dbUser.Profile.ID,
		Data:      &model.WorkoutGeoMeta{},
	}

	require.NoError(t, w.Save(db))

	return w
}

func TestWorkoutLikeRepository_LikeByUserAndMaps(t *testing.T) {
	db := createRepositoryMemoryDB(t)
	owner := createRepositoryUser(t, db, "likes-owner", "Likes Owner", "likes-owner-key")
	liker := createRepositoryUser(t, db, "likes-liker", "Likes Liker", "likes-liker-key")
	workout := createRepositoryWorkout(t, db, owner)

	injector := do.New()
	do.ProvideValue(injector, db)
	repo, err := NewWorkoutLike(injector)
	require.NoError(t, err)

	require.NoError(t, repo.LikeByProfile(workout.ID, liker.Profile.ID))
	require.NoError(t, repo.LikeByProfile(workout.ID, liker.Profile.ID))

	likes, err := repo.ListByWorkoutID(workout.ID)
	require.NoError(t, err)
	require.Len(t, likes, 1)
	require.NotNil(t, likes[0].ProfileID)
	assert.Equal(t, liker.Profile.ID, *likes[0].ProfileID)

	counts, err := repo.CountMapByWorkoutIDs([]uint64{workout.ID})
	require.NoError(t, err)
	assert.EqualValues(t, 1, counts[workout.ID])

	likedByLiker, err := repo.LikedMapByProfile([]uint64{workout.ID}, liker.Profile.ID)
	require.NoError(t, err)
	assert.True(t, likedByLiker[workout.ID])

	likedByOwner, err := repo.LikedMapByProfile([]uint64{workout.ID}, owner.Profile.ID)
	require.NoError(t, err)
	assert.False(t, likedByOwner[workout.ID])
}

func TestWorkoutLikeRepository_LikeByActorAndUndo(t *testing.T) {
	db := createRepositoryMemoryDB(t)
	owner := createRepositoryUser(t, db, "likes-owner-actor", "Likes Owner Actor", "likes-owner-actor-key")
	workout := createRepositoryWorkout(t, db, owner)

	injector := do.New()
	do.ProvideValue(injector, db)
	repo, err := NewWorkoutLike(injector)
	require.NoError(t, err)
	actorIRI := "https://remote.example/users/runner"

	require.NoError(t, repo.LikeByActorIRI(workout.ID, actorIRI))
	require.NoError(t, repo.LikeByActorIRI(workout.ID, actorIRI))

	likes, err := repo.ListByWorkoutID(workout.ID)
	require.NoError(t, err)
	require.Len(t, likes, 1)
	require.NotNil(t, likes[0].Profile)
	assert.Equal(t, actorIRI, likes[0].Profile.ActorURL())

	counts, err := repo.CountMapByWorkoutIDs([]uint64{workout.ID})
	require.NoError(t, err)
	assert.EqualValues(t, 1, counts[workout.ID])

	require.NoError(t, repo.UnlikeByActorIRI(workout.ID, actorIRI))

	counts, err = repo.CountMapByWorkoutIDs([]uint64{workout.ID})
	require.NoError(t, err)
	assert.EqualValues(t, 0, counts[workout.ID])
}
