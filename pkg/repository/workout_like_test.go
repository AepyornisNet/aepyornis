package repository

import (
	"testing"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func createRepositoryWorkout(t *testing.T, db *gorm.DB, dbUser *model.User) *model.Workout {
	t.Helper()

	w := &model.Workout{
		Date:   time.Now().UTC(),
		Name:   "Repository Workout",
		Type:   model.WorkoutTypeRunning,
		User:   dbUser,
		UserID: dbUser.ID,
		Data: &model.MapData{
			Creator: "repository-test",
		},
	}

	require.NoError(t, w.Save(db))

	return w
}

func TestWorkoutLikeRepository_LikeByUserAndMaps(t *testing.T) {
	db := createRepositoryMemoryDB(t)
	owner := createRepositoryUser(t, db, "likes-owner", "Likes Owner", "likes-owner-key")
	liker := createRepositoryUser(t, db, "likes-liker", "Likes Liker", "likes-liker-key")
	workout := createRepositoryWorkout(t, db, owner)

	repo := NewWorkoutLike(db)

	require.NoError(t, repo.LikeByUser(workout.ID, liker.ID))
	require.NoError(t, repo.LikeByUser(workout.ID, liker.ID))

	counts, err := repo.CountMapByWorkoutIDs([]uint64{workout.ID})
	require.NoError(t, err)
	assert.EqualValues(t, 1, counts[workout.ID])

	likedByLiker, err := repo.LikedMapByUser([]uint64{workout.ID}, liker.ID)
	require.NoError(t, err)
	assert.True(t, likedByLiker[workout.ID])

	likedByOwner, err := repo.LikedMapByUser([]uint64{workout.ID}, owner.ID)
	require.NoError(t, err)
	assert.False(t, likedByOwner[workout.ID])
}

func TestWorkoutLikeRepository_LikeByActorAndUndo(t *testing.T) {
	db := createRepositoryMemoryDB(t)
	owner := createRepositoryUser(t, db, "likes-owner-actor", "Likes Owner Actor", "likes-owner-actor-key")
	workout := createRepositoryWorkout(t, db, owner)

	repo := NewWorkoutLike(db)
	actorIRI := "https://remote.example/users/runner"

	require.NoError(t, repo.LikeByActorIRI(workout.ID, actorIRI))
	require.NoError(t, repo.LikeByActorIRI(workout.ID, actorIRI))

	counts, err := repo.CountMapByWorkoutIDs([]uint64{workout.ID})
	require.NoError(t, err)
	assert.EqualValues(t, 1, counts[workout.ID])

	require.NoError(t, repo.UnlikeByActorIRI(workout.ID, actorIRI))

	counts, err = repo.CountMapByWorkoutIDs([]uint64{workout.ID})
	require.NoError(t, err)
	assert.EqualValues(t, 0, counts[workout.ID])
}
