package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
	"github.com/google/uuid"
	ap "github.com/jovandeginste/workout-tracker/v2/pkg/activitypub"
	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type ApUserController interface {
	GetUser(c echo.Context) error
	Inbox(c echo.Context) error
	Outbox(c echo.Context) error
	OutboxItem(c echo.Context) error
	OutboxFit(c echo.Context) error
	Following(c echo.Context) error
	Followers(c echo.Context) error
}

type apUserController struct {
	context *container.Container
}

const followersPageSize = 20
const outboxPageSize = 20

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

func (ac *apUserController) Inbox(c echo.Context) error {
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
		return c.NoContent(http.StatusNotImplemented)
	default:
		return c.NoContent(http.StatusNotImplemented)
	}
}

func (ac *apUserController) Outbox(c echo.Context) error {
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

	actorURL := ap.LocalActorURL(ap.LocalActorURLConfig{
		Host:           ac.context.GetConfig().Host,
		WebRoot:        ac.context.GetConfig().WebRoot,
		FallbackHost:   c.Request().Host,
		FallbackScheme: c.Scheme(),
	}, targetUser.Username)
	outboxURL := actorURL + "/outbox"

	total, err := model.CountAPOutboxEntriesByUser(ac.context.GetDB(), targetUser.ID)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if page == 0 {
		resp := map[string]any{
			"@context":   "https://www.w3.org/ns/activitystreams",
			"id":         outboxURL,
			"type":       "OrderedCollection",
			"totalItems": total,
			"first":      outboxURL + "?page=1",
		}

		if total > 0 {
			totalPages := (int(total) + outboxPageSize - 1) / outboxPageSize
			resp["last"] = fmt.Sprintf("%s?page=%d", outboxURL, totalPages)
		}

		payload, err := json.Marshal(resp)
		if err != nil {
			return renderApiError(c, http.StatusInternalServerError, err)
		}

		return renderActivityPubResponse(c, http.StatusOK, payload)
	}

	offset := (page - 1) * outboxPageSize
	entries, err := model.GetAPOutboxEntriesByUser(ac.context.GetDB(), targetUser.ID, outboxPageSize, offset)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	orderedItems := make([]any, 0, len(entries))
	for _, entry := range entries {
		if len(entry.Activity) == 0 {
			continue
		}

		item := map[string]any{}
		if err := json.Unmarshal(entry.Activity, &item); err != nil {
			continue
		}

		orderedItems = append(orderedItems, item)
	}

	totalPages := 0
	if total > 0 {
		totalPages = (int(total) + outboxPageSize - 1) / outboxPageSize
	}

	resp := map[string]any{
		"@context":     "https://www.w3.org/ns/activitystreams",
		"id":           fmt.Sprintf("%s?page=%d", outboxURL, page),
		"type":         "OrderedCollectionPage",
		"partOf":       outboxURL,
		"startIndex":   offset,
		"orderedItems": orderedItems,
	}

	if page > 1 {
		resp["prev"] = fmt.Sprintf("%s?page=%d", outboxURL, page-1)
	}

	if page < totalPages {
		resp["next"] = fmt.Sprintf("%s?page=%d", outboxURL, page+1)
	}

	payload, err := json.Marshal(resp)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	return renderActivityPubResponse(c, http.StatusOK, payload)
}

func (ac *apUserController) OutboxItem(c echo.Context) error {
	targetUser, err := ac.targetActivityPubUser(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	outboxID, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	entry, err := model.GetAPOutboxEntryByUUIDAndUser(ac.context.GetDB(), targetUser.ID, outboxID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return renderApiError(c, http.StatusNotFound, err)
		}

		return renderApiError(c, http.StatusInternalServerError, err)
	}

	return renderActivityPubResponse(c, http.StatusOK, entry.Activity)
}

func (ac *apUserController) OutboxFit(c echo.Context) error {
	targetUser, err := ac.targetActivityPubUser(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	outboxID, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	entry, err := model.GetAPOutboxEntryByUUIDAndUser(ac.context.GetDB(), targetUser.ID, outboxID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return renderApiError(c, http.StatusNotFound, err)
		}

		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if entry.APOutboxWorkout == nil || len(entry.APOutboxWorkout.FitContent) == 0 {
		return renderApiError(c, http.StatusNotFound, errors.New("fit file not found"))
	}

	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", entry.APOutboxWorkout.FitFilename))
	return c.Blob(http.StatusOK, entry.APOutboxWorkout.FitContentType, entry.APOutboxWorkout.FitContent)
}

func (ac *apUserController) Following(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

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

		return renderActivityPubResponse(c, http.StatusOK, resp)
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

	return renderActivityPubResponse(c, http.StatusOK, resp)
}
