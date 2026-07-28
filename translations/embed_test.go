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
