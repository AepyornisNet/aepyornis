package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AepyornisNet/aepyornis/pkg/config"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/notification"
	"github.com/SherClockHolmes/webpush-go"
	"github.com/invopop/ctxi18n"
	"github.com/invopop/ctxi18n/i18n"
	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/mail"
	"github.com/samber/do/v2"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type BaseNotification interface {
	GetType() string
	GetSubject(t *i18n.Locale) string
	GetMessage(t *i18n.Locale) string
	GetMeta() *datatypes.JSON

	AllowDB() bool
	AllowEmail() bool
	AllowWebpush() bool
}

type NotificationService interface {
	SendRaw(ctx context.Context, user *model.User, subject string, message string) error
	Send(ctx context.Context, user *model.User, nfy BaseNotification) error
	SendAdminEmail(ctx context.Context, subject string, message string) error
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

func (s *notificationService) getTranslator(ctx context.Context, user *model.User) *i18n.Locale {
	lang := "en"
	if user != nil && user.Language != "" {
		lang = user.Language
	}

	lctx, err := ctxi18n.WithLocale(ctx, lang)
	if err != nil || lctx == nil {
		lctx, _ = ctxi18n.WithLocale(ctx, "en")
	}
	if lctx == nil {
		return nil
	}
	return ctxi18n.Locale(lctx)
}

func (s *notificationService) isChannelEnabled(ctx context.Context, user *model.User, method string, nType string) bool {
	var settings model.UserNotificationSettings
	err := s.db.WithContext(ctx).Where("user_id = ? AND method = ?", user.ID, method).First(&settings).Error
	if err != nil {
		return false
	}
	return settings.IsEnabled(nType)
}

func (s *notificationService) Send(ctx context.Context, user *model.User, in BaseNotification) error {
	t := s.getTranslator(ctx, user)

	nType := in.GetType()
	subject := in.GetSubject(t)
	message := in.GetMessage(t)

	if in.AllowDB() && s.isChannelEnabled(ctx, user, "database", nType) {
		nfy := model.Notification{
			UserID:  user.ID,
			Type:    nType,
			Subject: subject,
			Msg:     message,
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
		if err := n.Send(ctx, subject, message); err != nil {
			return err
		}
	}

	return nil
}

func (s *notificationService) SendAdminEmail(ctx context.Context, subject string, message string) error {
	adminEmail := strings.TrimSpace(s.cfg.AdminEmail)
	if adminEmail == "" {
		return nil
	}

	services := s.getEmailServiceForAddress(adminEmail)
	if len(services) > 0 {
		n := notify.NewWithServices(services...)
		return n.Send(ctx, subject, message)
	}

	return nil
}

func (s *notificationService) getEmailServiceForAddress(emailAddress string) []notify.Notifier {
	services := []notify.Notifier{}

	if s.cfg.SmtpHost != "" && s.cfg.MailSenderAddress != "" {
		mailService := mail.New(s.cfg.MailSenderAddress, s.cfg.SmtpHost)
		mailService.AddReceivers(emailAddress)
		services = append(services, mailService)
	} else if s.cfg.MailjetPublicKey != "" && s.cfg.MailjetPrivateKey != "" {
		mailService := notification.NewMailjet(s.cfg.MailjetPublicKey, s.cfg.MailjetPrivateKey, s.cfg.MailSenderAddress, s.cfg.MailSenderName)
		mailService.AddReceivers(emailAddress)
		services = append(services, mailService)
	}

	return services
}

func (s *notificationService) getEmailService(receiver *model.User) []notify.Notifier {
	return s.getEmailServiceForAddress(receiver.Email)
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
		var receiverSub webpush.Subscription
		if err := json.Unmarshal(*userConfig.MethodSettings, &receiverSub); err != nil {
			return services
		}

		webpushSvc := notification.NewWebPush(s.cfg.VapidPublicKey, s.cfg.VapidPrivateKey)
		webpushSvc.AddReceivers(receiverSub)
		services = append(services, webpushSvc)
	}

	return services
}
