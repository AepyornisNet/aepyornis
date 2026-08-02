package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/SherClockHolmes/webpush-go"
)

type AngularPushNotification struct {
	Title string         `json:"title"`
	Body  string         `json:"body"`
	Icon  string         `json:"icon,omitempty"`
	Badge string         `json:"badge,omitempty"`
	Data  map[string]any `json:"data,omitempty"`
}

type AngularPushPayload struct {
	Notification AngularPushNotification `json:"notification"`
}

// WebPush struct holds VAPID keys and subscribers for sending WebPush notifications.
type WebPush struct {
	vapidPublicKey  string
	vapidPrivateKey string
	subscriber      string
	receivers       []webpush.Subscription
	OnExpired       func(endpoint string)
}

// NewWebPush creates a new WebPush notifier with the given VAPID key pair.
func NewWebPush(vapidPublicKey, vapidPrivateKey string) *WebPush {
	return &WebPush{
		vapidPublicKey:  vapidPublicKey,
		vapidPrivateKey: vapidPrivateKey,
		receivers:       make([]webpush.Subscription, 0),
	}
}

// SetSubscriber sets the subscriber contact info (e.g., mailto:email or URL) for VAPID headers.
func (w *WebPush) SetSubscriber(subscriber string) {
	w.subscriber = subscriber
}

// AddReceivers adds webpush.Subscription targets to receive notifications.
func (w *WebPush) AddReceivers(receivers ...webpush.Subscription) {
	w.receivers = append(w.receivers, receivers...)
}

// Send formats the subject and message into Angular's expected JSON push payload structure:
// {"notification": {"title": "...", "body": "...", "icon": "...", "data": {"url": "..."}}} and dispatches it via WebPush.
func (w *WebPush) Send(ctx context.Context, subject, message string) error {
	if len(w.receivers) == 0 {
		return nil
	}

	payload := AngularPushPayload{
		Notification: AngularPushNotification{
			Title: subject,
			Body:  message,
			Icon:  "/icons/icon-192x192.png",
			Badge: "/icons/icon-72x72.png",
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webpush payload: %w", err)
	}

	options := &webpush.Options{
		Subscriber:      w.subscriber,
		VAPIDPublicKey:  w.vapidPublicKey,
		VAPIDPrivateKey: w.vapidPrivateKey,
		TTL:             30,
	}

	for _, receiver := range w.receivers {
		resp, err := webpush.SendNotificationWithContext(ctx, payloadBytes, &receiver, options)
		if resp != nil {
			if (resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone) && w.OnExpired != nil {
				w.OnExpired(receiver.Endpoint)
			}
			if resp.Body != nil {
				_ = resp.Body.Close()
			}
		}
		if err != nil {
			// Log or handle error for individual subscription without failing other subscriptions
			continue
		}
	}

	return nil
}
