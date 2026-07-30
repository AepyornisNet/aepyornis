package notification

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/SherClockHolmes/webpush-go"
)

type AngularPushNotification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
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
// {"notification": {"title": "...", "body": "..."}} and dispatches it via WebPush.
func (w *WebPush) Send(ctx context.Context, subject, message string) error {
	if len(w.receivers) == 0 {
		return nil
	}

	payload := AngularPushPayload{
		Notification: AngularPushNotification{
			Title: subject,
			Body:  message,
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
		if err != nil {
			return fmt.Errorf("send webpush notification: %w", err)
		}
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}

	return nil
}
