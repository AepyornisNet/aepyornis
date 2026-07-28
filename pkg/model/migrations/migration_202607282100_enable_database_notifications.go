package migrations

import (
	"errors"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"gorm.io/gorm"
)

func init() {
	model.RegisterMigration(
		202607282100,
		"enable database notifications by default for all users",
		nil,
		func(db *gorm.DB) error {
			var users []model.User
			if err := db.Find(&users).Error; err != nil {
				return err
			}

			for _, user := range users {
				var settings model.UserNotificationSettings
				err := db.Where("user_id = ? AND method = ?", user.ID, "database").First(&settings).Error
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						newSettings := model.UserNotificationSettings{
							UserID:        user.ID,
							Method:        "database",
							FollowRequest: true,
							WorkoutLike:   true,
							WorkoutReply:  true,
						}
						if err := db.Create(&newSettings).Error; err != nil {
							return err
						}
					} else {
						return err
					}
				} else {
					settings.FollowRequest = true
					settings.WorkoutLike = true
					settings.WorkoutReply = true
					if err := db.Save(&settings).Error; err != nil {
						return err
					}
				}
			}
			return nil
		},
		nil,
		nil,
	)
}
