package controller

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/config"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/model/dto"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/AepyornisNet/aepyornis/pkg/worker"
	"github.com/alexedwards/scs/v2"
	gorand "github.com/cat-dealer/go-rand/v2"
	"github.com/labstack/echo/v4"
	"github.com/samber/do/v2"
	"github.com/vgarvardt/gue/v6"
	"gorm.io/gorm"
)

const (
	hammerheadOAuthScope     = "activity:read"
	hammerheadAPIBaseURL     = "https://api.hammerhead.io/v1/api"
	hammerheadAuthBaseURL    = "https://api.hammerhead.io/v1/auth"
	hammerheadOAuthStateKey  = "hammerhead_oauth_state"
	hammerheadOAuthUserIDKey = "hammerhead_oauth_user_id"
)

var ErrHammerheadNotConfigured = errors.New("hammerhead oauth is not configured")

type HammerheadController interface {
	GetConnection(c echo.Context) error
	Connect(c echo.Context) error
	Callback(c echo.Context) error
	Disconnect(c echo.Context) error
	Webhook(c echo.Context) error
}

type hammerheadController struct {
	cfg            *config.Config
	db             *gorm.DB
	httpClient     *http.Client
	logger         *slog.Logger
	sessionManager *scs.SessionManager
	userRepo       repository.User
	workerClient   *gue.Client
}

type hammerheadTokenResponse struct {
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	UserID       string `json:"user_id"`
}

type hammerheadWebhookPayload struct {
	ActivityID string `json:"activityId"`
	UserID     string `json:"userId"`
}

type hammerheadConnectionResponse struct {
	Connected        bool   `json:"connected"`
	HammerheadUserID string `json:"hammerhead_user_id,omitempty"`
}

func NewHammerheadController(injector do.Injector) HammerheadController {
	return &hammerheadController{
		cfg:            do.MustInvoke[*config.Config](injector),
		db:             do.MustInvoke[*gorm.DB](injector),
		logger:         do.MustInvoke[*slog.Logger](injector),
		sessionManager: do.MustInvoke[*scs.SessionManager](injector),
		userRepo:       do.MustInvoke[repository.User](injector),
		workerClient:   do.MustInvoke[*gue.Client](injector),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (hc *hammerheadController) GetConnection(c echo.Context) error {
	user := currentUser(c)

	var conn model.HammerheadConnection
	err := hc.db.Where("user_id = ?", user.ID).First(&conn).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusOK, dto.Response[hammerheadConnectionResponse]{
				Results: hammerheadConnectionResponse{Connected: false},
			})
		}

		return renderApiError(c, http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, dto.Response[hammerheadConnectionResponse]{
		Results: hammerheadConnectionResponse{
			Connected:        true,
			HammerheadUserID: conn.HammerheadUserID,
		},
	})
}

func (hc *hammerheadController) Connect(c echo.Context) error {
	user := currentUser(c)
	cfg := hc.cfg
	if cfg.HammerheadClientID == "" || cfg.HammerheadSecret == "" {
		return renderApiError(c, http.StatusBadRequest, ErrHammerheadNotConfigured)
	}

	state := gorand.String(32, gorand.GetAlphaNumericPool())
	hc.sessionManager.Put(c.Request().Context(), hammerheadOAuthStateKey, state)
	hc.sessionManager.Put(c.Request().Context(), hammerheadOAuthUserIDKey, strconv.FormatUint(user.ID, 10))

	authorizeURL := url.URL{
		Scheme: "https",
		Host:   "api.hammerhead.io",
		Path:   "/v1/auth/oauth/authorize",
	}
	values := authorizeURL.Query()
	values.Set("client_id", cfg.HammerheadClientID)
	values.Set("redirect_uri", hc.redirectURI(c))
	values.Set("response_type", "code")
	values.Set("scope", hammerheadOAuthScope)
	values.Set("state", state)
	authorizeURL.RawQuery = values.Encode()

	return c.JSON(http.StatusOK, dto.Response[map[string]string]{
		Results: map[string]string{"authorize_url": authorizeURL.String()},
	})
}

