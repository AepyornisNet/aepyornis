// Package aptest provides shared helpers and test doubles for ActivityPub tests.
package aptest

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/stretchr/testify/require"
)

// GenerateKeyPair creates a 2048-bit RSA key pair and returns both as PEM strings.
func GenerateKeyPair(t *testing.T) (privateKeyPEM, publicKeyPEM string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privateBytes := x509.MarshalPKCS1PrivateKey(key)
	privatePEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privateBytes})

	publicBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicBytes})

	return string(privatePEM), string(publicPEM)
}

// MockActorServer starts an httptest TLS server that serves a minimal ActivityPub
// actor document at /users/<username>. It returns the server, actor IRI, and key ID.
// The caller is responsible for closing the server (use t.Cleanup).
func MockActorServer(t *testing.T, username, publicKeyPEM string) (server *httptest.Server, actorIRI, keyID string) {
	t.Helper()

	mux := http.NewServeMux()
	server = httptest.NewTLSServer(mux)
	t.Cleanup(server.Close)

	actorIRI = server.URL + "/users/" + username
	keyID = actorIRI + "#main-key"

	capActorIRI := actorIRI
	capKeyID := keyID
	capPublicKeyPEM := publicKeyPEM

	mux.HandleFunc("/users/"+username, func(w http.ResponseWriter, r *http.Request) {
		actor := vocab.Actor{
			ID:    vocab.ID(capActorIRI),
			Type:  vocab.PersonType,
			Inbox: vocab.IRI(capActorIRI + "/inbox"),
			PublicKey: vocab.PublicKey{
				ID:           vocab.ID(capKeyID),
				Owner:        vocab.IRI(capActorIRI),
				PublicKeyPem: capPublicKeyPEM,
			},
		}

		data, err := jsonld.WithContext(
			jsonld.IRI(vocab.ActivityBaseURI),
			jsonld.IRI("https://w3id.org/security/v1"),
		).Marshal(actor)
		if err != nil {
			http.Error(w, "marshal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/activity+json")
		_, _ = w.Write(data)
	})

	return server, actorIRI, keyID
}

// UseTestTLSTransport replaces http.DefaultTransport with the given test server's
// TLS transport for the duration of the test and restores it afterwards.
func UseTestTLSTransport(t *testing.T, server *httptest.Server) {
	t.Helper()

	old := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	t.Cleanup(func() { http.DefaultTransport = old })
}

// WebFingerHandler returns an http.HandlerFunc that responds to WebFinger requests
// with a JSON body containing a self link to actorIRI.
func WebFingerHandler(actorIRI string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"links": []map[string]string{
				{
					"rel":  "self",
					"type": "application/activity+json",
					"href": actorIRI,
				},
			},
		}
		data, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/jrd+json")
		_, _ = w.Write(data)
	}
}

// MockFollowerRepo is a test double for activitypub.InboxFollowerRepository.
type MockFollowerRepo struct {
	UpsertFn  func(userID uint64, actorIRI, actorInbox string) (*model.Follower, error)
	ApproveFn func(userID uint64, actorIRI string) (*model.Follower, error)
	RejectFn  func(userID uint64, actorIRI string) (*model.Follower, error)
	DeleteFn  func(userID uint64, actorIRI string) error
}

func (m *MockFollowerRepo) UpsertFollowerRequest(userID uint64, actorIRI, actorInbox string) (*model.Follower, error) {
	if m.UpsertFn != nil {
		return m.UpsertFn(userID, actorIRI, actorInbox)
	}
	return &model.Follower{ActorIRI: actorIRI, ActorInbox: actorInbox}, nil
}

func (m *MockFollowerRepo) MarkFollowingApprovedByActorIRI(userID uint64, actorIRI string) (*model.Follower, error) {
	if m.ApproveFn != nil {
		return m.ApproveFn(userID, actorIRI)
	}
	return &model.Follower{ActorIRI: actorIRI}, nil
}

func (m *MockFollowerRepo) MarkFollowingRejectedByActorIRI(userID uint64, actorIRI string) (*model.Follower, error) {
	if m.RejectFn != nil {
		return m.RejectFn(userID, actorIRI)
	}
	return &model.Follower{ActorIRI: actorIRI}, nil
}

func (m *MockFollowerRepo) DeleteFollowerByActorIRI(userID uint64, actorIRI string) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(userID, actorIRI)
	}
	return nil
}

// MockOutboxRepo is a test double for activitypub.InboxOutboxRepository.
type MockOutboxRepo struct {
	ResolveFn func(userID uint64, objectOrActivityID string) (uint64, error)
}

func (m *MockOutboxRepo) ResolveWorkoutIDByObjectOrActivityID(userID uint64, objectOrActivityID string) (uint64, error) {
	if m.ResolveFn != nil {
		return m.ResolveFn(userID, objectOrActivityID)
	}
	return 0, nil
}

// MockWorkoutLikeRepo is a test double for activitypub.InboxWorkoutLikeRepository.
type MockWorkoutLikeRepo struct {
	LikeFn   func(workoutID uint64, actorIRI string) error
	UnlikeFn func(workoutID uint64, actorIRI string) error
}

func (m *MockWorkoutLikeRepo) LikeByActorIRI(workoutID uint64, actorIRI string) error {
	if m.LikeFn != nil {
		return m.LikeFn(workoutID, actorIRI)
	}
	return nil
}

func (m *MockWorkoutLikeRepo) UnlikeByActorIRI(workoutID uint64, actorIRI string) error {
	if m.UnlikeFn != nil {
		return m.UnlikeFn(workoutID, actorIRI)
	}
	return nil
}

// MockWorkoutReplyRepo is a test double for activitypub.InboxWorkoutReplyRepository.
type MockWorkoutReplyRepo struct {
	ReplyFn         func(workoutID uint64, objectIRI, actorIRI, actorName, content string) error
	UpdateFn        func(workoutID uint64, objectIRI, actorIRI, actorName, content string) error
	DeleteFn        func(workoutID uint64, objectIRI string) error
	ResolveByObjFn  func(objectIRI string) (uint64, error)
}

func (m *MockWorkoutReplyRepo) ReplyByActorIRI(workoutID uint64, objectIRI, actorIRI, actorName, content string) error {
	if m.ReplyFn != nil {
		return m.ReplyFn(workoutID, objectIRI, actorIRI, actorName, content)
	}
	return nil
}

func (m *MockWorkoutReplyRepo) UpdateReplyByObjectIRI(workoutID uint64, objectIRI, actorIRI, actorName, content string) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(workoutID, objectIRI, actorIRI, actorName, content)
	}
	return nil
}

func (m *MockWorkoutReplyRepo) DeleteReplyByObjectIRI(workoutID uint64, objectIRI string) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(workoutID, objectIRI)
	}
	return nil
}

func (m *MockWorkoutReplyRepo) ResolveWorkoutIDByObjectIRI(objectIRI string) (uint64, error) {
	if m.ResolveByObjFn != nil {
		return m.ResolveByObjFn(objectIRI)
	}
	return 0, nil
}
