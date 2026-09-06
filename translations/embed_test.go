package apptranslations

import (
	"context"
	"testing"

	"github.com/invopop/ctxi18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbedded(t *testing.T) {
	c, err := FS().Open("en.yaml")
	require.NoError(t, err)

	s, err := c.Stat()
	require.NoError(t, err)

	require.NotZero(t, s.Size())
}

func TestNotificationTranslations(t *testing.T) {
	err := ctxi18n.LoadWithDefault(FS(), "en")
	require.NoError(t, err)

	deCtx, err := ctxi18n.WithLocale(context.Background(), "de")
	require.NoError(t, err)
	deLocale := ctxi18n.Locale(deCtx)
	require.NotNil(t, deLocale)

	assert.Equal(t, "Neue Folgeanfrage", deLocale.T("notifications.follow_request_subject"))
	assert.Equal(t, "Alice möchte dir folgen", deLocale.T("notifications.follow_request_message", "Alice"))
	assert.Equal(t, "Neues Gefällt-mir", deLocale.T("notifications.workout_like_subject"))
	assert.Equal(t, "Alice gefällt dein Training", deLocale.T("notifications.workout_like_message", "Alice"))
	assert.Equal(t, "Neuer Kommentar zum Training", deLocale.T("notifications.workout_reply_subject"))
	assert.Equal(t, "Alice hat dein Training kommentiert", deLocale.T("notifications.workout_reply_message", "Alice"))

	enCtx, err := ctxi18n.WithLocale(context.Background(), "en")
	require.NoError(t, err)
	enLocale := ctxi18n.Locale(enCtx)
	require.NotNil(t, enLocale)

	assert.Equal(t, "New follow request", enLocale.T("notifications.follow_request_subject"))
	assert.Equal(t, "Alice wants to follow you", enLocale.T("notifications.follow_request_message", "Alice"))
}

func TestValidationTranslations(t *testing.T) {
	err := ctxi18n.LoadWithDefault(FS(), "en")
	require.NoError(t, err)

	deCtx, err := ctxi18n.WithLocale(context.Background(), "de")
	require.NoError(t, err)
	deLocale := ctxi18n.Locale(deCtx)
	require.NotNil(t, deLocale)

	assert.Equal(t, "email ist erforderlich", deLocale.T("validation.required", "email"))
	assert.Equal(t, "email muss eine gültige E-Mail-Adresse sein", deLocale.T("validation.email", "email"))
	assert.Equal(t, "passwort muss mindestens 6 Zeichen lang sein", deLocale.T("validation.min_string", "passwort", "6"))
	assert.Equal(t, "passwort darf nicht leer sein", deLocale.T("validation.min_string_empty", "passwort"))
	assert.Equal(t, "rollen muss mindestens 1 Eintrag enthalten", deLocale.T("validation.min_items_one", "rollen"))
	assert.Equal(t, "rollen muss mindestens 2 Einträge enthalten", deLocale.T("validation.min_items", "rollen", "2"))
	assert.Equal(t, "alter muss mindestens 18 sein", deLocale.T("validation.min_numeric", "alter", "18"))
	assert.Equal(t, "status muss einer der folgenden Werte sein: active, pending", deLocale.T("validation.oneof", "status", "active, pending"))
	assert.Equal(t, "website muss eine gültige URL sein", deLocale.T("validation.url", "website"))
	assert.Equal(t, "passwort_bestätigung muss mit passwort übereinstimmen", deLocale.T("validation.eqfield", "passwort_bestätigung", "passwort"))

	enCtx, err := ctxi18n.WithLocale(context.Background(), "en")
	require.NoError(t, err)
	enLocale := ctxi18n.Locale(enCtx)
	require.NotNil(t, enLocale)

	assert.Equal(t, "email is required", enLocale.T("validation.required", "email"))
	assert.Equal(t, "email must be a valid email address", enLocale.T("validation.email", "email"))
	assert.Equal(t, "password must be at least 6 characters long", enLocale.T("validation.min_string", "password", "6"))
	assert.Equal(t, "password cannot be empty", enLocale.T("validation.min_string_empty", "password"))
	assert.Equal(t, "roles must contain at least 1 item", enLocale.T("validation.min_items_one", "roles"))
	assert.Equal(t, "roles must contain at least 2 items", enLocale.T("validation.min_items", "roles", "2"))
	assert.Equal(t, "age must be at least 18", enLocale.T("validation.min_numeric", "age", "18"))
	assert.Equal(t, "status must be one of: active, pending", enLocale.T("validation.oneof", "status", "active, pending"))
	assert.Equal(t, "website must be a valid URL", enLocale.T("validation.url", "website"))
	assert.Equal(t, "password_confirmation must match password", enLocale.T("validation.eqfield", "password_confirmation", "password"))
}
