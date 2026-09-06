package validator

import (
	"context"
	"errors"
	"testing"

	"github.com/AepyornisNet/aepyornis/pkg/model/dto"
	apptranslations "github.com/AepyornisNet/aepyornis/translations"
	"github.com/invopop/ctxi18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestUserPayload struct {
	Email    string   `json:"email" validate:"required,email"`
	Password string   `json:"password" validate:"required,min=6"`
	Age      int      `json:"age" validate:"gte=18,lte=120"`
	Roles    []string `json:"roles" validate:"min=1,max=5"`
	Status   string   `query:"status" validate:"omitempty,oneof=active pending suspended"`
	Website  string   `form:"website" validate:"omitempty,url"`
}

func TestCustomValidator_Validate(t *testing.T) {
	v := New()

	t.Run("Valid payload", func(t *testing.T) {
		payload := TestUserPayload{
			Email:    "test@example.com",
			Password: "securepassword",
			Age:      25,
			Roles:    []string{"admin"},
			Status:   "active",
			Website:  "https://example.com",
		}
		err := v.Validate(&payload)
		assert.NoError(t, err)
	})

	t.Run("Required email and password errors", func(t *testing.T) {
		payload := TestUserPayload{
			Age:   25,
			Roles: []string{"admin"},
		}
		err := v.Validate(&payload)
		require.Error(t, err)

		var valErr *ValidationError
		require.True(t, errors.As(err, &valErr))

		msgs := valErr.ErrorMessages()
		assert.Contains(t, msgs, "email is required")
		assert.Contains(t, msgs, "password is required")
	})

	t.Run("Invalid email format error", func(t *testing.T) {
		payload := TestUserPayload{
			Email:    "invalid-email",
			Password: "securepassword",
			Age:      25,
			Roles:    []string{"admin"},
		}
		err := v.Validate(&payload)
		require.Error(t, err)

		var valErr *ValidationError
		require.True(t, errors.As(err, &valErr))
		assert.Contains(t, valErr.ErrorMessages(), "email must be a valid email address")
	})

	t.Run("Password too short error", func(t *testing.T) {
		payload := TestUserPayload{
			Email:    "test@example.com",
			Password: "123",
			Age:      25,
			Roles:    []string{"admin"},
		}
		err := v.Validate(&payload)
		require.Error(t, err)

		var valErr *ValidationError
		require.True(t, errors.As(err, &valErr))
		assert.Contains(t, valErr.ErrorMessages(), "password must be at least 6 characters long")
	})

	t.Run("Numeric range (gte, lte) errors", func(t *testing.T) {
		payloadTooYoung := TestUserPayload{
			Email:    "test@example.com",
			Password: "securepassword",
			Age:      15,
			Roles:    []string{"admin"},
		}
		err := v.Validate(&payloadTooYoung)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "age must be at least 18")

		payloadTooOld := TestUserPayload{
			Email:    "test@example.com",
			Password: "securepassword",
			Age:      150,
			Roles:    []string{"admin"},
		}
		err = v.Validate(&payloadTooOld)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "age must be at most 120")
	})

	t.Run("Slice min / max items error", func(t *testing.T) {
		payloadEmptyRoles := TestUserPayload{
			Email:    "test@example.com",
			Password: "securepassword",
			Age:      25,
			Roles:    []string{},
		}
		err := v.Validate(&payloadEmptyRoles)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "roles must contain at least 1 item")

		payloadTooManyRoles := TestUserPayload{
			Email:    "test@example.com",
			Password: "securepassword",
			Age:      25,
			Roles:    []string{"1", "2", "3", "4", "5", "6"},
		}
		err = v.Validate(&payloadTooManyRoles)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "roles must contain at most 5 items")
	})

	t.Run("Oneof error with readable options", func(t *testing.T) {
		payload := TestUserPayload{
			Email:    "test@example.com",
			Password: "securepassword",
			Age:      25,
			Roles:    []string{"admin"},
			Status:   "unknown",
		}
		err := v.Validate(&payload)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status must be one of: active, pending, suspended")
	})

	t.Run("URL error", func(t *testing.T) {
		payload := TestUserPayload{
			Email:    "test@example.com",
			Password: "securepassword",
			Age:      25,
			Roles:    []string{"admin"},
			Website:  "not-a-url",
		}
		err := v.Validate(&payload)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "website must be a valid URL")
	})

	t.Run("Unwrap support", func(t *testing.T) {
		payload := TestUserPayload{}
		err := v.Validate(&payload)
		require.Error(t, err)

		var valErr *ValidationError
		require.True(t, errors.As(err, &valErr))
		unwrapped := valErr.Unwrap()
		assert.NotEmpty(t, unwrapped)
	})
}

