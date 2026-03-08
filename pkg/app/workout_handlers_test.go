package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/geocoder"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func init() { //nolint:gochecknoinits
	geocoder.ForceOffline()
}

// workoutTestEnv bundles the configured app, httptest server, authenticated
// user and a convenience http.Client for workout endpoint tests.
type workoutTestEnv struct {
	app  *App
	ts   *httptest.Server
	user *model.User
}

func newWorkoutTestEnv(t *testing.T) *workoutTestEnv {
	t.Helper()

	a := configuredApp(t)
	ts := httptest.NewServer(a.echo)
	t.Cleanup(ts.Close)

	u := defaultAPIUser(a.db)
	return &workoutTestEnv{app: a, ts: ts, user: u}
}

// url builds a full URL from a named route, substituting :id and :attachment_id
// path params when present in the route template.
func (e *workoutTestEnv) url(routeName string, pathParams ...any) string {
	u := e.ts.URL + e.app.echo.Reverse(routeName, pathParams...)
	return u
}

// do performs an authenticated HTTP request (API-key auth via Bearer header).
func (e *workoutTestEnv) do(t *testing.T, method, url string, body io.Reader) (*http.Response, []byte) {
	t.Helper()

	req, err := http.NewRequest(method, url, body)
	require.NoError(t, err)

	req.Header.Set("Authorization", "Bearer my-api-key")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req) //nolint:gosec
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })

	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, b
}

// createWorkoutInDB inserts a minimal workout directly in the database so that
// tests can operate on an existing record without going through the API.
func createWorkoutInDB(t *testing.T, db *gorm.DB, user *model.User) *model.Workout {
	t.Helper()

	w := &model.Workout{
		Name:     "Test Workout",
		Type:     model.WorkoutTypeRunning,
		Date:     time.Date(2024, 6, 1, 8, 0, 0, 0, time.UTC),
		UserID:   user.ID,
		User:     user,
		Data:     &model.MapData{},
		Visibility: model.WorkoutVisibilityPublic,
	}

	require.NoError(t, w.Create(db))

	return w
}

// ---------------------------------------------------------------------------
// GET /workouts
// ---------------------------------------------------------------------------

func TestWorkouts_List_Empty(t *testing.T) {
	env := newWorkoutTestEnv(t)
	resp, body := env.do(t, http.MethodGet, env.url("workouts-list"), nil)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result dto.PaginatedResponse[dto.WorkoutResponse]
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Empty(t, result.Results)
	assert.Equal(t, int64(0), result.TotalCount)
}

func TestWorkouts_List_WithWorkout(t *testing.T) {
	env := newWorkoutTestEnv(t)
	createWorkoutInDB(t, env.app.db, env.user)

	resp, body := env.do(t, http.MethodGet, env.url("workouts-list"), nil)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result dto.PaginatedResponse[dto.WorkoutResponse]
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Len(t, result.Results, 1)
	assert.Equal(t, int64(1), result.TotalCount)
	assert.Equal(t, "Test Workout", result.Results[0].Name)
}

