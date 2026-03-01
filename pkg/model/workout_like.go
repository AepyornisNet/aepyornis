package model

type WorkoutLike struct {
	Model

	WorkoutID uint64   `gorm:"index:idx_workout_like_workout_user,unique;index:idx_workout_like_workout_actor,unique;not null" json:"workout_id"`
	Workout   *Workout `gorm:"constraint:OnDelete:CASCADE" json:"-"`

	UserID *uint64 `gorm:"index:idx_workout_like_workout_user,unique" json:"user_id,omitempty"`
	User   *User   `gorm:"constraint:OnDelete:CASCADE" json:"-"`

	ActorIRI *string `gorm:"type:text;index:idx_workout_like_workout_actor,unique" json:"actor_iri,omitempty"`
}

func (WorkoutLike) TableName() string {
	return "workout_likes"
}
