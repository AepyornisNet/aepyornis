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
	"github.com/jovandeginste/workout-tracker/v2/pkg/model/dto"
	"github.com/labstack/echo/v4"
)

type _swaggerApInboxErrorResponse = dto.Response[any]

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

	user, err := model.GetUser(ac.context.GetDB(), username)
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

func actorInboxIRI(actor *vocab.Actor) string {
	if actor == nil || vocab.IsNil(actor.Inbox) {
		return ""
	}

	if vocab.IsIRI(actor.Inbox) {
		return actor.Inbox.GetLink().String()
	}

	var iri string
	_ = vocab.OnLink(actor.Inbox, func(link *vocab.Link) error {
		iri = link.Href.String()
		return nil
	})

	return iri
}

func isUndoFollowActivity(it vocab.Activity) bool {
	if !vocab.UndoType.Match(it.GetType()) {
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

	var it vocab.Activity
	err = jsonld.Unmarshal(payload, &it)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, fmt.Errorf("failed to parse JSON-LD: %w", err))
	}

	actor, err := requestingActor(c)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	switch it.GetType() {
	case vocab.FollowType:
		_, err := model.UpsertFollowerRequest(
			ac.context.GetDB(),
			targetUser.ID,
			actor.ID.String(),
			actorInboxIRI(actor),
		)
		if err != nil {
			return renderApiError(c, http.StatusInternalServerError, err)
		}

		return c.NoContent(http.StatusAccepted)
	case vocab.UndoType:
		if !isUndoFollowActivity(it) {
			return c.NoContent(http.StatusNotImplemented)
		}

		err := model.DeleteFollowerByActorIRI(ac.context.GetDB(), targetUser.ID, actor.ID.String())
		if err != nil {
			return renderApiError(c, http.StatusInternalServerError, err)
		}

		return c.NoContent(http.StatusAccepted)
	default:
		return c.NoContent(http.StatusNotImplemented)
	}
}
