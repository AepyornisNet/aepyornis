package notification

import (
	"encoding/json"
	"fmt"

	"gorm.io/datatypes"
)

type workoutReply struct {
	ActorName string
	WorkoutID uint64
}

func NewWorkoutReply(actorName string, workoutID uint64) *workoutReply {
	return &workoutReply{
		ActorName: actorName,
		WorkoutID: workoutID,
	}
}

func (*workoutReply) GetType() string {
	return "workout_reply"
}

func (*workoutReply) GetSubject() string {
	return "New workout comment"
}

func (w *workoutReply) GetMessage() string {
	return fmt.Sprintf("%s commented on your workout", w.ActorName)
}

func (w *workoutReply) GetMeta() *datatypes.JSON {
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

func (*workoutReply) AllowDB() bool {
	return true
}

func (*workoutReply) AllowEmail() bool {
	return true
}

func (*workoutReply) AllowWebpush() bool {
	return true
}
