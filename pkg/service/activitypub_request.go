package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strings"

	"github.com/AepyornisNet/aepyornis/pkg/aputil"
	"github.com/AepyornisNet/aepyornis/pkg/config"
	"github.com/labstack/echo/v4"
	"github.com/samber/do/v2"
)

type ActivityPubRequestService interface {
	HTTPClient() *http.Client
	SendSignedActivity(ctx context.Context, keyID, privateKeyPEM, inbox string, payload []byte) error
}

type activityPubRequestService struct {
	cfg    *config.Config
	router *echo.Echo
	client *http.Client
}

func NewActivityPubRequestService(injector do.Injector) (ActivityPubRequestService, error) {
	cfg := do.MustInvoke[*config.Config](injector)
	router := do.MustInvoke[*echo.Echo](injector)
	if router == nil {
		return nil, errors.New("activitypub request service: missing router")
	}

	svc := &activityPubRequestService{
		cfg:    cfg,
		router: router,
	}
	svc.client = &http.Client{
		Transport: activityPubRoundTripper{
			cfg:    cfg,
			router: router,
			base:   http.DefaultTransport,
		},
	}

	return svc, nil
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

type activityPubRoundTripper struct {
	cfg    *config.Config
	router *echo.Echo
	base   http.RoundTripper
}

func (rt activityPubRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, errors.New("nil request")
	}

	if rt.isLocalActivityPubRequest(req.URL) {
		return rt.roundTripLocal(req)
	}

	base := rt.base
	if base == nil {
		base = http.DefaultTransport
	}

	return base.RoundTrip(req)
}

func (rt activityPubRoundTripper) roundTripLocal(req *http.Request) (*http.Response, error) {
	if rt.router == nil {
		return nil, errors.New("activitypub request service: missing router")
	}

	cloned := req.Clone(req.Context())
	if cloned.Host == "" {
		cloned.Host = cloned.URL.Host
	}

	recorder := httptest.NewRecorder()
	rt.router.ServeHTTP(recorder, cloned)

	return recorder.Result(), nil
}

func (rt activityPubRoundTripper) isLocalActivityPubRequest(target *url.URL) bool {
	if target == nil {
		return false
	}

	localHost := configuredHost(rt.cfg)
	if localHost == "" || !strings.EqualFold(target.Host, localHost) {
		return false
	}

	requestPath := cleanPath(target.Path)
	webRoot := cleanWebRoot(rt.cfg)

	return hasPathPrefix(requestPath, path.Join(webRoot, "ap")) ||
		hasPathPrefix(requestPath, path.Join(webRoot, ".well-known"))
}

func configuredHost(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}

	raw := strings.TrimSpace(cfg.Host)
	if raw == "" {
		return ""
	}

	if parsed, err := url.Parse(raw); err == nil && parsed.Host != "" {
		return parsed.Host
	}

	return strings.TrimPrefix(strings.TrimPrefix(raw, "https://"), "http://")
}

func cleanWebRoot(cfg *config.Config) string {
	if cfg == nil {
		return "/"
	}

	root := path.Join("/", cfg.WebRoot)
	if root == "." {
		return "/"
	}

	return root
}

func cleanPath(p string) string {
	cleaned := path.Clean("/" + strings.TrimPrefix(p, "/"))
	if cleaned == "." {
		return "/"
	}

	return cleaned
}

func hasPathPrefix(actual, prefix string) bool {
	actual = cleanPath(actual)
	prefix = cleanPath(prefix)

	return actual == prefix || strings.HasPrefix(actual, prefix+"/")
}
