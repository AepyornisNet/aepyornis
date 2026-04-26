package aputil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	vocab "github.com/go-ap/activitypub"
	"gorm.io/gorm"
)

type InboxFollowerRepository interface {
	UpsertFollowerRequest(profileID uint64, followingProfile *model.Profile) (*model.Follower, error)
	MarkFollowingApprovedByURL(profileID uint64, followingProfileURL string) (*model.Follower, error)
	MarkFollowingRejectedByURL(profileID uint64, followingProfileURL string) (*model.Follower, error)
	DeleteFollowerByURL(profileID uint64, followingProfileURL string) error
}

type InboxOutboxRepository interface {
	ResolveWorkoutIDByObjectOrActivityID(userID uint64, objectOrActivityID string) (uint64, error)
}

type InboxWorkoutLikeRepository interface {
	LikeByActorIRI(workoutID uint64, actorIRI string) error
	UnlikeByActorIRI(workoutID uint64, actorIRI string) error
}

type InboxWorkoutReplyRepository interface {
	ReplyByActorIRI(workoutID uint64, objectIRI, actorIRI, actorName, content string) error
	UpdateReplyByObjectIRI(workoutID uint64, objectIRI, actorIRI, actorName, content string) error
	DeleteReplyByObjectIRI(workoutID uint64, objectIRI string) error
	ResolveWorkoutIDByObjectIRI(objectIRI string) (uint64, error)
}

type InboxStatusRepository interface {
	ResolveWorkoutIDByObjectOrActivityID(userID uint64, objectOrActivityID string) (uint64, error)
	ResolveWorkoutIDByObjectIRI(objectIRI string) (uint64, error)
	ReplyByActorIRI(workoutID uint64, objectIRI, actorIRI, actorName, content string) error
	UpdateReplyByObjectIRI(workoutID uint64, objectIRI, actorIRI, actorName, content string) error
	DeleteReplyByObjectIRI(workoutID uint64, objectIRI string) error
	UpsertRemoteWorkoutStatus(actorIRI, actorName, activityID, objectID, content string, activityJSON, payloadJSON []byte) error
}

type InboxProfileService interface {
	GetByActorIRI(ctx context.Context, actorIRI string) (*model.Profile, error)
}

type InboxActivityHandler struct {
	followerRepo     InboxFollowerRepository
	outboxRepo       InboxOutboxRepository
	workoutLikeRepo  InboxWorkoutLikeRepository
	workoutReplyRepo InboxWorkoutReplyRepository
	statusRepo       InboxStatusRepository
	profileService   InboxProfileService
}

func NewInboxActivityHandler(
	followerRepo InboxFollowerRepository,
	outboxRepo InboxOutboxRepository,
	workoutLikeRepo InboxWorkoutLikeRepository,
	workoutReplyRepo InboxWorkoutReplyRepository,
	statusRepo InboxStatusRepository,
	profileService InboxProfileService,
) *InboxActivityHandler {
	return &InboxActivityHandler{
		followerRepo:     followerRepo,
		outboxRepo:       outboxRepo,
		workoutLikeRepo:  workoutLikeRepo,
		workoutReplyRepo: workoutReplyRepo,
		statusRepo:       statusRepo,
		profileService:   profileService,
	}
}

func (h *InboxActivityHandler) HandleActivity(ctx context.Context, requestingActor *vocab.Actor, targetUserID uint64, targetProfileID uint64, activity *vocab.Activity) (bool, error) {
	if activity == nil {
		return false, nil
	}

	switch activity.GetType() {
	case vocab.FollowType:
		return true, h.handleFollowActivity(ctx, requestingActor, targetProfileID)
	case vocab.AcceptType:
		return true, h.handleFollowAcceptActivity(targetProfileID, activity)
	case vocab.RejectType:
		return true, h.handleFollowRejectActivity(targetProfileID, activity)
	case vocab.CreateType:
		return h.createActivity(requestingActor, targetUserID, activity)
	case vocab.UpdateType:
		return h.updateActivity(requestingActor, targetUserID, activity)
	case vocab.DeleteType, vocab.RemoveType:
		return h.deleteActivity(activity)
	case vocab.UndoType:
		return h.undoActivity(requestingActor, targetUserID, targetProfileID, activity)
	case vocab.LikeType:
		return true, h.handleLikeActivity(requestingActor, targetUserID, activity)
	default:
		return false, nil
	}
}

