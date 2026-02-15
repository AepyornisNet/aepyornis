package controller

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/labstack/echo/v4"
)

type ApUserController interface {
	GetUser(c echo.Context) error
	Inbox(c echo.Context) error
	Outbox(c echo.Context) error
	Following(c echo.Context) error
	Followers(c echo.Context) error
}

type apUserController struct {
	context *container.Container
}

func NewApUserController(c *container.Container) ApUserController {
	return &apUserController{context: c}
}

func (ac *apUserController) GetUser(c echo.Context) error {
	username := c.Param("username")
	if username == "" {
		return renderApiError(c, http.StatusNotFound, errors.New("username not found"))
	}

	user, err := model.GetUser(ac.context.GetDB(), username)
	if err != nil || !user.ActivityPubEnabled() {
		return renderApiError(c, http.StatusNotFound, errors.New("resource not found"))
	}

	actorPath := strings.TrimSuffix(c.Request().URL.Path, "/")
	actorURL := fmt.Sprintf("%s://%s%s", c.Scheme(), c.Request().Host, actorPath)

	person := vocab.Person{
		Type:              vocab.PersonType,
		ID:                vocab.ID(actorURL),
		Name:              vocab.DefaultNaturalLanguage(user.Name),
		PreferredUsername: vocab.DefaultNaturalLanguage(user.Username),
		Inbox:             vocab.IRI(actorURL + "/inbox"),
		Outbox:            vocab.IRI(actorURL + "/outbox"),
		Following:         vocab.IRI(actorURL + "/following"),
		Followers:         vocab.IRI(actorURL + "/followers"),
	}

	if user.PublicKey != "" {
		person.PublicKey = vocab.PublicKey{
			ID:           vocab.ID(actorURL + "#main-key"),
			Owner:        vocab.IRI(actorURL),
			PublicKeyPem: user.PublicKey,
		}
	}

	resp, err := jsonld.WithContext(
		jsonld.IRI(vocab.ActivityBaseURI),
		jsonld.IRI(vocab.SecurityContextURI),
	).Marshal(person)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, fmt.Errorf("failed to marshal profile: %w", err))
	}

	return renderActivityPubResponse(c, http.StatusOK, resp)
}

func (ac *apUserController) Inbox(c echo.Context) error {
	var it vocab.Activity

	payload, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, fmt.Errorf("failed to read request body: %w", err))
	} else {
		err = jsonld.Unmarshal(payload, &it)
		if err != nil {
			return renderApiError(c, http.StatusBadRequest, fmt.Errorf("failed to parse JSON-LD: %w", err))
		}
	}

	fmt.Println("Received activity:", it.GetType(), "from", it.Actor)
	fmt.Println(it)
	fmt.Println(c.Request().Header)
	fmt.Println(string(payload))

	return c.NoContent(http.StatusNotImplemented)
}

func (ac *apUserController) Outbox(c echo.Context) error {
	// Log the request for debugging purposes
	fmt.Printf("Received request for Outbox: %s\n", c.Request().URL.Path)

	return c.NoContent(http.StatusNotImplemented)
}

func (ac *apUserController) Following(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

func (ac *apUserController) Followers(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}
