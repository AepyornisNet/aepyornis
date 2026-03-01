package activitypub

import (
	"errors"

	vocab "github.com/go-ap/activitypub"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"gorm.io/gorm"
)

type InboxFollowerRepository interface {
	UpsertFollowerRequest(userID uint64, actorIRI, actorInbox string) (*model.Follower, error)
	MarkFollowingApprovedByActorIRI(userID uint64, actorIRI string) (*model.Follower, error)
	MarkFollowingRejectedByActorIRI(userID uint64, actorIRI string) (*model.Follower, error)
	DeleteFollowerByActorIRI(userID uint64, actorIRI string) error
}

type InboxOutboxRepository interface {
	ResolveWorkoutIDByObjectOrActivityID(userID uint64, objectOrActivityID string) (uint64, error)
}

type InboxWorkoutLikeRepository interface {
	LikeByActorIRI(workoutID uint64, actorIRI string) error
	UnlikeByActorIRI(workoutID uint64, actorIRI string) error
}

type InboxHandlerContext struct {
	TargetUserID    uint64
	RequestingActor *vocab.Actor
	FollowerRepo    InboxFollowerRepository
	OutboxRepo      InboxOutboxRepository
	WorkoutLikeRepo InboxWorkoutLikeRepository
	Activity        *vocab.Activity
}

func HandleInboxActivity(ctx InboxHandlerContext) (bool, error) {
	if ctx.Activity == nil {
		return false, nil
	}

	if hasActivityType(vocab.ReactionsActivityTypes, ctx.Activity.GetType()) {
		return routeReactionActivity(ctx)
	}

	switch ctx.Activity.GetType() {
	case vocab.FollowType:
		return true, handleFollowActivity(ctx)
	case vocab.AcceptType, vocab.RejectType:
		return true, handleFollowLifecycleActivity(ctx)
	case vocab.UndoType:
		return routeUndoActivity(ctx)
	default:
		return false, nil
	}
}

func hasActivityType(types vocab.ActivityVocabularyTypes, typ vocab.Typer) bool {
	for _, candidate := range types {
		if candidate.Match(typ) {
			return true
		}
	}

	return false
}

func routeReactionActivity(ctx InboxHandlerContext) (bool, error) {
	switch ctx.Activity.GetType() {
	case vocab.LikeType:
		return true, handleLikeActivity(ctx)
	default:
		return false, nil
	}
}

func routeUndoActivity(ctx InboxHandlerContext) (bool, error) {
	if isUndoFollowActivity(ctx.Activity) {
		return true, handleUndoFollowActivity(ctx)
	}

	if isUndoLikeActivity(ctx.Activity) {
		return true, handleUndoLikeActivity(ctx)
	}

	return false, nil
}

func handleFollowActivity(ctx InboxHandlerContext) error {
	if ctx.RequestingActor == nil {
		return errors.New("requesting actor invalid")
	}

	_, err := ctx.FollowerRepo.UpsertFollowerRequest(
		ctx.TargetUserID,
		ctx.RequestingActor.ID.String(),
		actorInboxIRI(ctx.RequestingActor),
	)

	return err
}

func handleFollowLifecycleActivity(ctx InboxHandlerContext) error {
	followTargetIRI := extractFollowLifecycleTarget(ctx.Activity)
	if followTargetIRI == "" {
		return nil
	}

	var lifecycleErr error
	if vocab.AcceptType.Match(ctx.Activity.GetType()) {
		_, lifecycleErr = ctx.FollowerRepo.MarkFollowingApprovedByActorIRI(ctx.TargetUserID, followTargetIRI)
	} else {
		_, lifecycleErr = ctx.FollowerRepo.MarkFollowingRejectedByActorIRI(ctx.TargetUserID, followTargetIRI)
	}

	if lifecycleErr != nil && !errors.Is(lifecycleErr, gorm.ErrRecordNotFound) {
		return lifecycleErr
	}

	return nil
}

func handleLikeActivity(ctx InboxHandlerContext) error {
	if ctx.RequestingActor == nil {
		return errors.New("requesting actor invalid")
	}

	targetObjectIRI := activityObjectIRI(ctx.Activity)
	if targetObjectIRI == "" {
		return nil
	}

	workoutID, resolveErr := ctx.OutboxRepo.ResolveWorkoutIDByObjectOrActivityID(ctx.TargetUserID, targetObjectIRI)
	if resolveErr != nil {
		if errors.Is(resolveErr, gorm.ErrRecordNotFound) {
			return nil
		}

		return resolveErr
	}

	return ctx.WorkoutLikeRepo.LikeByActorIRI(workoutID, ctx.RequestingActor.ID.String())
}

