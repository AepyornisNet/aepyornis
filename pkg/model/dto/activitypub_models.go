package dto

import (
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/model"
)

type FollowRequestResponse struct {
	ID        uint64    `json:"id"`
	ActorIRI  string    `json:"actor_iri"`
	CreatedAt time.Time `json:"created_at"`
}

func NewFollowRequestResponse(f model.Follower, actorIRI string) FollowRequestResponse {
	return FollowRequestResponse{
		ID:        f.ID,
		ActorIRI:  actorIRI,
		CreatedAt: f.CreatedAt,
	}
}
