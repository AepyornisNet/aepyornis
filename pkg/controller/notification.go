package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/AepyornisNet/aepyornis/pkg/config"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/model/dto"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/labstack/echo/v5"
	"github.com/nikoksr/notify/service/webpush"
	"github.com/samber/do/v2"
	"gorm.io/gorm"
)

type NotificationController interface {
	GetNotifications(c *echo.Context) error
	MarkAsRead(c *echo.Context) error
	GetConfig(c *echo.Context) error
	UpdateConfig(c *echo.Context) error
	GetWebpushSubscriptions(c *echo.Context) error
	SubscribeWebpush(c *echo.Context) error
	UnsubscribeWebpush(c *echo.Context) error
}

type notificationController struct {
	notificationRepo repository.Notification

	cfg *config.Config
	db  *gorm.DB
}

func NewNotificationController(injector do.Injector) NotificationController {
	return &notificationController{
		notificationRepo: do.MustInvoke[repository.Notification](injector),
		cfg:              do.MustInvoke[*config.Config](injector),
		db:               do.MustInvoke[*gorm.DB](injector),
	}
}

// GetNotifications returns all unread notifications of the user
// @Summary      Get user notifications
// @Tags         notification
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  dto.Response[[]model.Notification]
// @Failure      400  {object}  dto.Response[string]
// @Failure      500  {object}  dto.Response[string]
// @Router       /notifications [get]
func (nc *notificationController) GetNotifications(c *echo.Context) error {
	user := currentUser(c)

	unread, err := nc.notificationRepo.GetUnread(c.Request().Context(), user)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, errors.New("could not read notifications"))
	}

	resp := dto.Response[[]model.Notification]{
		Results: unread,
	}

	return c.JSON(http.StatusOK, resp)
}

// GetConfig returns current user's notification config
// @Summary      Get notification config
// @Tags         notification
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  dto.Response[[]model.UserNotificationSettings]
// @Failure      500  {object}  dto.Response[string]
// @Router       /notifications/settings [get]
func (nc *notificationController) GetConfig(c *echo.Context) error {
	user := currentUser(c)

	settings, err := nc.notificationRepo.GetAllUserSettings(c.Request().Context(), user)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	settingsByMethod := make(map[string]model.UserNotificationSettings, len(settings))
	for _, item := range settings {
		settingsByMethod[item.Method] = item
	}

	results := make([]model.UserNotificationSettings, 0, len(nc.cfg.AvailableNotificationProviders()))
	for _, provider := range nc.cfg.AvailableNotificationProviders() {
		if item, ok := settingsByMethod[provider]; ok {
			results = append(results, item)
			continue
		}

		results = append(results, model.UserNotificationSettings{
			UserID:        user.ID,
			Method:        provider,
			FollowRequest: false,
			WorkoutLike:   false,
			WorkoutReply:  false,
		})
	}

	return c.JSON(http.StatusOK, dto.Response[[]model.UserNotificationSettings]{
		Results: results,
	})
}

// UpdateConfig updates current user's notification config
// @Summary      Update notification config
// @Tags         notification
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Success      200  {object}  dto.Response[model.UserNotificationSettings]
// @Failure      400  {object}  dto.Response[string]
// @Failure      500  {object}  dto.Response[string]
// @Router       /notifications/{type} [post]
func (nc *notificationController) UpdateConfig(c *echo.Context) error {
	user := currentUser(c)

	nType := c.Param("type")
	if !slices.Contains(nc.cfg.AvailableNotificationProviders(), nType) {
		return renderApiError(c, http.StatusBadRequest, errors.New("invalid notification type"))
	}

	var updateData dto.UserNotificationSettingsData
	if err := c.Bind(&updateData); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	currentSettings, err := nc.notificationRepo.GetUserSettings(c.Request().Context(), nType, user)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if currentSettings == nil {
		currentSettings = &model.UserNotificationSettings{
			UserID: user.ID,
			Method: nType,
		}

		if err := gorm.G[model.UserNotificationSettings](nc.db).Create(c.Request().Context(), currentSettings); err != nil {
			return renderApiError(c, http.StatusInternalServerError, err)
		}
	}

	if strings.TrimSpace(updateData.MethodSettings) != "" {
		rawSettings := []byte(updateData.MethodSettings)
		if !json.Valid(rawSettings) {
			return renderApiError(c, http.StatusBadRequest, errors.New("method_settings must be valid JSON"))
		}

		if nType == "webpush" {
			var sub webpush.Subscription
			if err := json.Unmarshal(rawSettings, &sub); err != nil {
				return renderApiError(c, http.StatusBadRequest, errors.New("invalid webpush subscription JSON structure"))
			}
			if strings.TrimSpace(sub.Endpoint) == "" || strings.TrimSpace(sub.Keys.Auth) == "" || strings.TrimSpace(sub.Keys.P256dh) == "" {
				return renderApiError(c, http.StatusBadRequest, errors.New("webpush subscription must contain endpoint, auth, and p256dh keys"))
			}
			var existingSub model.UserWebpushSubscription
			errSub := nc.db.WithContext(c.Request().Context()).Where("user_id = ? AND endpoint = ?", user.ID, sub.Endpoint).First(&existingSub).Error
			if errors.Is(errSub, gorm.ErrRecordNotFound) {
				_ = nc.db.WithContext(c.Request().Context()).Create(&model.UserWebpushSubscription{
					UserID:    user.ID,
					Endpoint:  sub.Endpoint,
					P256dh:    sub.Keys.P256dh,
					Auth:      sub.Keys.Auth,
					UserAgent: c.Request().UserAgent(),
				}).Error
			}
		}

		settings := json.RawMessage(updateData.MethodSettings)
		currentSettings.MethodSettings = &settings
	}
	currentSettings.WorkoutReply = updateData.WorkoutReply
	currentSettings.WorkoutLike = updateData.WorkoutLike
	currentSettings.FollowRequest = updateData.FollowRequest

	if err := nc.db.WithContext(c.Request().Context()).Save(currentSettings).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, currentSettings)
}