func (hc *hammerheadController) Callback(c echo.Context) error {
	user := currentUser(c)

	if oauthErr := c.QueryParam("error"); oauthErr != "" {
		return hc.redirectToAppsPage(c, "oauth_error")
	}

	state := c.QueryParam("state")
	code := c.QueryParam("code")
	if state == "" || code == "" {
		return hc.redirectToAppsPage(c, "invalid_callback")
	}

	savedState := hc.sessionManager.GetString(c.Request().Context(), hammerheadOAuthStateKey)
	savedUserID := hc.sessionManager.GetString(c.Request().Context(), hammerheadOAuthUserIDKey)
	if savedState == "" || state != savedState || savedUserID != strconv.FormatUint(user.ID, 10) {
		return hc.redirectToAppsPage(c, "invalid_state")
	}

	tokenResp, err := hc.exchangeCodeForToken(c.Request().Context(), code, hc.redirectURI(c))
	if err != nil {
		hc.logger.Warn("Hammerhead token exchange failed", "error", err)
		return hc.redirectToAppsPage(c, "token_exchange_failed")
	}

	if tokenResp.UserID == "" || tokenResp.AccessToken == "" || tokenResp.RefreshToken == "" {
		return hc.redirectToAppsPage(c, "invalid_token_response")
	}

	var existingByHammerhead model.HammerheadConnection
	err = hc.db.Where("hammerhead_user_id = ? AND user_id <> ?", tokenResp.UserID, user.ID).First(&existingByHammerhead).Error
	if err == nil {
		return hc.redirectToAppsPage(c, "already_connected")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return hc.redirectToAppsPage(c, "save_failed")
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	if tokenResp.ExpiresIn <= 0 {
		expiresAt = time.Now().Add(6 * time.Hour)
	}

	var conn model.HammerheadConnection
	err = hc.db.Where("user_id = ?", user.ID).First(&conn).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return hc.redirectToAppsPage(c, "save_failed")
	}

	conn.UserID = user.ID
	conn.HammerheadUserID = tokenResp.UserID
	conn.AccessToken = tokenResp.AccessToken
	conn.RefreshToken = tokenResp.RefreshToken
	conn.Scope = hammerheadOAuthScope
	conn.ExpiresAt = expiresAt

	if err := hc.db.Save(&conn).Error; err != nil {
		hc.logger.Warn("Failed to save Hammerhead connection", "error", err)
		return hc.redirectToAppsPage(c, "save_failed")
	}

	return hc.redirectToAppsPage(c, "connected")
}

func (hc *hammerheadController) Disconnect(c echo.Context) error {
	user := currentUser(c)

	var conn model.HammerheadConnection
	err := hc.db.Where("user_id = ?", user.ID).First(&conn).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusOK, dto.Response[map[string]string]{
				Results: map[string]string{"message": "Hammerhead was not connected"},
			})
		}

		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if cfg := hc.cfg; cfg.HammerheadClientID != "" && cfg.HammerheadSecret != "" {
		if err := hc.deauthorize(c.Request().Context(), conn.AccessToken); err != nil {
			hc.logger.Warn("Hammerhead deauthorize failed", "error", err)
		}
	}

	if err := hc.db.Delete(&conn).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, dto.Response[map[string]string]{
		Results: map[string]string{"message": "Hammerhead disconnected"},
	})
}

func (hc *hammerheadController) Webhook(c echo.Context) error {
	payloadRaw, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	if !hc.validWebhookSignature(payloadRaw, c.Request().Header.Get("X-Hmac-Signature")) {
		return renderApiError(c, http.StatusUnauthorized, errors.New("invalid webhook signature"))
	}

	var payload hammerheadWebhookPayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	if payload.UserID == "" || payload.ActivityID == "" {
		return renderApiError(c, http.StatusBadRequest, errors.New("missing webhook fields"))
	}

	var conn model.HammerheadConnection
	err = hc.db.Where("hammerhead_user_id = ?", payload.UserID).First(&conn).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusOK, dto.Response[map[string]string]{
				Results: map[string]string{"message": "ignored: no connection"},
			})
		}

		return renderApiError(c, http.StatusInternalServerError, err)
	}

	user, err := hc.userRepo.GetByID(conn.UserID)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	fitData, err := hc.getActivityFile(c.Request().Context(), &conn, payload.ActivityID)
	if err != nil {
		hc.logger.Warn("Hammerhead activity fetch failed", "activity_id", payload.ActivityID, "error", err)
		return c.JSON(http.StatusOK, dto.Response[map[string]string]{
			Results: map[string]string{"message": "ignored: activity fetch failed"},
		})
	}

	user.Profile.User = user
	workouts, addErr := user.Profile.AddWorkout(hc.db, model.WorkoutTypeAutoDetect, "Imported from Hammerhead", payload.ActivityID+".fit", fitData)
	if len(addErr) > 0 {
		allDuplicates := true
		for _, itemErr := range addErr {
			if !errors.Is(itemErr, model.ErrWorkoutAlreadyExists) {
				allDuplicates = false
				hc.logger.Warn("Hammerhead workout import failed", "error", itemErr)
			}
		}

		if !allDuplicates {
			return c.JSON(http.StatusOK, dto.Response[map[string]string]{
				Results: map[string]string{"message": "ignored: workout import failed"},
			})
		}
	}

	for _, workout := range workouts {
		if err := worker.EnqueueWorkoutUpdate(c.Request().Context(), hc.workerClient, workout.ID); err != nil {
			hc.logger.Warn("Failed to enqueue workout update", "workout_id", workout.ID, "error", err)
		}
	}

	return c.JSON(http.StatusOK, dto.Response[map[string]string]{
		Results: map[string]string{"message": "ok"},
	})
}

