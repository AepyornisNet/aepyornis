package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/AepyornisNet/aepyornis/pkg/config"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/notification"
	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/mail"
	"github.com/nikoksr/notify/service/webpush"
	"github.com/samber/do/v2"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type BaseNotification interface {
	GetType() string
	GetSubject() string
	GetMessage() string
	GetMeta() *datatypes.JSON

	AllowDB() bool
	AllowEmail() bool
	AllowWebpush() bool
}

type NotificationService interface {
	SendRaw(ctx context.Context, user *model.User, subject string, message string) error
	Send(ctx context.Context, user *model.User, nfy BaseNotification) error
}

type notificationService struct {
	cfg *config.Config
	db  *gorm.DB
}

func NewNotificationService(injector do.Injector) (NotificationService, error) {
	return &notificationService{
		cfg: do.MustInvoke[*config.Config](injector),
		db:  do.MustInvoke[*gorm.DB](injector),
	}, nil
}

func (s *notificationService) SendRaw(ctx context.Context, user *model.User, subject string, message string) error {
	nfy := model.Notification{
		UserID:  user.ID,
		Type:    "raw",
		Subject: subject,
		Msg:     message,
	}

	err := gorm.G[model.Notification](s.db).Create(ctx, &nfy)
	if err != nil {
		return fmt.Errorf("could not save notification: %w", err)
	}

	n := notify.NewWithServices(s.getEmailService(user)...)
	if err := n.Send(ctx, subject, message); err != nil {
		return err
	}

	return nil
}

func (s *notificationService) isChannelEnabled(ctx context.Context, user *model.User, method string, nType string) bool {
	var settings model.UserNotificationSettings
	err := s.db.WithContext(ctx).Where("user_id = ? AND method = ?", user.ID, method).First(&settings).Error
	if err != nil {
		return true
	}
	return settings.IsEnabled(nType)
}

func (s *notificationService) Send(ctx context.Context, user *model.User, in BaseNotification) error {
	nType := in.GetType()

	if in.AllowDB() && s.isChannelEnabled(ctx, user, "database", nType) {
		nfy := model.Notification{
			UserID:  user.ID,
			Type:    nType,
			Subject: in.GetSubject(),
			Msg:     in.GetMessage(),
			Meta:    in.GetMeta(),
		}

		err := gorm.G[model.Notification](s.db).Create(ctx, &nfy)
		if err != nil {
			return fmt.Errorf("could not save notification: %w", err)
		}
	}

	services := []notify.Notifier{}
	if in.AllowEmail() && s.isChannelEnabled(ctx, user, "mail", nType) {
		services = append(services, s.getEmailService(user)...)
	}

	if in.AllowWebpush() && s.isChannelEnabled(ctx, user, "webpush", nType) {
		services = append(services, s.getWebpushService(user)...)
	}

	if len(services) > 0 {
		n := notify.NewWithServices(services...)
		if err := n.Send(ctx, in.GetSubject(), in.GetMessage()); err != nil {
			return err
		}
	}

	return nil
}

func (s *notificationService) getEmailService(receiver *model.User) []notify.Notifier {
	services := []notify.Notifier{}

	if s.cfg.SmtpHost != "" && s.cfg.MailSenderAddress != "" {
		mailService := mail.New(s.cfg.MailSenderAddress, s.cfg.SmtpHost)
		mailService.AddReceivers(receiver.Email)
		services = append(services, mailService)
	} else if s.cfg.MailjetPublicKey != "" && s.cfg.MailjetPrivateKey != "" {
		mailService := notification.NewMailjet(s.cfg.MailjetPublicKey, s.cfg.MailjetPrivateKey, s.cfg.MailSenderAddress, s.cfg.MailSenderName)
		mailService.AddReceivers(receiver.Email)
		services = append(services, mailService)
	}

	return services
}

func (s *notificationService) getWebpushService(receiver *model.User) []notify.Notifier {
	services := []notify.Notifier{}

	var userConfig model.UserNotificationSettings
	if err := s.db.Where("user_id = ? AND method = 'webpush'", receiver.ID).First(&userConfig).Error; err != nil {
		return services
	}

	if userConfig.MethodSettings == nil {
		return services
	}

	if s.cfg.VapidPrivateKey != "" && s.cfg.VapidPublicKey != "" {
		var receiver webpush.Subscription
		if err := json.Unmarshal(*userConfig.MethodSettings, &receiver); err != nil {
			return services
		}

		webpushSvc := webpush.New(s.cfg.VapidPublicKey, s.cfg.VapidPrivateKey)
		webpushSvc.AddReceivers(receiver)
		services = append(services, webpushSvc)
	}

	return services
}
