package repository

import (
	"context"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/samber/do/v2"
	"gorm.io/gorm"
)

type Notification interface {
	GetUnread(ctx context.Context, user *model.User) ([]model.Notification, error)
	MarkAsRead(ctx context.Context, user *model.User, ids []uint64) error
	MarkAllAsRead(ctx context.Context, user *model.User) error
	GetAllUserSettings(ctx context.Context, user *model.User) ([]model.UserNotificationSettings, error)
	GetUserSettings(ctx context.Context, nType string, user *model.User) (*model.UserNotificationSettings, error)
}

type notificationRepository struct {
	db *gorm.DB
}

func NewNotification(injector do.Injector) (Notification, error) {
	return &notificationRepository{db: do.MustInvoke[*gorm.DB](injector)}, nil
}

func (r *notificationRepository) GetUnread(ctx context.Context, user *model.User) ([]model.Notification, error) {
	notifications, err := gorm.G[model.Notification](r.db).
		Where("user_id = ? AND read_at IS NULL", user.ID).
		Order("created_at DESC").
		Find(ctx)
	if err != nil {
		return nil, err
	}

	return notifications, nil
}

func (r *notificationRepository) MarkAsRead(ctx context.Context, user *model.User, ids []uint64) error {
	now := time.Now()
	query := r.db.WithContext(ctx).Model(&model.Notification{}).Where("user_id = ? AND read_at IS NULL", user.ID)
	if len(ids) > 0 {
		query = query.Where("id IN ?", ids)
	}
	return query.Update("read_at", now).Error
}

func (r *notificationRepository) MarkAllAsRead(ctx context.Context, user *model.User) error {
	return r.MarkAsRead(ctx, user, nil)
}

func (r *notificationRepository) GetAllUserSettings(ctx context.Context, user *model.User) ([]model.UserNotificationSettings, error) {
	settings, err := gorm.G[model.UserNotificationSettings](r.db).Where("user_id = ?", user.ID).Find(ctx)
	if err != nil {
		return nil, err
	}

	return settings, nil
}

func (r *notificationRepository) GetUserSettings(ctx context.Context, nType string, user *model.User) (*model.UserNotificationSettings, error) {
	settings, err := gorm.G[model.UserNotificationSettings](r.db).Where("method = ? AND user_id = ?", nType, user.ID).Find(ctx)
	if err != nil {
		return nil, err
	}

	if len(settings) == 0 {
		return nil, nil
	}

	return &settings[0], nil
}
