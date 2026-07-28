package notification

import (
	"encoding/json"
	"fmt"

	"gorm.io/datatypes"
)

type workoutLike struct {
	ActorName string
	WorkoutID uint64
}

func NewWorkoutLike(actorName string, workoutID uint64) *workoutLike {
	return &workoutLike{
		ActorName: actorName,
		WorkoutID: workoutID,
	}
}

func (*workoutLike) GetType() string {
	return "workout_like"
}

func (*workoutLike) GetSubject() string {
	return "New workout like"
}

func (w *workoutLike) GetMessage() string {
	return fmt.Sprintf("%s liked your workout", w.ActorName)
}

func (w *workoutLike) GetMeta() *datatypes.JSON {
	meta := map[string]any{
		"url":        fmt.Sprintf("/workouts/%d", w.WorkoutID),
		"workout_id": w.WorkoutID,
	}

	data, err := json.Marshal(meta)
	if err != nil {
		return nil
	}

	jsonData := datatypes.JSON(data)
	return &jsonData
}

func (*workoutLike) AllowDB() bool {
	return true
}

func (*workoutLike) AllowEmail() bool {
	return true
}

func (*workoutLike) AllowWebpush() bool {
	return true
}
