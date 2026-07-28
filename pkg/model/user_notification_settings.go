package model

import (
	"encoding/json"
	"reflect"
	"strings"
)

type UserNotificationSettings struct {
	Model

	UserID uint64 `gorm:"not null" json:"-"`          // The ID of the user the notification is sent to
	User   *User  `gorm:"foreignKey:UserID" json:"-"` // The user this notification is sent to

	Method         string           `json:"method"`
	MethodSettings *json.RawMessage `json:"method_settings,omitempty"`

	FollowRequest bool `gorm:"default:true" json:"follow_request"`
	WorkoutLike   bool `gorm:"default:true" json:"workout_like"`
	WorkoutReply  bool `gorm:"default:true" json:"workout_reply"`
}

// IsEnabled dynamically checks if the specified notification type is enabled for this channel setting
func (s *UserNotificationSettings) IsEnabled(notificationType string) bool {
	if s == nil {
		return true
	}

	normalized := strings.ReplaceAll(strings.ToLower(notificationType), "-", "_")
	val := reflect.Indirect(reflect.ValueOf(s))
	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		jsonTag := field.Tag.Get("json")
		jsonName := strings.Split(jsonTag, ",")[0]
		if jsonName == normalized && field.Type.Kind() == reflect.Bool {
			return val.Field(i).Bool()
		}
	}

	return true
}
