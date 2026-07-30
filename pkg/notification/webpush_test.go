package notification

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAngularPushPayloadJSON(t *testing.T) {
	payload := AngularPushPayload{
		Notification: AngularPushNotification{
			Title: "Test Title",
			Body:  "Test Message Body",
		},
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	expectedJSON := `{"notification":{"title":"Test Title","body":"Test Message Body"}}`
	assert.JSONEq(t, expectedJSON, string(data))
}

func TestWebPushNotifierEmptyReceivers(t *testing.T) {
	wp := NewWebPush("pubkey", "privkey")
	err := wp.Send(context.Background(), "Subject", "Message")
	assert.NoError(t, err)
}

func TestWebPushAddReceivers(t *testing.T) {
	wp := NewWebPush("pubkey", "privkey")
	wp.SetSubscriber("mailto:test@example.com")
	sub := webpush.Subscription{
		Endpoint: "https://example.com/push",
		Keys: webpush.Keys{
			Auth:   "authkey",
			P256dh: "p256dhkey",
		},
	}
	wp.AddReceivers(sub)
	assert.Len(t, wp.receivers, 1)
	assert.Equal(t, "https://example.com/push", wp.receivers[0].Endpoint)
}
