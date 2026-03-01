package controller

import (
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

func requestingActor(c echo.Context) (*vocab.Actor, error) {
	actor, ok := c.Get(ap.RequestingActorContextKey).(*vocab.Actor)
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
			RequestingActor:  actor,
			FollowerRepo:     ac.context.FollowerRepo(),
			OutboxRepo:       ac.context.APOutboxRepo(),
			WorkoutLikeRepo:  ac.context.WorkoutLikeRepo(),
			WorkoutReplyRepo: ac.context.WorkoutReplyRepo(),
			Activity:         act,
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

	return c.NoContent(http.StatusAccepted)
}