func (hc *hammerheadController) exchangeCodeForToken(ctx context.Context, code, redirectURI string) (*hammerheadTokenResponse, error) {
	cfg := hc.cfg
	if cfg.HammerheadClientID == "" || cfg.HammerheadSecret == "" {
		return nil, ErrHammerheadNotConfigured
	}

	values := url.Values{}
	values.Set("client_id", cfg.HammerheadClientID)
	values.Set("client_secret", cfg.HammerheadSecret)
	values.Set("grant_type", "authorization_code")
	values.Set("code", code)
	values.Set("redirect_uri", redirectURI)

	return hc.requestToken(ctx, values)
}

func (hc *hammerheadController) refreshToken(ctx context.Context, conn *model.HammerheadConnection) error {
	cfg := hc.cfg
	if cfg.HammerheadClientID == "" || cfg.HammerheadSecret == "" {
		return ErrHammerheadNotConfigured
	}

	values := url.Values{}
	values.Set("client_id", cfg.HammerheadClientID)
	values.Set("client_secret", cfg.HammerheadSecret)
	values.Set("grant_type", "refresh_token")
	values.Set("refresh_token", conn.RefreshToken)

	tokenResp, err := hc.requestToken(ctx, values)
	if err != nil {
		return err
	}

	if tokenResp.AccessToken == "" || tokenResp.RefreshToken == "" {
		return errors.New("invalid refresh response")
	}

	conn.AccessToken = tokenResp.AccessToken
	conn.RefreshToken = tokenResp.RefreshToken
	if tokenResp.UserID != "" {
		conn.HammerheadUserID = tokenResp.UserID
	}
	if tokenResp.ExpiresIn > 0 {
		conn.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	} else {
		conn.ExpiresAt = time.Now().Add(6 * time.Hour)
	}

	return hc.db.Save(conn).Error
}

func (hc *hammerheadController) requestToken(ctx context.Context, values url.Values) (*hammerheadTokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hammerheadAuthBaseURL+"/oauth/token", strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

	resp, err := hc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("token request failed with status %d", resp.StatusCode)
	}

	var tokenResp hammerheadTokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

func (hc *hammerheadController) getActivityFile(ctx context.Context, conn *model.HammerheadConnection, activityID string) ([]byte, error) {
	body, statusCode, err := hc.fetchActivityFile(ctx, conn.AccessToken, activityID)
	if err == nil {
		return body, nil
	}

	if statusCode != http.StatusUnauthorized && statusCode != http.StatusForbidden {
		return nil, err
	}

	if refreshErr := hc.refreshToken(ctx, conn); refreshErr != nil {
		return nil, refreshErr
	}

	body, _, err = hc.fetchActivityFile(ctx, conn.AccessToken, activityID)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (hc *hammerheadController) fetchActivityFile(ctx context.Context, accessToken, activityID string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, hammerheadAPIBaseURL+"/activities/"+url.PathEscape(activityID)+"/file", nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+accessToken)

	resp, err := hc.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, resp.StatusCode, fmt.Errorf("activity file request failed with status %d", resp.StatusCode)
	}

	return body, resp.StatusCode, nil
}

func (hc *hammerheadController) deauthorize(ctx context.Context, accessToken string) error {
	cfg := hc.cfg
	values := url.Values{}
	values.Set("client_id", cfg.HammerheadClientID)
	values.Set("client_secret", cfg.HammerheadSecret)
	values.Set("token", accessToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hammerheadAuthBaseURL+"/oauth/deauthorize", strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

	resp, err := hc.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("deauthorize request failed with status %d", resp.StatusCode)
	}

	return nil
}

func (hc *hammerheadController) validWebhookSignature(payload []byte, signature string) bool {
	secret := hc.cfg.HammerheadWebhook
	if secret == "" {
		return false
	}

	if signature == "" {
		return false
	}

	rawSignature := strings.TrimSpace(signature)
	if strings.Contains(rawSignature, "=") {
		parts := strings.SplitN(rawSignature, "=", 2)
		rawSignature = parts[1]
	}

	provided, err := hex.DecodeString(rawSignature)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := mac.Sum(nil)

	return hmac.Equal(expected, provided)
}

func (hc *hammerheadController) redirectURI(c echo.Context) string {
	if configured := hc.cfg.HammerheadRedirect; configured != "" {
		return configured
	}

	scheme := c.Scheme()
	host := c.Request().Host
	if scheme == "" {
		scheme = "https"
	}

	path := joinWithWebRoot(hc.cfg.WebRoot, "/api/v2/profile/apps/hammerhead/callback")

	return scheme + "://" + host + path
}

func (hc *hammerheadController) redirectToAppsPage(c echo.Context, status string) error {
	target := joinWithWebRoot(hc.cfg.WebRoot, "/profile/settings/apps")
	if status != "" {
		target = target + "?hammerhead=" + url.QueryEscape(status)
	}

	return c.Redirect(http.StatusFound, target)
}

func joinWithWebRoot(webRoot string, path string) string {
	if path == "" {
		return "/"
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	root := strings.TrimSpace(webRoot)
	if root == "" || root == "/" {
		return path
	}

	return strings.TrimRight(root, "/") + path
}
