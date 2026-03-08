package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
	ap "github.com/jovandeginste/workout-tracker/v2/pkg/activitypub"
	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/labstack/echo/v4"
)

type ApInboxController interface {
	Inbox(c echo.Context) error
}

type apInboxController struct {
	context *container.Container
}

func NewApInboxController(c *container.Container) ApInboxController {
	return &apInboxController{context: c}
}

func (ac *apInboxController) targetActivityPubUser(c echo.Context) (*model.User, error) {
	username := c.Param("username")
	if username == "" {
		return nil, errors.New("username not found")
	}

	user, err := ac.context.UserRepo().GetByUsername(username)
	if err != nil || !user.ActivityPubEnabled() {
		return nil, errors.New("resource not found")
	}

	return user, nil
}

func requestingActor(c echo.Context) (*ap.RequestActor, error) {
	actor, ok := c.Get(ap.RequestingActorContextKey).(*ap.RequestActor)
	if ok && actor != nil {
		return actor, nil
	}

	return nil, errors.New("requesting actor invalid")
}

// Inbox handles incoming ActivityPub activities for a local user inbox
// @Summary      Post ActivityPub inbox activity
// @Tags         activity-pub
// @Param        username  path  string  true  "Username"
// @Accept       json
// @Success      202  {string}  string
// @Failure      400  {object}  dto.Response[any]
// @Failure      404  {object}  dto.Response[any]
// @Failure      500  {object}  dto.Response[any]
// @Router       /ap/users/{username}/inbox [post]
func (ac *apInboxController) Inbox(c echo.Context) error {
	targetUser, err := ac.targetActivityPubUser(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	payload, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, fmt.Errorf("failed to read request body: %w", err))
	}

	var activity vocab.Activity
	err = jsonld.Unmarshal(payload, &activity)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, fmt.Errorf("failed to parse JSON-LD: %w", err))
	}

	actor, err := requestingActor(c)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	var item vocab.Item = activity
	handled := false

	err = vocab.On[vocab.Activity](item, func(act *vocab.Activity) error {
		routed, routeErr := ap.HandleInboxActivity(ap.InboxHandlerContext{
			TargetUserID:     targetUser.ID,
			RequestingActor:  &actor.Actor,
			FollowerRepo:     ac.context.FollowerRepo(),
			OutboxRepo:       ac.context.APOutboxRepo(),
			WorkoutLikeRepo:  ac.context.WorkoutLikeRepo(),
			WorkoutReplyRepo: ac.context.WorkoutReplyRepo(),
			WorkoutRepo:      ac.context.WorkoutRepo(),
			Activity:         act,
			RawPayload:       payload,
		})
		handled = routed
		return routeErr
	})
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if !handled {
		return c.NoContent(http.StatusNotImplemented)
	}

	// After a follow Accept, sync the accepted user's outbox so that
	// their existing published workouts are pulled in.
	if vocab.AcceptType.Match(activity.GetType()) {
		go ac.syncAcceptedFollowOutbox(targetUser, &actor.Actor)
	}

	return c.NoContent(http.StatusAccepted)
}

// syncAcceptedFollowOutbox fetches the outbox of the remote actor who just
// accepted a follow request and processes each workout Create activity.
// This runs in a goroutine so it doesn't block the inbox response.
func (ac *apInboxController) syncAcceptedFollowOutbox(localUser *model.User, remoteActor *vocab.Actor) {
	if remoteActor == nil || vocab.IsNil(remoteActor.Outbox) {
		return
	}

	outboxIRI := remoteActor.Outbox.GetLink().String()
	if outboxIRI == "" {
		return
	}

	ctx := context.Background()
	logger := ac.context.Logger()

	items, err := fetchOutboxItems(ctx, outboxIRI)
	if err != nil {
		logger.Warn("Failed to fetch remote outbox after follow accept", "outbox", outboxIRI, "error", err)
		return
	}

	for _, raw := range items {
		var activity vocab.Activity
		if err := jsonld.Unmarshal(raw, &activity); err != nil {
			continue
		}

		if !vocab.CreateType.Match(activity.GetType()) {
			continue
		}

		_, _ = ap.HandleInboxActivity(ap.InboxHandlerContext{
			TargetUserID:     localUser.ID,
			RequestingActor:  remoteActor,
			FollowerRepo:     ac.context.FollowerRepo(),
			OutboxRepo:       ac.context.APOutboxRepo(),
			WorkoutLikeRepo:  ac.context.WorkoutLikeRepo(),
			WorkoutReplyRepo: ac.context.WorkoutReplyRepo(),
			WorkoutRepo:      ac.context.WorkoutRepo(),
			Activity:         &activity,
			RawPayload:       raw,
		})
	}

	logger.Info("Synced remote outbox after follow accept", "outbox", outboxIRI, "items", len(items))
}

// fetchOutboxItems fetches the first page of a remote actor's outbox and
// returns the raw JSON of each ordered item.
func fetchOutboxItems(ctx context.Context, outboxIRI string) ([][]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, outboxIRI, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", ap.ContentType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("outbox fetch rejected: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Try to get items directly (some implementations inline them)
	items := extractOrderedItems(body)
	if len(items) > 0 {
		return items, nil
	}

	// Otherwise follow the "first" link for the first page
	firstIRI := extractFirstPageIRI(body)
	if firstIRI == "" {
		return nil, nil
	}

	pageReq, err := http.NewRequestWithContext(ctx, http.MethodGet, firstIRI, nil)
	if err != nil {
		return nil, err
	}

	pageReq.Header.Set("Accept", ap.ContentType)

	pageResp, err := http.DefaultClient.Do(pageReq)
	if err != nil {
		return nil, err
	}
	defer pageResp.Body.Close()

	if pageResp.StatusCode < http.StatusOK || pageResp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("outbox page fetch rejected: %s", pageResp.Status)
	}

	pageBody, err := io.ReadAll(pageResp.Body)
	if err != nil {
		return nil, err
	}

	return extractOrderedItems(pageBody), nil
}

// extractOrderedItems extracts the "orderedItems" array from JSON-LD and
// returns each element as raw JSON bytes.
func extractOrderedItems(body []byte) [][]byte {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil
	}

	itemsRaw, ok := raw["orderedItems"]
	if !ok {
		return nil
	}

	var arr []json.RawMessage
	if err := json.Unmarshal(itemsRaw, &arr); err != nil {
		return nil
	}

	result := make([][]byte, 0, len(arr))
	for _, item := range arr {
		result = append(result, []byte(item))
	}

	return result
}

// extractFirstPageIRI gets the "first" link from an OrderedCollection.
func extractFirstPageIRI(body []byte) string {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return ""
	}

	firstRaw, ok := raw["first"]
	if !ok {
		return ""
	}

	// "first" can be a string IRI or an object with "id"
	var iri string
	if err := json.Unmarshal(firstRaw, &iri); err == nil && iri != "" {
		return iri
	}

	var obj struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(firstRaw, &obj); err == nil {
		return obj.ID
	}

	return ""
}
