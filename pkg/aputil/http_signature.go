package aputil

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/dadrus/httpsig"
	vocab "github.com/go-ap/activitypub"
)

const (
	RequestingActorContextKey = "requesting_actor"
	ContentType               = `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`
)

type KeyResolver interface {
	ResolveKey(ctx context.Context, keyID string) (httpsig.Key, error)
	Actor() *vocab.Actor
}

func VerifyRequest(req *http.Request, loader KeyResolver) (*RequestActor, error) {
	if req == nil || req.Header == nil {
		return nil, nil
	}

	if sigInput := req.Header.Get("Signature-Input"); sigInput != "" {
		actor, err := VerifyRFCSignature(req, loader)
		if err != nil {
			return nil, err
		}
		return actor, nil
	}

	return VerifyDraftSignature(req, loader)
}

const (
	sigValidDeltaDuration = 10 * time.Minute
	sigMaxAgeDuration     = 10 * 365 * 24 * time.Hour
)

type syncedNonceStore struct {
	sync.Map
}

var errInvalidNonce = errors.New("nonce already seen")

func (s *syncedNonceStore) CheckNonce(_ context.Context, n string) error {
	if n == "" {
		return nil
	}
	_, ok := s.Map.LoadOrStore(n, struct{}{})
	if ok {
		return errInvalidNonce
	}
	return nil
}

var nonceStore = new(syncedNonceStore)

func VerifyRFCSignature(req *http.Request, resolver KeyResolver) (*RequestActor, error) {
	verifier, err := httpsig.NewVerifier(
		resolver,
		httpsig.WithNonceChecker(nonceStore),
		httpsig.WithValidityTolerance(sigValidDeltaDuration),
		httpsig.WithMaxAge(sigMaxAgeDuration),
		httpsig.WithCreatedTimestampRequired(false),
		httpsig.WithExpiredTimestampRequired(false),
		httpsig.WithValidateAllSignatures(),
	)
	if err != nil {
		return nil, err
	}

	if err := verifier.Verify(httpsig.MessageFromRequest(req)); err != nil {
		return nil, err
	}

	actor := resolver.Actor()
	if actor == nil {
		return nil, errors.New("unable to resolve actor from signature")
	}

	return &RequestActor{
		Actor: *actor,
	}, nil
}
