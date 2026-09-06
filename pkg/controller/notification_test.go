package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AepyornisNet/aepyornis/pkg/config"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/model/dto"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/AepyornisNet/aepyornis/pkg/validator"
	"github.com/glebarez/sqlite"
	"github.com/labstack/echo/v5"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTestNotificationController(t *testing.T) (NotificationController, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&model.User{}, &model.UserNotificationSettings{}, &model.Notification{})
	require.NoError(t, err)

	cfg := &config.Config{
		Config: model.Config{
			EnvConfig: model.EnvConfig{
				VapidPublicKey:  "test-public-key",
				VapidPrivateKey: "test-private-key",
			},
		},
	}

	injector := do.New()
	do.ProvideValue(injector, db)
	do.ProvideValue(injector, cfg)
	do.Provide(injector, repository.NewNotification)

	return NewNotificationController(injector), db
}

func TestUpdateConfig_Validation(t *testing.T) {
	nc, db := setupTestNotificationController(t)

	user := &model.User{
		UserSecrets: model.UserSecrets{
			Email: "test@example.com",
		},
	}
	require.NoError(t, user.SetPassword("validpassword123"))
	require.NoError(t, db.Create(user).Error)

	e := echo.New()
	e.Validator = validator.New()

	t.Run("Invalid Notification Type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/notifications/invalid_provider", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "type", Value: "invalid_provider"}})
		c.Set("user", user)

		err := nc.UpdateConfig(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Garbage JSON in method_settings", func(t *testing.T) {
		body := dto.UserNotificationSettingsData{
			MethodSettings: "not valid json {",
			FollowRequest:  true,
		}
		data, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/notifications/webpush", bytes.NewReader(data))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "type", Value: "webpush"}})
		c.Set("user", user)

		err := nc.UpdateConfig(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Incomplete WebPush Subscription", func(t *testing.T) {
		body := dto.UserNotificationSettingsData{
			MethodSettings: `{"endpoint": "https://example.com"}`, // missing keys.auth and keys.p256dh
			FollowRequest:  true,
		}
		data, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/notifications/webpush", bytes.NewReader(data))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "type", Value: "webpush"}})
		c.Set("user", user)

		err := nc.UpdateConfig(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Valid WebPush Subscription", func(t *testing.T) {
		validSub := `{"endpoint":"https://push.example.com/sub/123","keys":{"auth":"authSecret123==","p256dh":"p256dhKey123=="}}`
		body := dto.UserNotificationSettingsData{
			MethodSettings: validSub,
			FollowRequest:  true,
			WorkoutLike:    true,
			WorkoutReply:   true,
		}
		data, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/notifications/webpush", bytes.NewReader(data))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "type", Value: "webpush"}})
		c.Set("user", user)

		err := nc.UpdateConfig(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
