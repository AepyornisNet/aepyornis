package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/AepyornisNet/aepyornis/pkg/aputil"
	"github.com/AepyornisNet/aepyornis/pkg/config"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/AepyornisNet/aepyornis/pkg/service"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
	"github.com/labstack/echo/v4"
	"github.com/samber/do/v2"
)

type ApUserController interface {
	GetUser(c echo.Context) error
	Following(c echo.Context) error
	Followers(c echo.Context) error
}

type apUserController struct {
	cfg          *config.Config
	actorService service.ActivityPubActorService
	followerRepo repository.Follower
	userRepo     repository.User
}

const followersPageSize = 20

func NewApUserController(injector do.Injector) ApUserController {
	return &apUserController{
		cfg:          do.MustInvoke[*config.Config](injector),
		actorService: do.MustInvoke[service.ActivityPubActorService](injector),
		followerRepo: do.MustInvoke[repository.Follower](injector),
		userRepo:     do.MustInvoke[repository.User](injector),
	}
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

	user, err := ac.userRepo.GetByUsername(username)
	if err != nil || !user.ActivityPubEnabled() {
		return renderApiError(c, http.StatusNotFound, errors.New("resource not found"))
	}

	actorURL, err := ac.actorService.ActorURL(&user.Profile)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, fmt.Errorf("failed to resolve actor URL: %w", err))
	}

	person := vocab.Person{
		Type:              vocab.PersonType,
		ID:                vocab.ID(actorURL),
		Name:              vocab.DefaultNaturalLanguage(user.Profile.DisplayName),
		PreferredUsername: vocab.DefaultNaturalLanguage(user.Profile.Username),
		Published:         user.CreatedAt.UTC(),
		Inbox:             vocab.IRI(actorURL + "/inbox"),
		Outbox:            vocab.IRI(actorURL + "/outbox"),
		Following:         vocab.IRI(actorURL + "/following"),
		Followers:         vocab.IRI(actorURL + "/followers"),
	}

	if user.Profile.PublicKey != "" {
		person.PublicKey = vocab.PublicKey{
			ID:           vocab.ID(actorURL + "#main-key"),
			Owner:        vocab.IRI(actorURL),
			PublicKeyPem: user.Profile.PublicKey,
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

	user, err := ac.userRepo.GetByUsername(username)
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

	following, err := ac.followerRepo.ListApprovedFollowing(targetUser.Profile.ID)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	items := make(vocab.ItemCollection, 0, len(following))
	for _, entry := range following {
		actorURL, actorErr := ac.actorService.ActorURL(entry.FollowingProfile)
		if actorErr != nil || actorURL == "" {
			continue
		}
		items = append(items, vocab.IRI(actorURL))
	}

	followingURL := aputil.LocalActorURL(aputil.LocalActorURLConfig{
		Host:           ac.cfg.Host,
		WebRoot:        ac.cfg.WebRoot,
		FallbackHost:   c.Request().Host,
		FallbackScheme: "https",
	}, targetUser.Profile.Username) + "/following"

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

	followers, err := ac.followerRepo.ListApprovedFollowers(targetUser.Profile.ID)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	items := make(vocab.ItemCollection, 0, len(followers))
	for _, follower := range followers {
		actorURL, actorErr := ac.actorService.ActorURL(follower.Profile)
		if actorErr != nil || actorURL == "" {
			continue
		}
		items = append(items, vocab.IRI(actorURL))
	}

	followersURL := aputil.LocalActorURL(aputil.LocalActorURLConfig{
		Host:           ac.cfg.Host,
		WebRoot:        ac.cfg.WebRoot,
		FallbackHost:   c.Request().Host,
		FallbackScheme: "https",
	}, targetUser.Profile.Username) + "/followers"

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