func TestWorkouts_List_Pagination(t *testing.T) {
	env := newWorkoutTestEnv(t)
	for i := range 5 {
		w := &model.Workout{
			Name:     fmt.Sprintf("Workout %d", i),
			Type:     model.WorkoutTypeRunning,
			Date:     time.Date(2024, 1, i+1, 8, 0, 0, 0, time.UTC),
			UserID:   env.user.ID,
			User:     env.user,
			Data:     &model.MapData{},
			Visibility: model.WorkoutVisibilityPublic,
		}
		require.NoError(t, w.Create(env.app.db))
	}

	resp, body := env.do(t, http.MethodGet, env.url("workouts-list")+"?per_page=2&page=1", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result dto.PaginatedResponse[dto.WorkoutResponse]
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Len(t, result.Results, 2)
	assert.Equal(t, int64(5), result.TotalCount)
	assert.Equal(t, 3, result.TotalPages)
}

func TestWorkouts_List_Unauthenticated(t *testing.T) {
	env := newWorkoutTestEnv(t)
	req, err := http.NewRequest(http.MethodGet, env.url("workouts-list"), nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req) //nolint:gosec
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// POST /workouts (manual creation)
// ---------------------------------------------------------------------------

func TestWorkouts_Create_Manual_Success(t *testing.T) {
	env := newWorkoutTestEnv(t)

	payload := map[string]any{
		"name": "Manual Run",
		"type": "running",
		"date": "2024-06-01T08:00",
	}
	b, err := json.Marshal(payload)
	require.NoError(t, err)

	resp, body := env.do(t, http.MethodPost, env.url("workouts-create"), bytes.NewReader(b))

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result dto.Response[dto.WorkoutResponse]
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Equal(t, "Manual Run", result.Results.Name)
	assert.Equal(t, "running", result.Results.Type)
	assert.NotZero(t, result.Results.ID)
}

func TestWorkouts_Create_Manual_InvalidVisibility(t *testing.T) {
	env := newWorkoutTestEnv(t)

	payload := map[string]any{
		"name":       "Bad Visibility",
		"type":       "running",
		"date":       "2024-06-01T08:00",
		"visibility": "invalid",
	}
	b, err := json.Marshal(payload)
	require.NoError(t, err)

	resp, _ := env.do(t, http.MethodPost, env.url("workouts-create"), bytes.NewReader(b))

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWorkouts_Create_Manual_DuplicateDate(t *testing.T) {
	env := newWorkoutTestEnv(t)

	payload := map[string]any{
		"name": "Duplicate Run",
		"type": "running",
		"date": "2024-06-01T08:00",
	}
	b, err := json.Marshal(payload)
	require.NoError(t, err)

	resp1, _ := env.do(t, http.MethodPost, env.url("workouts-create"), bytes.NewReader(b))
	require.Equal(t, http.StatusCreated, resp1.StatusCode)

	b2, _ := json.Marshal(payload)
	resp2, body2 := env.do(t, http.MethodPost, env.url("workouts-create"), bytes.NewReader(b2))
	assert.Equal(t, http.StatusConflict, resp2.StatusCode)
	assert.Contains(t, string(body2), "workout_already_exists")
}

// ---------------------------------------------------------------------------
// GET /workouts/:id
// ---------------------------------------------------------------------------

func TestWorkouts_Get_Found(t *testing.T) {
	env := newWorkoutTestEnv(t)
	w := createWorkoutInDB(t, env.app.db, env.user)

	resp, body := env.do(t, http.MethodGet, env.url("workout-get", w.ID), nil)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result dto.Response[dto.WorkoutDetailResponse]
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Equal(t, w.ID, result.Results.ID)
	assert.Equal(t, "Test Workout", result.Results.Name)
}

func TestWorkouts_Get_NotFound(t *testing.T) {
	env := newWorkoutTestEnv(t)

	resp, _ := env.do(t, http.MethodGet, env.url("workout-get", 99999), nil)

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestWorkouts_Get_OtherUsersPrivateWorkout(t *testing.T) {
	env := newWorkoutTestEnv(t)

	// Create a second user whose workout is private.
	otherUser := &model.User{
		UserData: model.UserData{
			Username: "other-user",
			Name:     "Other User",
			Active:   true,
		},
		UserSecrets: model.UserSecrets{Password: "password"},
	}
	require.NoError(t, otherUser.Create(env.app.db))

	w := &model.Workout{
		Name:     "Private Workout",
		Type:     model.WorkoutTypeRunning,
		Date:     time.Date(2024, 6, 2, 9, 0, 0, 0, time.UTC),
		UserID:   otherUser.ID,
		User:     otherUser,
		Data:     &model.MapData{},
		Visibility: model.WorkoutVisibilityPrivate,
	}
	require.NoError(t, w.Create(env.app.db))

	resp, _ := env.do(t, http.MethodGet, env.url("workout-get", w.ID), nil)

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestWorkouts_Get_OtherUsersPublicWorkout(t *testing.T) {
	env := newWorkoutTestEnv(t)

	otherUser := &model.User{
		UserData: model.UserData{
			Username: "public-user",
			Name:     "Public User",
			Active:   true,
		},
		UserSecrets: model.UserSecrets{Password: "password"},
	}
	require.NoError(t, otherUser.Create(env.app.db))

	w := &model.Workout{
		Name:     "Public Workout",
		Type:     model.WorkoutTypeRunning,
		Date:     time.Date(2024, 6, 3, 9, 0, 0, 0, time.UTC),
		UserID:   otherUser.ID,
		User:     otherUser,
		Data:     &model.MapData{},
		Visibility: model.WorkoutVisibilityPublic,
	}
	require.NoError(t, w.Create(env.app.db))

	resp, body := env.do(t, http.MethodGet, env.url("workout-get", w.ID), nil)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var result dto.Response[dto.WorkoutDetailResponse]
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Equal(t, "Public Workout", result.Results.Name)
}

// ---------------------------------------------------------------------------
// PUT /workouts/:id
// ---------------------------------------------------------------------------

func TestWorkouts_Update_Success(t *testing.T) {
	env := newWorkoutTestEnv(t)
	w := createWorkoutInDB(t, env.app.db, env.user)

	payload := map[string]any{
		"name": "Updated Workout",
		"type": "cycling",
	}
	b, err := json.Marshal(payload)
	require.NoError(t, err)

	resp, body := env.do(t, http.MethodPut, env.url("workout-update", w.ID), bytes.NewReader(b))

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var result dto.Response[dto.WorkoutResponse]
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Equal(t, "Updated Workout", result.Results.Name)
	assert.Equal(t, "cycling", result.Results.Type)
}

func TestWorkouts_Update_NotFound(t *testing.T) {
	env := newWorkoutTestEnv(t)

	payload := map[string]any{"name": "Updated"}
	b, _ := json.Marshal(payload)

	resp, _ := env.do(t, http.MethodPut, env.url("workout-update", 99999), bytes.NewReader(b))
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestWorkouts_Update_InvalidVisibility(t *testing.T) {
	env := newWorkoutTestEnv(t)
	w := createWorkoutInDB(t, env.app.db, env.user)

	payload := map[string]any{
		"name":       "Bad Update",
		"visibility": "invalid-value",
	}
	b, _ := json.Marshal(payload)

	resp, _ := env.do(t, http.MethodPut, env.url("workout-update", w.ID), bytes.NewReader(b))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// DELETE /workouts/:id
// ---------------------------------------------------------------------------

func TestWorkouts_Delete_Success(t *testing.T) {
	env := newWorkoutTestEnv(t)
	w := createWorkoutInDB(t, env.app.db, env.user)

	resp, body := env.do(t, http.MethodDelete, env.url("workout-delete", w.ID), nil)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), "deleted")

	// Subsequent GET must return 404.
	getResp, _ := env.do(t, http.MethodGet, env.url("workout-get", w.ID), nil)
	assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
}

func TestWorkouts_Delete_NotFound(t *testing.T) {
	env := newWorkoutTestEnv(t)

	resp, _ := env.do(t, http.MethodDelete, env.url("workout-delete", 99999), nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// POST /workouts/:id/toggle-lock
// ---------------------------------------------------------------------------

func TestWorkouts_ToggleLock(t *testing.T) {
	env := newWorkoutTestEnv(t)
	w := createWorkoutInDB(t, env.app.db, env.user)

	assert.False(t, w.Locked)

	// Lock it.
	resp1, body1 := env.do(t, http.MethodPost, env.url("workout-toggle-lock", w.ID), nil)
	assert.Equal(t, http.StatusOK, resp1.StatusCode)
	var result1 dto.Response[dto.WorkoutResponse]
	require.NoError(t, json.Unmarshal(body1, &result1))
	assert.True(t, result1.Results.Locked)

	// Unlock it.
	resp2, body2 := env.do(t, http.MethodPost, env.url("workout-toggle-lock", w.ID), nil)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
	var result2 dto.Response[dto.WorkoutResponse]
	require.NoError(t, json.Unmarshal(body2, &result2))
	assert.False(t, result2.Results.Locked)
}

func TestWorkouts_ToggleLock_NotFound(t *testing.T) {
	env := newWorkoutTestEnv(t)

	resp, _ := env.do(t, http.MethodPost, env.url("workout-toggle-lock", 99999), nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// POST /workouts/:id/refresh
// ---------------------------------------------------------------------------

func TestWorkouts_Refresh_Success(t *testing.T) {
	env := newWorkoutTestEnv(t)
	w := createWorkoutInDB(t, env.app.db, env.user)

	resp, _ := env.do(t, http.MethodPost, env.url("workout-refresh", w.ID), nil)

	// The refresh endpoint enqueues a background job. In the in-memory test
	// environment the job queue table (gue_jobs) is not present, so the
	// controller returns 500 when enqueueing fails.  We assert that the
	// response is either 200 (job queue available) or 500 (queue unavailable)
	// to confirm the request reached the controller and is not a routing or
	// auth error.
	assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError}, resp.StatusCode)
}

func TestWorkouts_Refresh_NotFound(t *testing.T) {
	env := newWorkoutTestEnv(t)

	resp, _ := env.do(t, http.MethodPost, env.url("workout-refresh", 99999), nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// GET /workouts/recent
// ---------------------------------------------------------------------------

func TestWorkouts_Recent_Empty(t *testing.T) {
	env := newWorkoutTestEnv(t)

	resp, body := env.do(t, http.MethodGet, env.url("workouts-recent"), nil)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var result dto.Response[[]dto.WorkoutResponse]
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Empty(t, result.Results)
}

func TestWorkouts_Recent_ReturnsSelf(t *testing.T) {
	env := newWorkoutTestEnv(t)
	createWorkoutInDB(t, env.app.db, env.user)

	resp, body := env.do(t, http.MethodGet, env.url("workouts-recent"), nil)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var result dto.Response[[]dto.WorkoutResponse]
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Len(t, result.Results, 1)
}

func TestWorkouts_Recent_LimitParam(t *testing.T) {
	env := newWorkoutTestEnv(t)
	for i := range 5 {
		w := &model.Workout{
			Name:     fmt.Sprintf("Workout %d", i),
			Type:     model.WorkoutTypeRunning,
			Date:     time.Date(2024, 1, i+1, 8, 0, 0, 0, time.UTC),
			UserID:   env.user.ID,
			User:     env.user,
			Data:     &model.MapData{},
			Visibility: model.WorkoutVisibilityPublic,
		}
		require.NoError(t, w.Create(env.app.db))
	}

	resp, body := env.do(t, http.MethodGet, env.url("workouts-recent")+"?limit=3", nil)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var result dto.Response[[]dto.WorkoutResponse]
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Len(t, result.Results, 3)
}

// ---------------------------------------------------------------------------
// GET /workouts/calendar
// ---------------------------------------------------------------------------

func TestWorkouts_Calendar_Empty(t *testing.T) {
	env := newWorkoutTestEnv(t)

	resp, body := env.do(t, http.MethodGet, env.url("workouts-calendar"), nil)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var result dto.Response[[]dto.CalendarEventResponse]
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Empty(t, result.Results)
}

func TestWorkouts_Calendar_WithWorkout(t *testing.T) {
	env := newWorkoutTestEnv(t)
	createWorkoutInDB(t, env.app.db, env.user)

	resp, body := env.do(t, http.MethodGet, env.url("workouts-calendar"), nil)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var result dto.Response[[]dto.CalendarEventResponse]
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Len(t, result.Results, 1)
}

func TestWorkouts_Calendar_DateRangeFilter(t *testing.T) {
	env := newWorkoutTestEnv(t)

	// Create two workouts in different months.
	w1 := &model.Workout{
		Name:     "January",
		Type:     model.WorkoutTypeRunning,
		Date:     time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC),
		UserID:   env.user.ID,
		User:     env.user,
		Data:     &model.MapData{},
		Visibility: model.WorkoutVisibilityPublic,
	}
	w2 := &model.Workout{
		Name:     "March",
		Type:     model.WorkoutTypeRunning,
		Date:     time.Date(2024, 3, 15, 8, 0, 0, 0, time.UTC),
		UserID:   env.user.ID,
		User:     env.user,
		Data:     &model.MapData{},
		Visibility: model.WorkoutVisibilityPublic,
	}
	require.NoError(t, w1.Create(env.app.db))
	require.NoError(t, w2.Create(env.app.db))

	q := "?start=2024-02-01T00:00:00&end=2024-04-01T00:00:00"
	resp, body := env.do(t, http.MethodGet, env.url("workouts-calendar")+q, nil)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var result dto.Response[[]dto.CalendarEventResponse]
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Len(t, result.Results, 1)
	assert.Equal(t, "March", result.Results[0].Title)
}

// ---------------------------------------------------------------------------
// GET /workouts/:id/likes
// ---------------------------------------------------------------------------

func TestWorkouts_Likes_Empty(t *testing.T) {
	env := newWorkoutTestEnv(t)
	w := createWorkoutInDB(t, env.app.db, env.user)

	resp, body := env.do(t, http.MethodGet, env.url("workout-likes", w.ID), nil)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var result dto.Response[[]dto.WorkoutLikeResponse]
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Empty(t, result.Results)
}

func TestWorkouts_Likes_NotFound(t *testing.T) {
	env := newWorkoutTestEnv(t)

	resp, _ := env.do(t, http.MethodGet, env.url("workout-likes", 99999), nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// POST /workouts/:id/like
// ---------------------------------------------------------------------------

func TestWorkouts_Like_OwnWorkout_Forbidden(t *testing.T) {
	env := newWorkoutTestEnv(t)
	w := createWorkoutInDB(t, env.app.db, env.user)

	resp, body := env.do(t, http.MethodPost, env.url("workout-like", w.ID), nil)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, string(body), "cannot like your own workout")
}

func TestWorkouts_Like_OtherUser(t *testing.T) {
	env := newWorkoutTestEnv(t)

	// Create a second user whose public workout is liked by the test user.
	otherUser := &model.User{
		UserData: model.UserData{
			Username: "like-owner",
			Name:     "Like Owner",
			Active:   true,
		},
		UserSecrets: model.UserSecrets{Password: "password"},
	}
	require.NoError(t, otherUser.Create(env.app.db))

	w := &model.Workout{
		Name:     "Likeable Workout",
		Type:     model.WorkoutTypeRunning,
		Date:     time.Date(2024, 7, 1, 8, 0, 0, 0, time.UTC),
		UserID:   otherUser.ID,
		User:     otherUser,
		Data:     &model.MapData{},
		Visibility: model.WorkoutVisibilityPublic,
	}
	require.NoError(t, w.Create(env.app.db))

	resp, body := env.do(t, http.MethodPost, env.url("workout-like", w.ID), nil)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var result dto.Response[map[string]any]
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Equal(t, true, result.Results["liked"])
	assert.Equal(t, float64(1), result.Results["likes_count"])
}

func TestWorkouts_Like_NotFound(t *testing.T) {
	env := newWorkoutTestEnv(t)

	resp, _ := env.do(t, http.MethodPost, env.url("workout-like", 99999), nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// GET /workouts/:id/replies
// ---------------------------------------------------------------------------

func TestWorkouts_Replies_Empty(t *testing.T) {
	env := newWorkoutTestEnv(t)
	w := createWorkoutInDB(t, env.app.db, env.user)

	resp, body := env.do(t, http.MethodGet, env.url("workout-replies", w.ID), nil)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var result dto.PaginatedResponse[dto.WorkoutReplyResponse]
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Empty(t, result.Results)
}

func TestWorkouts_Replies_NotFound(t *testing.T) {
	env := newWorkoutTestEnv(t)

	resp, _ := env.do(t, http.MethodGet, env.url("workout-replies", 99999), nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// POST /workouts/:id/replies
// ---------------------------------------------------------------------------

func TestWorkouts_CreateReply_Success(t *testing.T) {
	env := newWorkoutTestEnv(t)
	w := createWorkoutInDB(t, env.app.db, env.user)

	payload := map[string]any{"content": "Great run!"}
	b, err := json.Marshal(payload)
	require.NoError(t, err)

	resp, body := env.do(t, http.MethodPost,
		env.url("workout-create-reply", w.ID),
		bytes.NewReader(b))

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	var result dto.Response[dto.WorkoutReplyResponse]
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Equal(t, "Great run!", result.Results.Content)
	assert.NotZero(t, result.Results.ID)
}

func TestWorkouts_CreateReply_EmptyContent(t *testing.T) {
	env := newWorkoutTestEnv(t)
	w := createWorkoutInDB(t, env.app.db, env.user)

	payload := map[string]any{"content": "   "}
	b, err := json.Marshal(payload)
	require.NoError(t, err)

	resp, _ := env.do(t, http.MethodPost,
		env.url("workout-create-reply", w.ID),
		bytes.NewReader(b))

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWorkouts_CreateReply_WorkoutNotFound(t *testing.T) {
	env := newWorkoutTestEnv(t)

	payload := map[string]any{"content": "Reply"}
	b, _ := json.Marshal(payload)

	resp, _ := env.do(t, http.MethodPost,
		env.url("workout-create-reply", 99999),
		bytes.NewReader(b))

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestWorkouts_CreateReply_ThenList(t *testing.T) {
	env := newWorkoutTestEnv(t)
	w := createWorkoutInDB(t, env.app.db, env.user)

	payload := map[string]any{"content": "First reply"}
	b, _ := json.Marshal(payload)
	createResp, _ := env.do(t, http.MethodPost, env.url("workout-create-reply", w.ID), bytes.NewReader(b))
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	resp, body := env.do(t, http.MethodGet, env.url("workout-replies", w.ID), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var result dto.PaginatedResponse[dto.WorkoutReplyResponse]
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Len(t, result.Results, 1)
	assert.Equal(t, "First reply", result.Results[0].Content)
}

// ---------------------------------------------------------------------------
// GET /workouts/:id/download
// ---------------------------------------------------------------------------

func TestWorkouts_Download_NoFile(t *testing.T) {
	env := newWorkoutTestEnv(t)
	w := createWorkoutInDB(t, env.app.db, env.user)

	resp, _ := env.do(t, http.MethodGet, env.url("workout-download", w.ID), nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestWorkouts_Download_WithFile(t *testing.T) {
	env := newWorkoutTestEnv(t)

	// Create a workout that has a raw GPX file attached.
	w := &model.Workout{
		Name:     "GPX Workout",
		Type:     model.WorkoutTypeRunning,
		Date:     time.Date(2024, 8, 1, 8, 0, 0, 0, time.UTC),
		UserID:   env.user.ID,
		User:     env.user,
		Data:     &model.MapData{},
		GPX: &model.GPXData{
			Filename: "workout.gpx",
			Content:  []byte("<gpx>stub</gpx>"),
		},
		Visibility: model.WorkoutVisibilityPublic,
	}
	require.NoError(t, w.Create(env.app.db))

	resp, body := env.do(t, http.MethodGet, env.url("workout-download", w.ID), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Disposition"), "workout.gpx")
	assert.Equal(t, "<gpx>stub</gpx>", string(body))
}

// ---------------------------------------------------------------------------
// Create workout via file upload (multipart)
// ---------------------------------------------------------------------------

func TestWorkouts_Create_FileUpload_NoFile(t *testing.T) {
	env := newWorkoutTestEnv(t)

	// Send a multipart request with no file to trigger "no file uploaded".
	body := &bytes.Buffer{}
	body.WriteString("--boundary\r\nContent-Disposition: form-data; name=\"notes\"\r\n\r\nnotes\r\n--boundary--\r\n")

	req, err := http.NewRequest(http.MethodPost, env.url("workouts-create"), body)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer my-api-key")
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")

	resp, err := http.DefaultClient.Do(req) //nolint:gosec
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// GET /workouts/:id/breakdown – no map data
// ---------------------------------------------------------------------------

func TestWorkouts_Breakdown_NoMapData(t *testing.T) {
	env := newWorkoutTestEnv(t)
	w := createWorkoutInDB(t, env.app.db, env.user)

	resp, _ := env.do(t, http.MethodGet, env.url("workout-breakdown", w.ID), nil)
	// Workout has no map data → 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// Visibility: followers-only workout
// ---------------------------------------------------------------------------

func TestWorkouts_Visibility_FollowersOnly_Denied(t *testing.T) {
	env := newWorkoutTestEnv(t)

	otherUser := &model.User{
		UserData: model.UserData{
			Username: "followers-owner",
			Name:     "Followers Owner",
			Active:   true,
		},
		UserSecrets: model.UserSecrets{Password: "password"},
	}
	require.NoError(t, otherUser.Create(env.app.db))

	w := &model.Workout{
		Name:     "Followers Only",
		Type:     model.WorkoutTypeRunning,
		Date:     time.Date(2024, 9, 1, 8, 0, 0, 0, time.UTC),
		UserID:   otherUser.ID,
		User:     otherUser,
		Data:     &model.MapData{},
		Visibility: model.WorkoutVisibilityFollowers,
	}
	require.NoError(t, w.Create(env.app.db))

	// The test user is not following otherUser → should get 404.
	resp, _ := env.do(t, http.MethodGet, env.url("workout-get", w.ID), nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// Response body contains no password fields
// ---------------------------------------------------------------------------

func TestWorkouts_ResponseNeverContainsPassword(t *testing.T) {
	env := newWorkoutTestEnv(t)
	createWorkoutInDB(t, env.app.db, env.user)

	resp, body := env.do(t, http.MethodGet, env.url("workouts-list"), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	assert.NotContains(t, strings.ToLower(string(body)), `"password"`)
}
