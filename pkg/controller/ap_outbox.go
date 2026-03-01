package controller

import (
	"encoding/json"
	"errors"
	"fmt"
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

type ApOutboxController interface {
	Outbox(c echo.Context) error
	OutboxItem(c echo.Context) error
	OutboxFit(c echo.Context) error
	OutboxRouteImage(c echo.Context) error
	OutboxReplies(c echo.Context) error
}

type apOutboxController struct {
	context *container.Container
}

const outboxPageSize = 20

func NewApOutboxController(c *container.Container) ApOutboxController {
	return &apOutboxController{context: c}
}

func (ac *apOutboxController) targetActivityPubUser(c echo.Context) (*model.User, error) {
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

// Outbox returns the ActivityPub outbox collection for a local user
// @Summary      Get ActivityPub outbox collection
// @Tags         activity-pub
// @Param        username  path   string  true   "Username"
// @Param        page      query  int     false  "Page number (1-based)"
// @Produce      json
// @Success      200  {object}  map[string]any
// @Failure      400  {object}  dto.Response[any]
// @Failure      404  {object}  dto.Response[any]
// @Router       /ap/users/{username}/outbox [get]
func (ac *apOutboxController) Outbox(c echo.Context) error { //nolint:gocyclo
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

	total, err := ac.context.APOutboxRepo().CountEntriesByUser(targetUser.ID)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	collection := vocab.OrderedCollectionNew(vocab.ID(outboxURL))
	collection.TotalItems = uint(total)
	collection.First = vocab.IRI(outboxURL + "?page=1")
	if total > 0 {
		totalPages := (int(total) + outboxPageSize - 1) / outboxPageSize
		collection.Last = vocab.IRI(fmt.Sprintf("%s?page=%d", outboxURL, totalPages))
	}

	if page == 0 {
		payload, err := jsonld.WithContext(
			jsonld.IRI(vocab.ActivityBaseURI),
		).Marshal(collection)
		if err != nil {
			return renderApiError(c, http.StatusInternalServerError, err)
		}

		return renderActivityPubResponse(c, payload)
	}

	offset := (page - 1) * outboxPageSize
	entries, err := ac.context.APOutboxRepo().GetEntriesByUser(targetUser.ID, outboxPageSize, offset)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	orderedItems := make(vocab.ItemCollection, 0, len(entries))
	for _, entry := range entries {
		if len(entry.Activity) == 0 {
			continue
		}

		activity := vocab.Activity{}
		if err := jsonld.Unmarshal(entry.Activity, &activity); err != nil {
			continue
		}

		orderedItems = append(orderedItems, activity)
	}

	totalPages := 0
	if total > 0 {
		totalPages = (int(total) + outboxPageSize - 1) / outboxPageSize
	}

	collectionPage := vocab.OrderedCollectionPageNew(collection)
	collectionPage.ID = vocab.ID(fmt.Sprintf("%s?page=%d", outboxURL, page))
	collectionPage.OrderedItems = orderedItems
	collectionPage.StartIndex = uint(offset)

	if page > 1 {
		collectionPage.Prev = vocab.IRI(fmt.Sprintf("%s?page=%d", outboxURL, page-1))
	}

	if page < totalPages {
		collectionPage.Next = vocab.IRI(fmt.Sprintf("%s?page=%d", outboxURL, page+1))
	}

	payload, err := jsonld.WithContext(
		jsonld.IRI(vocab.ActivityBaseURI),
	).Marshal(collectionPage)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	return renderActivityPubResponse(c, payload)
}

// OutboxItem returns a single ActivityPub outbox activity by UUID
// @Summary      Get ActivityPub outbox item
// @Tags         activity-pub
// @Param        username  path  string  true  "Username"
// @Param        id        path  string  true  "Outbox entry UUID"
// @Produce      json
// @Success      200  {object}  map[string]any
// @Failure      400  {object}  dto.Response[any]
// @Failure      404  {object}  dto.Response[any]
// @Router       /ap/users/{username}/outbox/{id} [get]
func (ac *apOutboxController) OutboxItem(c echo.Context) error {
	targetUser, err := ac.targetActivityPubUser(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	outboxID, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	entry, err := ac.context.APOutboxRepo().GetEntryByUUIDAndUser(targetUser.ID, outboxID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return renderApiError(c, http.StatusNotFound, err)
		}

		return renderApiError(c, http.StatusInternalServerError, err)
	}

	activityPayload := entry.Activity

	// For workout entries, add replies collection info to the note
	if entry.APOutboxWorkout != nil {
		replyCount, countErr := ac.context.WorkoutReplyRepo().CountByWorkoutID(entry.APOutboxWorkout.WorkoutID)
		if countErr == nil && replyCount > 0 {
			// Deserialize the activity to add replies collection
			var activity map[string]interface{}
			if err := json.Unmarshal(activityPayload, &activity); err == nil {
				if object, ok := activity["object"].(map[string]interface{}); ok {
					repliesID := entry.ObjectID + "/replies"
					object["replies"] = map[string]interface{}{
						"id":         repliesID,
						"type":       "OrderedCollection",
						"totalItems": replyCount,
						"first":      repliesID,
					}
					activity["object"] = object
				}
				if updatedPayload, err := json.Marshal(activity); err == nil {
					activityPayload = updatedPayload
				}
			}
		}
	}

	return renderActivityPubResponse(c, activityPayload)
}

// OutboxFit downloads the FIT attachment for an ActivityPub outbox entry
// @Summary      Download ActivityPub outbox FIT file
// @Tags         activity-pub
// @Param        username  path  string  true  "Username"
// @Param        id        path  string  true  "Outbox entry UUID"
// @Produce      octet-stream
// @Success      200  {string}  string  "binary FIT content"
// @Failure      400  {object}  dto.Response[any]
// @Failure      404  {object}  dto.Response[any]
// @Router       /ap/users/{username}/outbox/{id}/fit [get]
func (ac *apOutboxController) OutboxFit(c echo.Context) error {
	targetUser, err := ac.targetActivityPubUser(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	outboxID, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	entry, err := ac.context.APOutboxRepo().GetEntryByUUIDAndUser(targetUser.ID, outboxID)
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

// OutboxRouteImage returns the route image attachment for an outbox entry
// @Summary      Get ActivityPub outbox route image
// @Tags         activity-pub
// @Param        username  path  string  true  "Username"
// @Param        id        path  string  true  "Outbox entry UUID"
// @Produce      octet-stream
// @Success      200  {string}  string  "binary image content"
// @Failure      400  {object}  dto.Response[any]
// @Failure      404  {object}  dto.Response[any]
// @Router       /ap/users/{username}/outbox/{id}/route-image [get]
func (ac *apOutboxController) OutboxRouteImage(c echo.Context) error {
	targetUser, err := ac.targetActivityPubUser(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	outboxID, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	entry, err := ac.context.APOutboxRepo().GetEntryByUUIDAndUser(targetUser.ID, outboxID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return renderApiError(c, http.StatusNotFound, err)
		}

		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if entry.APOutboxWorkout == nil {
		return renderApiError(c, http.StatusNotFound, errors.New("route image not found"))
	}

	attachment, attachmentErr := model.GetRouteImageAttachment(ac.context.GetDB(), entry.APOutboxWorkout.WorkoutID)
	if attachmentErr != nil {
		if errors.Is(attachmentErr, gorm.ErrRecordNotFound) {
			return renderApiError(c, http.StatusNotFound, errors.New("route image not found"))
		}

		return renderApiError(c, http.StatusInternalServerError, attachmentErr)
	}

	filename := attachment.Filename
	if filename == "" {
		filename = "workout-route.png"
	}

	contentType := attachment.ContentType
	if contentType == "" {
		contentType = model.RouteImageMIMEType
	}

	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", filename))
	return c.Blob(http.StatusOK, contentType, attachment.Content)
}

// OutboxReplies returns the ActivityPub replies collection for a workout note
// @Summary      Get ActivityPub workout replies collection
// @Tags         activity-pub
// @Param        username  path   string  true   "Username"
// @Param        id        path   string  true   "Outbox entry UUID"
// @Param        page      query  int     false  "Page number (1-based)"
// @Produce      json
// @Success      200  {object}  map[string]any
// @Failure      400  {object}  dto.Response[any]
// @Failure      404  {object}  dto.Response[any]
// @Router       /ap/users/{username}/outbox/{id}/replies [get]
func (ac *apOutboxController) OutboxReplies(c echo.Context) error {
	targetUser, err := ac.targetActivityPubUser(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	outboxID, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	entry, err := ac.context.APOutboxRepo().GetEntryByUUIDAndUser(targetUser.ID, outboxID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return renderApiError(c, http.StatusNotFound, err)
		}

		return renderApiError(c, http.StatusInternalServerError, err)
	}

	// Only workout entries have replies
	if entry.APOutboxWorkout == nil {
		return renderApiError(c, http.StatusNotFound, errors.New("this outbox entry does not support replies"))
	}

	page := 0
	if rawPage := strings.TrimSpace(c.QueryParam("page")); rawPage != "" {
		page, err = strconv.Atoi(rawPage)
		if err != nil || page < 1 {
			return renderApiError(c, http.StatusBadRequest, errors.New("invalid page"))
		}
	}

	// Get replies for this workout
	replies, err := ac.context.WorkoutReplyRepo().ListByWorkoutID(entry.APOutboxWorkout.WorkoutID, 10000, 0)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	// Build the replies collection
	repliesID := entry.ObjectID + "/replies"
	repliesCollection := vocab.OrderedCollectionNew(vocab.ID(repliesID))
	repliesCollection.TotalItems = uint(len(replies))

	// If not paginating, include items directly
	if page == 0 {
		items := vocab.ItemCollection{}
		for _, r := range replies {
			items = append(items, vocab.IRI(r.ObjectIRI))
		}

		repliesCollection.OrderedItems = items

		payload, err := jsonld.WithContext(
			jsonld.IRI(vocab.ActivityBaseURI),
		).Marshal(repliesCollection)
		if err != nil {
			return renderApiError(c, http.StatusInternalServerError, err)
		}

		return renderActivityPubResponse(c, payload)
	}

	// Handle pagination
	totalPages := (len(replies) + outboxPageSize - 1) / outboxPageSize
	if page > totalPages {
		return renderApiError(c, http.StatusBadRequest, errors.New("page out of range"))
	}

	offset := (page - 1) * outboxPageSize
	endOffset := offset + outboxPageSize
	if endOffset > len(replies) {
		endOffset = len(replies)
	}

	pageReplies := replies[offset:endOffset]
	items := vocab.ItemCollection{}
	for _, r := range pageReplies {
		items = append(items, vocab.IRI(r.ObjectIRI))
	}

	repliesPage := vocab.OrderedCollectionPageNew(repliesCollection)
	repliesPage.OrderedItems = items
	repliesPage.StartIndex = uint(offset)
	repliesPage.ID = vocab.ID(fmt.Sprintf("%s?page=%d", repliesID, page))

	if page > 1 {
		repliesPage.Prev = vocab.IRI(fmt.Sprintf("%s?page=%d", repliesID, page-1))
	}

	if page < totalPages {
		repliesPage.Next = vocab.IRI(fmt.Sprintf("%s?page=%d", repliesID, page+1))
	}

	payload, err := jsonld.WithContext(
		jsonld.IRI(vocab.ActivityBaseURI),
	).Marshal(repliesPage)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	return renderActivityPubResponse(c, payload)
}