func (h *InboxActivityHandler) handleFollowActivity(ctx context.Context, requestingActor *vocab.Actor, targetProfileID uint64) error {
	if requestingActor == nil {
		return errors.New("requesting actor invalid")
	}

	actorIRI := strings.TrimSpace(requestingActor.ID.String())
	if actorIRI == "" {
		return errors.New("requesting actor profile invalid")
	}

	profile, err := h.profileService.GetByActorIRI(ctx, actorIRI)
	if err != nil {
		return err
	}

	_, err = h.followerRepo.UpsertFollowerRequest(
		targetProfileID,
		profile,
	)

	return err
}

func (h *InboxActivityHandler) handleFollowAcceptActivity(targetProfileID uint64, activity *vocab.Activity) error {
	followTargetIRI := extractFollowLifecycleTarget(activity)
	if followTargetIRI == "" {
		return errors.New("invalid target follower given")
	}

	_, err := h.followerRepo.MarkFollowingApprovedByURL(targetProfileID, followTargetIRI)
	if err != nil {
		return fmt.Errorf("failed writing follower approve: %w", err)
	}

	return nil
}

func (h *InboxActivityHandler) handleFollowRejectActivity(targetProfileID uint64, activity *vocab.Activity) error {
	followTargetIRI := extractFollowLifecycleTarget(activity)
	if followTargetIRI == "" {
		return errors.New("invalid target follower given")
	}

	_, err := h.followerRepo.MarkFollowingRejectedByURL(targetProfileID, followTargetIRI)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed writing follower reject: %w", err)
	}

	return nil
}

func (h *InboxActivityHandler) handleLikeActivity(requestingActor *vocab.Actor, targetUserID uint64, activity *vocab.Activity) error {
	if requestingActor == nil {
		return errors.New("requesting actor invalid")
	}

	targetObjectIRI := activityObjectIRI(activity)
	if targetObjectIRI == "" {
		return nil
	}

	workoutID, resolveErr := h.outboxRepo.ResolveWorkoutIDByObjectOrActivityID(targetUserID, targetObjectIRI)
	if resolveErr != nil {
		if errors.Is(resolveErr, gorm.ErrRecordNotFound) {
			return nil
		}

		return resolveErr
	}

	return h.workoutLikeRepo.LikeByActorIRI(workoutID, requestingActor.ID.String())
}

func (h *InboxActivityHandler) handleUndoFollowActivity(requestingActor *vocab.Actor, targetProfileID uint64) error {
	if requestingActor == nil {
		return errors.New("requesting actor invalid")
	}

	return h.followerRepo.DeleteFollowerByURL(targetProfileID, requestingActor.ID.String())
}