func TestValidationLocalization(t *testing.T) {
	err := ctxi18n.LoadWithDefault(apptranslations.FS(), "en")
	require.NoError(t, err)

	v := New()
	payload := TestUserPayload{
		Email:    "invalid-email",
		Password: "123",
		Age:      15,
		Roles:    []string{},
		Status:   "unknown",
		Website:  "not-a-url",
	}

	err = v.Validate(&payload)
	require.Error(t, err)

	var valErr *ValidationError
	require.True(t, errors.As(err, &valErr))

	t.Run("German localization via Localize", func(t *testing.T) {
		deCtx, err := ctxi18n.WithLocale(context.Background(), "de")
		require.NoError(t, err)

		msgs := valErr.Localize(deCtx)
		assert.Contains(t, msgs, "email muss eine gültige E-Mail-Adresse sein")
		assert.Contains(t, msgs, "password muss mindestens 6 Zeichen lang sein")
		assert.Contains(t, msgs, "age muss mindestens 18 sein")
		assert.Contains(t, msgs, "roles muss mindestens 1 Eintrag enthalten")
		assert.Contains(t, msgs, "status muss einer der folgenden Werte sein: active, pending, suspended")
		assert.Contains(t, msgs, "website muss eine gültige URL sein")
	})

	t.Run("English localization via Localize", func(t *testing.T) {
		enCtx, err := ctxi18n.WithLocale(context.Background(), "en")
		require.NoError(t, err)

		msgs := valErr.Localize(enCtx)
		assert.Contains(t, msgs, "email must be a valid email address")
		assert.Contains(t, msgs, "password must be at least 6 characters long")
		assert.Contains(t, msgs, "age must be at least 18")
		assert.Contains(t, msgs, "roles must contain at least 1 item")
		assert.Contains(t, msgs, "status must be one of: active, pending, suspended")
		assert.Contains(t, msgs, "website must be a valid URL")
	})

	t.Run("Nil context fallback to default English messages", func(t *testing.T) {
		msgs := valErr.Localize(nil)
		assert.Contains(t, msgs, "email must be a valid email address")
	})

	t.Run("DTO Response AddContextError with German locale", func(t *testing.T) {
		deCtx, err := ctxi18n.WithLocale(context.Background(), "de")
		require.NoError(t, err)

		resp := dto.Response[any]{}
		resp.AddContextError(deCtx, valErr)

		assert.Contains(t, resp.Errors, "email muss eine gültige E-Mail-Adresse sein")
		assert.Contains(t, resp.Errors, "password muss mindestens 6 Zeichen lang sein")
	})

	t.Run("DTO PaginatedResponse AddContextError with German locale", func(t *testing.T) {
		deCtx, err := ctxi18n.WithLocale(context.Background(), "de")
		require.NoError(t, err)

		pResp := dto.PaginatedResponse[any]{}
		pResp.AddContextError(deCtx, valErr)

		assert.Contains(t, pResp.Errors, "email muss eine gültige E-Mail-Adresse sein")
		assert.Contains(t, pResp.Errors, "password muss mindestens 6 Zeichen lang sein")
	})

	t.Run("Unwrapped fieldError Localize", func(t *testing.T) {
		deCtx, err := ctxi18n.WithLocale(context.Background(), "de")
		require.NoError(t, err)

		unwrapped := valErr.Unwrap()
		require.NotEmpty(t, unwrapped)

		var localized []string
		for _, e := range unwrapped {
			if le, ok := e.(interface{ Localize(context.Context) string }); ok {
				localized = append(localized, le.Localize(deCtx))
			}
		}
		assert.Contains(t, localized, "email muss eine gültige E-Mail-Adresse sein")
	})
}
