package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/AepyornisNet/aepyornis/pkg/aputil"
	"github.com/samber/do/v2"
	"gorm.io/gorm"
)

type ActivityPubRequestService interface {
	HTTPClient() *http.Client
	SendSignedActivity(ctx context.Context, keyID, privateKeyPEM, inbox string, payload []byte) error
	VerifyRequest(req *http.Request) (*aputil.RequestActor, error)
}

type activityPubRequestService struct {
	client *http.Client
	db     *gorm.DB
}

func NewActivityPubRequestService(injector do.Injector) (ActivityPubRequestService, error) {
	db, _ := do.Invoke[*gorm.DB](injector)

	return &activityPubRequestService{
		client: &http.Client{Transport: http.DefaultTransport},
		db:     db,
	}, nil
}

func (s *activityPubRequestService) HTTPClient() *http.Client {
	return s.client
}

func (s *activityPubRequestService) SendSignedActivity(ctx context.Context, keyID, privateKeyPEM, inbox string, payload []byte) error {
	if strings.TrimSpace(inbox) == "" {
		return errors.New("inbox is empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, inbox, bytes.NewReader(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", aputil.ContentType)
	req.Header.Set("Accept", aputil.ContentType)

	if err := aputil.SignRequest(req, privateKeyPEM, keyID); err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("remote inbox rejected activity: %s", resp.Status)
	}

	return nil
}

func (s *activityPubRequestService) VerifyRequest(req *http.Request) (*aputil.RequestActor, error) {
	return aputil.VerifyRequest(req, &activityPubRequestKeyResolver{db: s.db})
}