func (h *InboxActivityHandler) handleUndoLikeActivity(requestingActor *vocab.Actor, targetUserID uint64, activity *vocab.Activity) error {
	if requestingActor == nil {
		return errors.New("requesting actor invalid")
	}

	targetObjectIRI := activityObjectIRI(activity)
	if targetObjectIRI == "" {
		return nil
	}

	workoutID, resolveErr := h.outboxRepo.ResolveWorkoutIDByObjectOrActivityID(targetUserID, targetObjectIRI)
	if resolveErr != nil {
		if errors.Is(resolveErr, gorm.ErrRecordNotFound) {
			return nil
		}

		return resolveErr
	}

	return h.workoutLikeRepo.UnlikeByActorIRI(workoutID, requestingActor.ID.String())
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

func (h *InboxActivityHandler) createActivity(requestingActor *vocab.Actor, targetUserID uint64, activity *vocab.Activity) (bool, error) {
	if isCreateReplyActivity(activity) {
		return true, h.handleCreateReplyActivity(requestingActor, targetUserID, activity)
	}

	if isCreateWorkoutActivity(activity) {
		return true, h.handleCreateWorkoutActivity(requestingActor, activity)
	}

	return false, nil
}

func (h *InboxActivityHandler) updateActivity(requestingActor *vocab.Actor, targetUserID uint64, activity *vocab.Activity) (bool, error) {
	if isUpdateReplyActivity(activity) {
		return true, h.handleUpdateReplyActivity(requestingActor, targetUserID, activity)
	}

	return false, nil
}

func (h *InboxActivityHandler) deleteActivity(activity *vocab.Activity) (bool, error) {
	if isDeleteReplyActivity(activity) {
		return true, h.handleDeleteReplyActivity(activity)
	}

	return false, nil
}

func (h *InboxActivityHandler) undoActivity(requestingActor *vocab.Actor, targetUserID uint64, targetProfileID uint64, activity *vocab.Activity) (bool, error) {
	obj, err := vocab.ToActivity(activity.Object)
	if err != nil {
		return false, err
	}

	switch obj.GetType() {
	case vocab.FollowType:
		return true, h.handleUndoFollowActivity(requestingActor, targetProfileID)
	case vocab.LikeType:
		return true, h.handleUndoLikeActivity(requestingActor, targetUserID, obj)
	}

	return false, nil
}

func isCreateReplyActivity(it *vocab.Activity) bool {
	if it == nil || !vocab.CreateType.Match(it.GetType()) {
		return false
	}

	hasReply := false
	_ = vocab.OnObject(it.Object, func(obj *vocab.Object) error {
		if !vocab.IsNil(obj.InReplyTo) {
			hasReply = true
		}

		return nil
	})

	return hasReply
}

func isUpdateReplyActivity(it *vocab.Activity) bool {
	if it == nil || !vocab.UpdateType.Match(it.GetType()) {
		return false
	}

	replyObjectIRI, content, inReplyToIRI := extractReplyInfo(it)
	if replyObjectIRI == "" {
		return false
	}

	return content != "" || inReplyToIRI != ""
}

func isDeleteReplyActivity(it *vocab.Activity) bool {
	if it == nil {
		return false
	}

	if !(vocab.DeleteType.Match(it.GetType()) || vocab.RemoveType.Match(it.GetType())) {
		return false
	}

	return extractDeleteTargetObjectIRI(it) != ""
}

func (h *InboxActivityHandler) handleCreateReplyActivity(requestingActor *vocab.Actor, targetUserID uint64, activity *vocab.Activity) error {
	if requestingActor == nil {
		return errors.New("requesting actor invalid")
	}

	replyObjectIRI, content, inReplyToIRI := extractReplyInfo(activity)
	if replyObjectIRI == "" || inReplyToIRI == "" {
		return nil
	}

	workoutID, resolveErr := h.outboxRepo.ResolveWorkoutIDByObjectOrActivityID(targetUserID, inReplyToIRI)
	if resolveErr != nil {
		if errors.Is(resolveErr, gorm.ErrRecordNotFound) {
			return nil
		}

		return resolveErr
	}

	// Get actor name from the requesting actor
	actorName := ""
	if requestingActor.Name != nil {
		actorName = requestingActor.Name.String()
	}

	if h.statusRepo != nil {
		if err := h.statusRepo.ReplyByActorIRI(workoutID, replyObjectIRI, requestingActor.ID.String(), actorName, content); err != nil {
			return err
		}
	}

	return h.workoutReplyRepo.ReplyByActorIRI(workoutID, replyObjectIRI, requestingActor.ID.String(), actorName, content)
}

func (h *InboxActivityHandler) handleUpdateReplyActivity(requestingActor *vocab.Actor, targetUserID uint64, activity *vocab.Activity) error {
	if requestingActor == nil {
		return errors.New("requesting actor invalid")
	}

	replyObjectIRI, content, inReplyToIRI := extractReplyInfo(activity)
	if replyObjectIRI == "" || content == "" {
		return nil
	}

	var workoutID uint64
	if inReplyToIRI != "" {
		resolvedWorkoutID, resolveErr := h.outboxRepo.ResolveWorkoutIDByObjectOrActivityID(targetUserID, inReplyToIRI)
		if resolveErr != nil {
			if errors.Is(resolveErr, gorm.ErrRecordNotFound) {
				return nil
			}

			return resolveErr
		}

		workoutID = resolvedWorkoutID
	} else {
		resolvedWorkoutID, resolveErr := h.workoutReplyRepo.ResolveWorkoutIDByObjectIRI(replyObjectIRI)
		if resolveErr != nil {
			if errors.Is(resolveErr, gorm.ErrRecordNotFound) {
				return nil
			}

			return resolveErr
		}

		workoutID = resolvedWorkoutID
	}

	actorName := ""
	if requestingActor.Name != nil {
		actorName = requestingActor.Name.String()
	}

	err := h.workoutReplyRepo.UpdateReplyByObjectIRI(workoutID, replyObjectIRI, requestingActor.ID.String(), actorName, content)
	if h.statusRepo != nil {
		statusErr := h.statusRepo.UpdateReplyByObjectIRI(workoutID, replyObjectIRI, requestingActor.ID.String(), actorName, content)
		if statusErr != nil && !errors.Is(statusErr, gorm.ErrRecordNotFound) {
			return statusErr
		}
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	return err
}

func (h *InboxActivityHandler) handleDeleteReplyActivity(activity *vocab.Activity) error {
	targetObjectIRI := extractDeleteTargetObjectIRI(activity)
	if targetObjectIRI == "" {
		return nil
	}

	workoutID, resolveErr := h.workoutReplyRepo.ResolveWorkoutIDByObjectIRI(targetObjectIRI)
	if resolveErr != nil {
		if errors.Is(resolveErr, gorm.ErrRecordNotFound) {
			return nil
		}

		return resolveErr
	}

	err := h.workoutReplyRepo.DeleteReplyByObjectIRI(workoutID, targetObjectIRI)
	if h.statusRepo != nil {
		statusErr := h.statusRepo.DeleteReplyByObjectIRI(workoutID, targetObjectIRI)
		if statusErr != nil && !errors.Is(statusErr, gorm.ErrRecordNotFound) {
			return statusErr
		}
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	return err
}

func extractReplyInfo(it *vocab.Activity) (objectIRI, content, inReplyToIRI string) {
	if it == nil || vocab.IsNil(it.Object) {
		return "", "", ""
	}

	_ = vocab.OnObject(it.Object, func(obj *vocab.Object) error {
		objectIRI = obj.ID.String()
		if objectIRI == "" {
			objectIRI = itemIRIString(obj.URL)
		}
		if obj.Content != nil {
			content = obj.Content.String()
		}

		if !vocab.IsNil(obj.InReplyTo) {
			inReplyToIRI = obj.InReplyTo.GetLink().String()
		}

		return nil
	})

	return
}

func extractDeleteTargetObjectIRI(it *vocab.Activity) string {
	if it == nil || vocab.IsNil(it.Object) {
		return ""
	}

	target := itemIRIString(it.Object)
	if target != "" {
		return target
	}

	_ = vocab.OnObject(it.Object, func(obj *vocab.Object) error {
		target = obj.ID.String()
		if target == "" {
			target = itemIRIString(obj.URL)
		}

		return nil
	})

	return target
}

func isCreateWorkoutActivity(it *vocab.Activity) bool {
	if it == nil || !vocab.CreateType.Match(it.GetType()) {
		return false
	}

	if isCreateReplyActivity(it) {
		return false
	}

	isNote := false
	_ = vocab.OnObject(it.Object, func(obj *vocab.Object) error {
		if vocab.NoteType.Match(obj.GetType()) {
			isNote = true
		}

		return nil
	})

	if isNote {
		return true
	}

	return false
}

func (h *InboxActivityHandler) handleCreateWorkoutActivity(requestingActor *vocab.Actor, activity *vocab.Activity) error {
	if requestingActor == nil || h.statusRepo == nil {
		return nil
	}

	objectIRI, content, _ := extractReplyInfo(activity)
	if objectIRI == "" {
		return nil
	}

	activityJSON, err := json.Marshal(activity)
	if err != nil {
		return err
	}

	objectPayload, objectRaw, ok := extractCreateObjectPayload(activity)
	if !ok {
		return nil
	}

	if content == "" {
		content = objectPayload
	}

	actorName := ""
	if requestingActor.Name != nil {
		actorName = requestingActor.Name.String()
	}

	return h.statusRepo.UpsertRemoteWorkoutStatus(
		requestingActor.ID.String(),
		actorName,
		firstNonEmpty(activity.ID.String(), objectIRI),
		objectIRI,
		content,
		activityJSON,
		objectRaw,
	)
}

func extractCreateObjectPayload(it *vocab.Activity) (string, []byte, bool) {
	if it == nil || vocab.IsNil(it.Object) {
		return "", nil, false
	}

	var objectJSON []byte
	if err := vocab.OnObject(it.Object, func(obj *vocab.Object) error {
		raw, marshalErr := json.Marshal(obj)
		if marshalErr != nil {
			return marshalErr
		}

		objectJSON = raw
		return nil
	}); err != nil {
		return "", nil, false
	}

	if len(objectJSON) == 0 {
		return "", nil, false
	}

	return string(objectJSON), objectJSON, true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}
