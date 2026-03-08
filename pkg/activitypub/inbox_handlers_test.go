package activitypub

import (
	"errors"
	"testing"

	vocab "github.com/go-ap/activitypub"
	"github.com/jovandeginste/workout-tracker/v2/pkg/activitypub/aptest"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newActor is a convenience helper to build a minimal vocab.Actor for tests.
func newActor(iri, inbox string) *vocab.Actor {
	a := &vocab.Actor{}
	a.ID = vocab.ID(iri)
	a.Type = vocab.PersonType
	if inbox != "" {
		a.Inbox = vocab.IRI(inbox)
	}
	return a
}

// newActivity builds a minimal vocab.Activity with the provided type and actor.
func newActivity(activityType vocab.ActivityVocabularyType, actorIRI string) *vocab.Activity {
	return &vocab.Activity{
		Type:  activityType,
		Actor: vocab.IRI(actorIRI),
	}
}

// ---------------------------------------------------------------------------
// HandleInboxActivity – nil activity
// ---------------------------------------------------------------------------

func TestHandleInboxActivity_NilActivity(t *testing.T) {
	ctx := InboxHandlerContext{Activity: nil}
	handled, err := HandleInboxActivity(ctx)
	require.NoError(t, err)
	assert.False(t, handled)
}

// ---------------------------------------------------------------------------
// Follow activity
// ---------------------------------------------------------------------------

func TestHandleFollowActivity_StoresFollowerRequest(t *testing.T) {
	var (
		calledUserID    uint64
		calledActorIRI  string
		calledActorInbox string
	)

	followerRepo := &aptest.MockFollowerRepo{
		UpsertFn: func(userID uint64, actorIRI, actorInbox string) (*model.Follower, error) {
			calledUserID = userID
			calledActorIRI = actorIRI
			calledActorInbox = actorInbox
			return &model.Follower{ActorIRI: actorIRI, ActorInbox: actorInbox}, nil
		},
	}

	activity := newActivity(vocab.FollowType, "https://example.com/users/follower")

	ctx := InboxHandlerContext{
		TargetUserID:    42,
		RequestingActor: newActor("https://example.com/users/follower", "https://example.com/users/follower/inbox"),
		FollowerRepo:    followerRepo,
		Activity:        activity,
	}

	handled, err := HandleInboxActivity(ctx)
	require.NoError(t, err)
	assert.True(t, handled)
	assert.Equal(t, uint64(42), calledUserID)
	assert.Equal(t, "https://example.com/users/follower", calledActorIRI)
	assert.Equal(t, "https://example.com/users/follower/inbox", calledActorInbox)
}

func TestHandleFollowActivity_NilActor(t *testing.T) {
	ctx := InboxHandlerContext{
		TargetUserID:    1,
		RequestingActor: nil,
		FollowerRepo:    &aptest.MockFollowerRepo{},
		Activity:        newActivity(vocab.FollowType, "https://example.com/users/follower"),
	}

	handled, err := HandleInboxActivity(ctx)
	require.Error(t, err)
	assert.True(t, handled)
}

// ---------------------------------------------------------------------------
// Undo Follow activity
// ---------------------------------------------------------------------------

func TestHandleUndoFollowActivity(t *testing.T) {
	var (
		calledUserID   uint64
		calledActorIRI string
	)

	followerRepo := &aptest.MockFollowerRepo{
		DeleteFn: func(userID uint64, actorIRI string) error {
			calledUserID = userID
			calledActorIRI = actorIRI
			return nil
		},
	}

	actorIRI := "https://example.com/users/unfollower"

	// Undo { Follow { actor } }
	followActivity := &vocab.Activity{
		Type:   vocab.FollowType,
		Actor:  vocab.IRI(actorIRI),
		Object: vocab.IRI("https://recipient.example.com/users/me"),
	}
	undoActivity := &vocab.Activity{
		Type:   vocab.UndoType,
		Actor:  vocab.IRI(actorIRI),
		Object: followActivity,
	}

	ctx := InboxHandlerContext{
		TargetUserID:    7,
		RequestingActor: newActor(actorIRI, actorIRI+"/inbox"),
		FollowerRepo:    followerRepo,
		Activity:        undoActivity,
	}

	handled, err := HandleInboxActivity(ctx)
	require.NoError(t, err)
	assert.True(t, handled)
	assert.Equal(t, uint64(7), calledUserID)
	assert.Equal(t, actorIRI, calledActorIRI)
}

// ---------------------------------------------------------------------------
// Like activity
// ---------------------------------------------------------------------------

func TestHandleLikeActivity_CallsWorkoutLikeRepo(t *testing.T) {
	const targetWorkoutID = uint64(99)

	var (
		calledWorkoutID uint64
		calledActorIRI  string
	)

	outboxRepo := &aptest.MockOutboxRepo{
		ResolveFn: func(userID uint64, objectOrActivityID string) (uint64, error) {
			return targetWorkoutID, nil
		},
	}
	likeRepo := &aptest.MockWorkoutLikeRepo{
		LikeFn: func(workoutID uint64, actorIRI string) error {
			calledWorkoutID = workoutID
			calledActorIRI = actorIRI
			return nil
		},
	}

	actorIRI := "https://example.com/users/liker"
	activity := &vocab.Activity{
		Type:   vocab.LikeType,
		Actor:  vocab.IRI(actorIRI),
		Object: vocab.IRI("https://example.com/workout/1"),
	}

	ctx := InboxHandlerContext{
		TargetUserID:    1,
		RequestingActor: newActor(actorIRI, actorIRI+"/inbox"),
		OutboxRepo:      outboxRepo,
		WorkoutLikeRepo: likeRepo,
		Activity:        activity,
	}

	handled, err := HandleInboxActivity(ctx)
	require.NoError(t, err)
	assert.True(t, handled)
	assert.Equal(t, targetWorkoutID, calledWorkoutID)
	assert.Equal(t, actorIRI, calledActorIRI)
}

func TestHandleLikeActivity_WorkoutNotFound(t *testing.T) {
	outboxRepo := &aptest.MockOutboxRepo{
		ResolveFn: func(userID uint64, objectOrActivityID string) (uint64, error) {
			return 0, gorm.ErrRecordNotFound
		},
	}
	likeRepo := &aptest.MockWorkoutLikeRepo{
		LikeFn: func(workoutID uint64, actorIRI string) error {
			t.Error("LikeByActorIRI should not be called when workout is not found")
			return nil
		},
	}

	actorIRI := "https://example.com/users/liker"
	activity := &vocab.Activity{
		Type:   vocab.LikeType,
		Actor:  vocab.IRI(actorIRI),
		Object: vocab.IRI("https://example.com/workout/999"),
	}

	ctx := InboxHandlerContext{
		TargetUserID:    1,
		RequestingActor: newActor(actorIRI, actorIRI+"/inbox"),
		OutboxRepo:      outboxRepo,
		WorkoutLikeRepo: likeRepo,
		Activity:        activity,
	}

	handled, err := HandleInboxActivity(ctx)
	require.NoError(t, err)
	assert.True(t, handled)
}

// ---------------------------------------------------------------------------
// Undo Like activity
// ---------------------------------------------------------------------------

func TestHandleUndoLikeActivity(t *testing.T) {
	const targetWorkoutID = uint64(55)

	var (
		calledWorkoutID uint64
		calledActorIRI  string
	)

	outboxRepo := &aptest.MockOutboxRepo{
		ResolveFn: func(userID uint64, objectOrActivityID string) (uint64, error) {
			return targetWorkoutID, nil
		},
	}
	likeRepo := &aptest.MockWorkoutLikeRepo{
		UnlikeFn: func(workoutID uint64, actorIRI string) error {
			calledWorkoutID = workoutID
			calledActorIRI = actorIRI
			return nil
		},
	}

	actorIRI := "https://example.com/users/unliker"
	likeActivity := &vocab.Activity{
		Type:   vocab.LikeType,
		Actor:  vocab.IRI(actorIRI),
		Object: vocab.IRI("https://example.com/workout/1"),
	}
	undoActivity := &vocab.Activity{
		Type:   vocab.UndoType,
		Actor:  vocab.IRI(actorIRI),
		Object: likeActivity,
	}

	ctx := InboxHandlerContext{
		TargetUserID:    1,
		RequestingActor: newActor(actorIRI, actorIRI+"/inbox"),
		OutboxRepo:      outboxRepo,
		WorkoutLikeRepo: likeRepo,
		Activity:        undoActivity,
	}

	handled, err := HandleInboxActivity(ctx)
	require.NoError(t, err)
	assert.True(t, handled)
	assert.Equal(t, targetWorkoutID, calledWorkoutID)
	assert.Equal(t, actorIRI, calledActorIRI)
}

// ---------------------------------------------------------------------------
// Accept / Reject Follow lifecycle activities
//
// Note: AcceptType and RejectType are members of vocab.ReactionsActivityTypes,
// so HandleInboxActivity routes them through routeReactionActivity first.
// Since routeReactionActivity only handles LikeType, Accept/Reject activities
// return (false, nil) from HandleInboxActivity. The handleFollowLifecycleActivity
// function is tested directly below to verify its own logic.
// ---------------------------------------------------------------------------

func TestHandleInboxActivity_Accept_NotHandled(t *testing.T) {
	remoteIRI := "https://remote.example.com/users/them"

	followActivity := &vocab.Activity{
		Type:   vocab.FollowType,
		Actor:  vocab.IRI(remoteIRI),
		Object: vocab.IRI("https://example.com/users/me"),
	}
	acceptActivity := &vocab.Activity{
		Type:   vocab.AcceptType,
		Actor:  vocab.IRI(remoteIRI),
		Object: followActivity,
	}

	ctx := InboxHandlerContext{
		TargetUserID:    1,
		RequestingActor: newActor(remoteIRI, remoteIRI+"/inbox"),
		FollowerRepo:    &aptest.MockFollowerRepo{},
		Activity:        acceptActivity,
	}

	// AcceptType is in vocab.ReactionsActivityTypes, so it is dispatched to
	// routeReactionActivity which returns (false, nil) for non-Like reactions.
	handled, err := HandleInboxActivity(ctx)
	require.NoError(t, err)
	assert.False(t, handled)
}

func TestHandleFollowLifecycleActivity_Accept(t *testing.T) {
	var approvedActorIRI string

	followerRepo := &aptest.MockFollowerRepo{
		ApproveFn: func(userID uint64, actorIRI string) (*model.Follower, error) {
			approvedActorIRI = actorIRI
			return &model.Follower{ActorIRI: actorIRI}, nil
		},
	}

	myIRI := "https://example.com/users/me"
	remoteIRI := "https://remote.example.com/users/them"

	// Accept { Follow { actor: remoteIRI, object: myIRI } }
	followActivity := &vocab.Activity{
		Type:   vocab.FollowType,
		Actor:  vocab.IRI(remoteIRI),
		Object: vocab.IRI(myIRI),
	}
	acceptActivity := &vocab.Activity{
		Type:   vocab.AcceptType,
		Actor:  vocab.IRI(remoteIRI),
		Object: followActivity,
	}

	ctx := InboxHandlerContext{
		TargetUserID:    1,
		RequestingActor: newActor(remoteIRI, remoteIRI+"/inbox"),
		FollowerRepo:    followerRepo,
		Activity:        acceptActivity,
	}

	require.NoError(t, handleFollowLifecycleActivity(ctx))
	assert.Equal(t, myIRI, approvedActorIRI)
}

func TestHandleFollowLifecycleActivity_Reject(t *testing.T) {
	var rejectedActorIRI string

	followerRepo := &aptest.MockFollowerRepo{
		RejectFn: func(userID uint64, actorIRI string) (*model.Follower, error) {
			rejectedActorIRI = actorIRI
			return &model.Follower{ActorIRI: actorIRI}, nil
		},
	}

	myIRI := "https://example.com/users/me"
	remoteIRI := "https://remote.example.com/users/them"

	followActivity := &vocab.Activity{
		Type:   vocab.FollowType,
		Actor:  vocab.IRI(remoteIRI),
		Object: vocab.IRI(myIRI),
	}
	rejectActivity := &vocab.Activity{
		Type:   vocab.RejectType,
		Actor:  vocab.IRI(remoteIRI),
		Object: followActivity,
	}

	ctx := InboxHandlerContext{
		TargetUserID:    1,
		RequestingActor: newActor(remoteIRI, remoteIRI+"/inbox"),
		FollowerRepo:    followerRepo,
		Activity:        rejectActivity,
	}

	require.NoError(t, handleFollowLifecycleActivity(ctx))
	assert.Equal(t, myIRI, rejectedActorIRI)
}

// ---------------------------------------------------------------------------
// Create Reply activity
// ---------------------------------------------------------------------------

func TestHandleCreateReplyActivity(t *testing.T) {
	const targetWorkoutID = uint64(12)

	var (
		calledObjectIRI string
		calledActorIRI  string
		calledActorName string
		calledContent   string
	)

	outboxRepo := &aptest.MockOutboxRepo{
		ResolveFn: func(userID uint64, objectOrActivityID string) (uint64, error) {
			return targetWorkoutID, nil
		},
	}
	replyRepo := &aptest.MockWorkoutReplyRepo{
		ReplyFn: func(workoutID uint64, objectIRI, actorIRI, actorName, content string) error {
			calledObjectIRI = objectIRI
			calledActorIRI = actorIRI
			calledActorName = actorName
			calledContent = content
			return nil
		},
	}

	actorIRI := "https://example.com/users/commenter"
	actor := newActor(actorIRI, actorIRI+"/inbox")
	actor.Name = vocab.DefaultNaturalLanguage("Commenter Name")

	replyNote := vocab.ObjectNew(vocab.NoteType)
	replyNote.ID = "https://example.com/notes/42"
	replyNote.Content = vocab.DefaultNaturalLanguage("Great workout!")
	replyNote.InReplyTo = vocab.IRI("https://example.com/workout/outbox/1")

	createActivity := &vocab.Activity{
		Type:   vocab.CreateType,
		Actor:  vocab.IRI(actorIRI),
		Object: replyNote,
	}

	ctx := InboxHandlerContext{
		TargetUserID:    1,
		RequestingActor: actor,
		OutboxRepo:      outboxRepo,
		WorkoutReplyRepo: replyRepo,
		Activity:        createActivity,
	}

	handled, err := HandleInboxActivity(ctx)
	require.NoError(t, err)
	assert.True(t, handled)
	assert.Equal(t, "https://example.com/notes/42", calledObjectIRI)
	assert.Equal(t, actorIRI, calledActorIRI)
	assert.Equal(t, "Commenter Name", calledActorName)
	assert.Equal(t, "Great workout!", calledContent)
}

// ---------------------------------------------------------------------------
// Delete Reply activity
// ---------------------------------------------------------------------------

func TestHandleDeleteReplyActivity(t *testing.T) {
	const targetWorkoutID = uint64(7)

	var (
		calledObjectIRI string
	)

	replyRepo := &aptest.MockWorkoutReplyRepo{
		ResolveByObjFn: func(objectIRI string) (uint64, error) {
			return targetWorkoutID, nil
		},
		DeleteFn: func(workoutID uint64, objectIRI string) error {
			calledObjectIRI = objectIRI
			return nil
		},
	}

	deleteActivity := &vocab.Activity{
		Type:   vocab.DeleteType,
		Actor:  vocab.IRI("https://example.com/users/commenter"),
		Object: vocab.IRI("https://example.com/notes/42"),
	}

	ctx := InboxHandlerContext{
		TargetUserID:     1,
		WorkoutReplyRepo: replyRepo,
		Activity:         deleteActivity,
	}

	handled, err := HandleInboxActivity(ctx)
	require.NoError(t, err)
	assert.True(t, handled)
	assert.Equal(t, "https://example.com/notes/42", calledObjectIRI)
}

func TestHandleDeleteReplyActivity_WorkoutNotFound(t *testing.T) {
	replyRepo := &aptest.MockWorkoutReplyRepo{
		ResolveByObjFn: func(objectIRI string) (uint64, error) {
			return 0, gorm.ErrRecordNotFound
		},
		DeleteFn: func(workoutID uint64, objectIRI string) error {
			t.Error("DeleteReplyByObjectIRI should not be called when workout is not found")
			return nil
		},
	}

	deleteActivity := &vocab.Activity{
		Type:   vocab.DeleteType,
		Object: vocab.IRI("https://example.com/notes/nonexistent"),
	}

	ctx := InboxHandlerContext{
		TargetUserID:     1,
		WorkoutReplyRepo: replyRepo,
		Activity:         deleteActivity,
	}

	handled, err := HandleInboxActivity(ctx)
	require.NoError(t, err)
	assert.True(t, handled)
}

// ---------------------------------------------------------------------------
// Unknown activity type
// ---------------------------------------------------------------------------

func TestHandleInboxActivity_UnknownType(t *testing.T) {
	activity := &vocab.Activity{
		Type:  vocab.AnnounceType,
		Actor: vocab.IRI("https://example.com/users/someone"),
	}

	ctx := InboxHandlerContext{
		TargetUserID: 1,
		Activity:     activity,
	}

	handled, err := HandleInboxActivity(ctx)
	require.NoError(t, err)
	assert.False(t, handled)
}

// ---------------------------------------------------------------------------
// Repo error propagation
// ---------------------------------------------------------------------------

func TestHandleFollowActivity_RepoError(t *testing.T) {
	expectedErr := errors.New("db connection failed")

	followerRepo := &aptest.MockFollowerRepo{
		UpsertFn: func(userID uint64, actorIRI, actorInbox string) (*model.Follower, error) {
			return nil, expectedErr
		},
	}

	actorIRI := "https://example.com/users/follower"
	ctx := InboxHandlerContext{
		TargetUserID:    1,
		RequestingActor: newActor(actorIRI, actorIRI+"/inbox"),
		FollowerRepo:    followerRepo,
		Activity:        newActivity(vocab.FollowType, actorIRI),
	}

	_, err := HandleInboxActivity(ctx)
	require.ErrorIs(t, err, expectedErr)
}
