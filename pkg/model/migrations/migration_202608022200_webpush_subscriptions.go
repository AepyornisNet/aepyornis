package migrations

import (
	"encoding/json"
	"strings"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/SherClockHolmes/webpush-go"
	"gorm.io/gorm"
)

func init() {
	model.RegisterMigration(
		202608022200,
		"migrate single webpush subscription method_settings to user_webpush_subscriptions table",
		nil,
		func(db *gorm.DB) error {
			var settings []model.UserNotificationSettings
			if err := db.Where("method = ? AND method_settings IS NOT NULL", "webpush").Find(&settings).Error; err != nil {
				return err
			}

			for _, s := range settings {
				if s.MethodSettings == nil || len(*s.MethodSettings) == 0 {
					continue
				}

				var sub webpush.Subscription
				if err := json.Unmarshal(*s.MethodSettings, &sub); err != nil {
					continue
				}

				if strings.TrimSpace(sub.Endpoint) == "" || strings.TrimSpace(sub.Keys.Auth) == "" || strings.TrimSpace(sub.Keys.P256dh) == "" {
					continue
				}

				var count int64
				db.Model(&model.UserWebpushSubscription{}).
					Where("user_id = ? AND endpoint = ?", s.UserID, sub.Endpoint).
					Count(&count)

				if count == 0 {
					entry := model.UserWebpushSubscription{
						UserID:   s.UserID,
						Endpoint: sub.Endpoint,
						P256dh:   sub.Keys.P256dh,
						Auth:     sub.Keys.Auth,
					}
					_ = db.Create(&entry).Error
				}
			}
			return nil
		},
		nil,
		nil,
	)
}