type MarkAsReadPayload struct {
	IDs []uint64 `json:"ids"`
}

// MarkAsRead marks specified or all notifications as read for the user
// @Summary      Mark notifications as read
// @Tags         notification
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Success      200  {object}  dto.Response[map[string]bool]
// @Failure      500  {object}  dto.Response[string]
// @Router       /notifications/read [post]
func (nc *notificationController) MarkAsRead(c *echo.Context) error {
	user := currentUser(c)
	var payload MarkAsReadPayload
	_ = c.Bind(&payload)

	if err := nc.notificationRepo.MarkAsRead(c.Request().Context(), user, payload.IDs); err != nil {
		return renderApiError(c, http.StatusInternalServerError, errors.New("could not mark notifications as read"))
	}

	return c.JSON(http.StatusOK, dto.Response[map[string]bool]{
		Results: map[string]bool{"success": true},
	})
}

type SubscribeWebpushPayload struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		Auth   string `json:"auth"`
		P256dh string `json:"p256dh"`
	} `json:"keys"`
	UserAgent string `json:"user_agent"`
}

type UnsubscribeWebpushPayload struct {
	Endpoint string `json:"endpoint"`
}

// GetWebpushSubscriptions returns current user's registered WebPush subscriptions
// @Summary      Get registered WebPush subscriptions
// @Tags         notification
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  dto.Response[[]model.UserWebpushSubscription]
// @Failure      500  {object}  dto.Response[string]
// @Router       /notifications/webpush/subscriptions [get]
func (nc *notificationController) GetWebpushSubscriptions(c *echo.Context) error {
	user := currentUser(c)
	var subs []model.UserWebpushSubscription
	if err := nc.db.WithContext(c.Request().Context()).Where("user_id = ?", user.ID).Order("created_at desc").Find(&subs).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, dto.Response[[]model.UserWebpushSubscription]{
		Results: subs,
	})
}

// SubscribeWebpush registers or updates a WebPush subscription for current user
// @Summary      Subscribe WebPush endpoint
// @Tags         notification
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Success      200  {object}  dto.Response[model.UserWebpushSubscription]
// @Failure      400  {object}  dto.Response[string]
// @Failure      500  {object}  dto.Response[string]
// @Router       /notifications/webpush/subscribe [post]
func (nc *notificationController) SubscribeWebpush(c *echo.Context) error {
	user := currentUser(c)

	var payload SubscribeWebpushPayload
	if err := c.Bind(&payload); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	endpoint := strings.TrimSpace(payload.Endpoint)
	auth := strings.TrimSpace(payload.Keys.Auth)
	p256dh := strings.TrimSpace(payload.Keys.P256dh)

	if endpoint == "" || auth == "" || p256dh == "" {
		return renderApiError(c, http.StatusBadRequest, errors.New("webpush subscription must contain endpoint, auth, and p256dh keys"))
	}

	userAgent := strings.TrimSpace(payload.UserAgent)
	if userAgent == "" {
		userAgent = c.Request().UserAgent()
	}

	var existing model.UserWebpushSubscription
	err := nc.db.WithContext(c.Request().Context()).Where("user_id = ? AND endpoint = ?", user.ID, endpoint).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			existing = model.UserWebpushSubscription{
				UserID:    user.ID,
				Endpoint:  endpoint,
				P256dh:    p256dh,
				Auth:      auth,
				UserAgent: userAgent,
			}
			if err := nc.db.WithContext(c.Request().Context()).Create(&existing).Error; err != nil {
				return renderApiError(c, http.StatusInternalServerError, err)
			}
		} else {
			return renderApiError(c, http.StatusInternalServerError, err)
		}
	} else {
		existing.P256dh = p256dh
		existing.Auth = auth
		existing.UserAgent = userAgent
		if err := nc.db.WithContext(c.Request().Context()).Save(&existing).Error; err != nil {
			return renderApiError(c, http.StatusInternalServerError, err)
		}
	}

	return c.JSON(http.StatusOK, dto.Response[model.UserWebpushSubscription]{
		Results: existing,
	})
}

// UnsubscribeWebpush deletes a WebPush subscription for current user
// @Summary      Unsubscribe WebPush endpoint
// @Tags         notification
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Success      200  {object}  dto.Response[map[string]bool]
// @Failure      500  {object}  dto.Response[string]
// @Router       /notifications/webpush/unsubscribe [post]
func (nc *notificationController) UnsubscribeWebpush(c *echo.Context) error {
	user := currentUser(c)

	var payload UnsubscribeWebpushPayload
	_ = c.Bind(&payload)

	endpoint := strings.TrimSpace(payload.Endpoint)
	if endpoint == "" {
		endpoint = strings.TrimSpace(c.QueryParam("endpoint"))
	}

	query := nc.db.WithContext(c.Request().Context()).Where("user_id = ?", user.ID)
	if endpoint != "" {
		query = query.Where("endpoint = ?", endpoint)
	}

	if err := query.Delete(&model.UserWebpushSubscription{}).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, dto.Response[map[string]bool]{
		Results: map[string]bool{"success": true},
	})
}
