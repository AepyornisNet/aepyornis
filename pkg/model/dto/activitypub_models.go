package dto

import (
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
)

type FollowRequestResponse struct {
	ID        uint64    `json:"id"`
	ActorIRI  string    `json:"actor_iri"`
	CreatedAt time.Time `json:"created_at"`
}

func NewFollowRequestResponse(f model.Follower) FollowRequestResponse {
	return FollowRequestResponse{
		ID:        f.ID,
		ActorIRI:  f.ActorIRI,
		CreatedAt: f.CreatedAt,
	}
}