func handleUndoFollowActivity(ctx InboxHandlerContext) error {
	if ctx.RequestingActor == nil {
		return errors.New("requesting actor invalid")
	}

	return ctx.FollowerRepo.DeleteFollowerByActorIRI(ctx.TargetUserID, ctx.RequestingActor.ID.String())
}

func handleUndoLikeActivity(ctx InboxHandlerContext) error {
	if ctx.RequestingActor == nil {
		return errors.New("requesting actor invalid")
	}

	targetObjectIRI := extractUndoLikeTarget(ctx.Activity)
	if targetObjectIRI == "" {
		return nil
	}

	workoutID, resolveErr := ctx.OutboxRepo.ResolveWorkoutIDByObjectOrActivityID(ctx.TargetUserID, targetObjectIRI)
	if resolveErr != nil {
		if errors.Is(resolveErr, gorm.ErrRecordNotFound) {
			return nil
		}

		return resolveErr
	}

	return ctx.WorkoutLikeRepo.UnlikeByActorIRI(workoutID, ctx.RequestingActor.ID.String())
}

func actorInboxIRI(actor *vocab.Actor) string {
	if actor == nil || vocab.IsNil(actor.Inbox) {
		return ""
	}

	if vocab.IsIRI(actor.Inbox) {
		return actor.Inbox.GetLink().String()
	}

	iri := ""
	_ = vocab.OnLink(actor.Inbox, func(link *vocab.Link) error {
		iri = link.Href.String()
		return nil
	})

	return iri
}

func actorIRIFromItem(item vocab.Item) string {
	if vocab.IsNil(item) {
		return ""
	}

	if vocab.IsIRI(item) {
		return item.GetLink().String()
	}

	actorIRI := ""
	_ = vocab.OnActor(item, func(actor *vocab.Actor) error {
		actorIRI = actor.ID.String()
		return nil
	})

	if actorIRI != "" {
		return actorIRI
	}

	_ = vocab.OnLink(item, func(link *vocab.Link) error {
		actorIRI = link.Href.String()
		return nil
	})

	return actorIRI
}

func extractFollowLifecycleTarget(it *vocab.Activity) string {
	if it == nil || vocab.IsNil(it.Object) || !(vocab.AcceptType.Match(it.GetType()) || vocab.RejectType.Match(it.GetType())) {
		return ""
	}

	targetIRI := ""
	_ = vocab.OnActivity(it.Object, func(obj *vocab.Activity) error {
		if !vocab.FollowType.Match(obj.GetType()) {
			return nil
		}

		targetIRI = actorIRIFromItem(obj.Object)
		return nil
	})

	return targetIRI
}

func isUndoFollowActivity(it *vocab.Activity) bool {
	if it == nil || !vocab.UndoType.Match(it.GetType()) {
		return false
	}

	isFollow := false
	if err := vocab.OnActivity(it.Object, func(object *vocab.Activity) error {
		if vocab.FollowType.Match(object.GetType()) {
			isFollow = true
		}

		return nil
	}); err != nil {
		return false
	}

	return isFollow
}

func isUndoLikeActivity(it *vocab.Activity) bool {
	if it == nil || !vocab.UndoType.Match(it.GetType()) {
		return false
	}

	isLike := false
	if err := vocab.OnActivity(it.Object, func(object *vocab.Activity) error {
		if vocab.LikeType.Match(object.GetType()) {
			isLike = true
		}

		return nil
	}); err != nil {
		return false
	}

	return isLike
}

func activityObjectIRI(it *vocab.Activity) string {
	if it == nil || vocab.IsNil(it.Object) {
		return ""
	}

	return actorIRIFromItem(it.Object)
}

func extractUndoLikeTarget(it *vocab.Activity) string {
	if !isUndoLikeActivity(it) || vocab.IsNil(it.Object) {
		return ""
	}

	targetIRI := ""
	_ = vocab.OnActivity(it.Object, func(obj *vocab.Activity) error {
		if !vocab.LikeType.Match(obj.GetType()) {
			return nil
		}

		targetIRI = activityObjectIRI(obj)
		return nil
	})

	return targetIRI
}
