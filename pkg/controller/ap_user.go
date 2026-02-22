package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
	ap "github.com/jovandeginste/workout-tracker/v2/pkg/activitypub"
	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/labstack/echo/v4"
)

type ApUserController interface {
	GetUser(c echo.Context) error
	Following(c echo.Context) error
	Followers(c echo.Context) error
}

type apUserController struct {
	context *container.Container
}

const followersPageSize = 20

func NewApUserController(c *container.Container) ApUserController {
	return &apUserController{context: c}
}

// GetUser returns the ActivityPub actor for a local user
// @Summary      Get ActivityPub actor
// @Tags         activity-pub
// @Param        username  path  string  true  "Username"
// @Produce      json
// @Success      200  {object}  map[string]any
// @Failure      404  {object}  dto.Response[any]
// @Router       /ap/users/{username} [get]
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
		Published:         user.CreatedAt.UTC(),
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

	return renderActivityPubResponse(c, resp)
}

func (ac *apUserController) targetActivityPubUser(c echo.Context) (*model.User, error) {
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

// Following returns the ActivityPub following collection for a local user
// @Summary      Get ActivityPub following collection
// @Tags         activity-pub
// @Param        username  path   string  true   "Username"
// @Param        page      query  int     false  "Page number (1-based)"
// @Produce      json
// @Success      200  {object}  map[string]any
// @Failure      400  {object}  dto.Response[any]
// @Failure      404  {object}  dto.Response[any]
// @Router       /ap/users/{username}/following [get]
func (ac *apUserController) Following(c echo.Context) error {
	targetUser, err := ac.targetActivityPubUser(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	page := 0
	if rawPage := strings.TrimSpace(c.QueryParam("page")); rawPage != "" {
		page, err = strconv.Atoi(rawPage)
		if err != nil || page < 1 {
			return renderApiError(c, http.StatusBadRequest, errors.New("invalid page"))
		}
	}

	following, err := model.ListApprovedFollowing(ac.context.GetDB(), targetUser.ID)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	items := make(vocab.ItemCollection, 0, len(following))
	for _, entry := range following {
		if entry.ActorIRI == "" {
			continue
		}
		items = append(items, vocab.IRI(entry.ActorIRI))
	}

	followingURL := ap.LocalActorURL(ap.LocalActorURLConfig{
		Host:           ac.context.GetConfig().Host,
		WebRoot:        ac.context.GetConfig().WebRoot,
		FallbackHost:   c.Request().Host,
		FallbackScheme: "https",
	}, targetUser.Username) + "/following"

	totalItems := len(items)
	collection := vocab.OrderedCollectionNew(vocab.ID(followingURL))
	collection.TotalItems = uint(totalItems)
	collection.First = vocab.IRI(followingURL + "?page=1")
	if totalItems > 0 {
		totalPages := (totalItems + followersPageSize - 1) / followersPageSize
		collection.Last = vocab.IRI(fmt.Sprintf("%s?page=%d", followingURL, totalPages))
	}

	if page == 0 {
		resp, err := jsonld.WithContext(
			jsonld.IRI(vocab.ActivityBaseURI),
		).Marshal(collection)
		if err != nil {
			return renderApiError(c, http.StatusInternalServerError, err)
		}

		return renderActivityPubResponse(c, resp)
	}

	start := min((page-1)*followersPageSize, totalItems)
	end := min(start+followersPageSize, totalItems)

	pageItems := items[start:end]
	totalPages := (totalItems + followersPageSize - 1) / followersPageSize

	collectionPage := vocab.OrderedCollectionPageNew(collection)
	collectionPage.ID = vocab.ID(fmt.Sprintf("%s?page=%d", followingURL, page))
	collectionPage.OrderedItems = pageItems
	collectionPage.StartIndex = uint(start)

	if page > 1 {
		collectionPage.Prev = vocab.IRI(fmt.Sprintf("%s?page=%d", followingURL, page-1))
	}
	if page < totalPages {
		collectionPage.Next = vocab.IRI(fmt.Sprintf("%s?page=%d", followingURL, page+1))
	}

	resp, err := jsonld.WithContext(
		jsonld.IRI(vocab.ActivityBaseURI),
	).Marshal(collectionPage)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	return renderActivityPubResponse(c, resp)
}

// Followers returns the ActivityPub followers collection for a local user
// @Summary      Get ActivityPub followers collection
// @Tags         activity-pub
// @Param        username  path   string  true   "Username"
// @Param        page      query  int     false  "Page number (1-based)"
// @Produce      json
// @Success      200  {object}  map[string]any
// @Failure      400  {object}  dto.Response[any]
// @Failure      404  {object}  dto.Response[any]
// @Router       /ap/users/{username}/followers [get]
func (ac *apUserController) Followers(c echo.Context) error {
	targetUser, err := ac.targetActivityPubUser(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	page := 0
	if rawPage := strings.TrimSpace(c.QueryParam("page")); rawPage != "" {
		page, err = strconv.Atoi(rawPage)
		if err != nil || page < 1 {
			return renderApiError(c, http.StatusBadRequest, errors.New("invalid page"))
		}
	}

	followers, err := model.ListApprovedFollowers(ac.context.GetDB(), targetUser.ID)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	items := make(vocab.ItemCollection, 0, len(followers))
	for _, follower := range followers {
		if follower.ActorIRI == "" {
			continue
		}
		items = append(items, vocab.IRI(follower.ActorIRI))
	}

	followersURL := ap.LocalActorURL(ap.LocalActorURLConfig{
		Host:           ac.context.GetConfig().Host,
		WebRoot:        ac.context.GetConfig().WebRoot,
		FallbackHost:   c.Request().Host,
		FallbackScheme: "https",
	}, targetUser.Username) + "/followers"

	totalItems := len(items)
	collection := vocab.OrderedCollectionNew(vocab.ID(followersURL))
	collection.TotalItems = uint(totalItems)
	collection.First = vocab.IRI(followersURL + "?page=1")
	if totalItems > 0 {
		totalPages := (totalItems + followersPageSize - 1) / followersPageSize
		collection.Last = vocab.IRI(fmt.Sprintf("%s?page=%d", followersURL, totalPages))
	}

	if page == 0 {
		resp, err := jsonld.WithContext(
			jsonld.IRI(vocab.ActivityBaseURI),
		).Marshal(collection)
		if err != nil {
			return renderApiError(c, http.StatusInternalServerError, err)
		}

		return renderActivityPubResponse(c, resp)
	}

	start := min((page-1)*followersPageSize, totalItems)
	end := min(start+followersPageSize, totalItems)

	pageItems := items[start:end]
	totalPages := (totalItems + followersPageSize - 1) / followersPageSize

	collectionPage := vocab.OrderedCollectionPageNew(collection)
	collectionPage.ID = vocab.ID(fmt.Sprintf("%s?page=%d", followersURL, page))
	collectionPage.OrderedItems = pageItems
	collectionPage.StartIndex = uint(start)

	if page > 1 {
		collectionPage.Prev = vocab.IRI(fmt.Sprintf("%s?page=%d", followersURL, page-1))
	}
	if page < totalPages {
		collectionPage.Next = vocab.IRI(fmt.Sprintf("%s?page=%d", followersURL, page+1))
	}

	resp, err := jsonld.WithContext(
		jsonld.IRI(vocab.ActivityBaseURI),
	).Marshal(collectionPage)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	return renderActivityPubResponse(c, resp)
}
