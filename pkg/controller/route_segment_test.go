package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	_ "github.com/AepyornisNet/aepyornis/pkg/converters"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/model/dto"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/AepyornisNet/aepyornis/pkg/service"
	"github.com/AepyornisNet/aepyornis/pkg/validator"
	"github.com/fsouza/slognil"
	"github.com/labstack/echo/v5"
	"github.com/restayway/gogis"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vgarvardt/gue/v6"
)

const sampleRouteSegmentGPX = `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="Test" xmlns="http://www.topografix.com/GPX/1/1">
  <trk>
    <name>Sample Track</name>
    <trkseg>
      <trkpt lat="50.95786" lon="4.72410"><ele>25.0</ele><time>2026-01-01T10:00:00Z</time></trkpt>
      <trkpt lat="50.95816" lon="4.72391"><ele>26.0</ele><time>2026-01-01T10:01:00Z</time></trkpt>
      <trkpt lat="50.95900" lon="4.72500"><ele>27.0</ele><time>2026-01-01T10:02:00Z</time></trkpt>
    </trkseg>
  </trk>
</gpx>`

func setupRouteSegmentTestController(t *testing.T) (*routeSegmentController, *model.User) {
	t.Helper()
	db := createTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	gc, err := gue.NewClient(sqlDB)
	require.NoError(t, err)

	injector := do.New(repository.Package, service.Package)
	do.ProvideValue(injector, db)
	do.ProvideValue(injector, slognil.NewLogger())
	do.ProvideValue(injector, gc)
	ctrl := NewRouteSegmentController(injector).(*routeSegmentController)

	user := &model.User{
		UserData: model.UserData{Active: true},
		UserSecrets: model.UserSecrets{
			Email:    fmt.Sprintf("user_%d@example.com", time.Now().UnixNano()),
			Password: "pass",
		},
		Profile: model.Profile{Username: fmt.Sprintf("user_%d", time.Now().UnixNano()), DisplayName: "Test User"},
	}
	user.SetDB(db)
	require.NoError(t, user.Create(db))

	return ctrl, user
}

func TestRouteSegmentController_CreateWithCategory(t *testing.T) {
	ctrl, user := setupRouteSegmentTestController(t)

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	part, err := w.CreateFormFile("file", "sample.gpx")
	require.NoError(t, err)
	_, err = part.Write([]byte(sampleRouteSegmentGPX))
	require.NoError(t, err)
	require.NoError(t, w.WriteField("category", "cycling"))
	require.NoError(t, w.WriteField("notes", "Test notes"))
	require.NoError(t, w.WriteField("bidirectional", "true"))
	require.NoError(t, w.WriteField("circular", "true"))
	require.NoError(t, w.Close())

	e := echo.New()
	e.Validator = validator.New()
	req := httptest.NewRequest(http.MethodPost, "/route-segments", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_info", user)

	require.NoError(t, ctrl.CreateRouteSegment(c))
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp dto.Response[dto.RouteSegmentsDetailResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Results, 1)
	assert.Equal(t, "cycling", resp.Results[0].Category)
	assert.True(t, resp.Results[0].Bidirectional)
	assert.True(t, resp.Results[0].Circular)
}

func TestRouteSegmentController_CreateFromWorkout_PrefillCategory(t *testing.T) {
	ctrl, user := setupRouteSegmentTestController(t)

	workout := &model.Workout{
		ProfileID:  user.Profile.ID,
		Type:       model.WorkoutTypeRunning,
		Visibility: model.WorkoutVisibilityPublic,
		Date:       time.Now().UTC(),
		Records: []model.WorkoutRecord{
			{Point: &gogis.Point{Lat: 50.95786, Lng: 4.72410}, SortOrder: 0},
			{Point: &gogis.Point{Lat: 50.95816, Lng: 4.72391}, SortOrder: 1},
			{Point: &gogis.Point{Lat: 50.95900, Lng: 4.72500}, SortOrder: 2},
			{Point: &gogis.Point{Lat: 50.96000, Lng: 4.72600}, SortOrder: 3},
		},
	}
	require.NoError(t, workout.Save(ctrl.db))

	e := echo.New()
	e.Validator = validator.New()
	workoutIDStr := strconv.FormatUint(workout.ID, 10)

	// 1. With explicitly specified category
	{
		body := `{"name":"Seg 1","start":1,"end":3,"category":"cycling","bidirectional":true}`
		req := httptest.NewRequest(http.MethodPost, "/workouts/"+workoutIDStr+"/route-segment", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/workouts/:id/route-segment")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: workoutIDStr}})
		c.Set("user_info", user)

		require.NoError(t, ctrl.CreateRouteSegmentFromWorkout(c))
		assert.Equal(t, http.StatusCreated, rec.Code)

		var resp dto.Response[dto.RouteSegmentDetailResponse]
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, "cycling", resp.Results.Category)
		assert.True(t, resp.Results.Bidirectional)
	}

	// 2. Without category, falls back to workout.Type ("running")
	{
		body := `{"name":"Seg 2","start":2,"end":4}`
		req := httptest.NewRequest(http.MethodPost, "/workouts/"+workoutIDStr+"/route-segment", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/workouts/:id/route-segment")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: workoutIDStr}})
		c.Set("user_info", user)

		require.NoError(t, ctrl.CreateRouteSegmentFromWorkout(c))
		assert.Equal(t, http.StatusCreated, rec.Code)

		var resp dto.Response[dto.RouteSegmentDetailResponse]
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, "running", resp.Results.Category)
	}
}

func TestRouteSegmentController_Visibility_GetRouteSegment(t *testing.T) {
	ctrl, userOwner := setupRouteSegmentTestController(t)

	// Create userViewer
	userViewer := &model.User{
		UserData: model.UserData{Active: true},
		UserSecrets: model.UserSecrets{
			Email:    "viewer@example.com",
			Password: "pass",
		},
		Profile: model.Profile{Username: "viewer", DisplayName: "Viewer User"},
	}
	userViewer.SetDB(ctrl.db)
	require.NoError(t, userViewer.Create(ctrl.db))

	// Create userFollower
	userFollower := &model.User{
		UserData: model.UserData{Active: true},
		UserSecrets: model.UserSecrets{
			Email:    "follower@example.com",
			Password: "pass",
		},
		Profile: model.Profile{Username: "follower", DisplayName: "Follower User"},
	}
	userFollower.SetDB(ctrl.db)
	require.NoError(t, userFollower.Create(ctrl.db))

	// follower follows userOwner
	followerRecord := &model.Follower{
		ProfileID:          userFollower.Profile.ID,
		FollowingProfileID: userOwner.Profile.ID,
		Approved:           true,
	}
	require.NoError(t, ctrl.db.Create(followerRecord).Error)

	// 1. Private route segment
	rsPrivate, err := ctrl.routeSegmentRepo.CreateFromContent("Private note", "private.gpx", []byte(sampleRouteSegmentGPX))
	require.NoError(t, err)
	rsPrivate.ProfileID = userOwner.Profile.ID
	rsPrivate.Visibility = model.WorkoutVisibilityPrivate
	require.NoError(t, rsPrivate.Save(ctrl.db))

	// 2. Followers-only route segment
	gpxFollowers := bytes.ReplaceAll([]byte(sampleRouteSegmentGPX), []byte("50.95786"), []byte("50.95787"))
	rsFollowers, err := ctrl.routeSegmentRepo.CreateFromContent("Followers note", "followers.gpx", gpxFollowers)
	require.NoError(t, err)
	rsFollowers.ProfileID = userOwner.Profile.ID
	rsFollowers.Visibility = model.WorkoutVisibilityFollowers
	require.NoError(t, rsFollowers.Save(ctrl.db))

	// 3. Public route segment
	gpxPublic := bytes.ReplaceAll([]byte(sampleRouteSegmentGPX), []byte("50.95786"), []byte("50.95788"))
	rsPublic, err := ctrl.routeSegmentRepo.CreateFromContent("Public note", "public.gpx", gpxPublic)
	require.NoError(t, err)
	rsPublic.ProfileID = userOwner.Profile.ID
	rsPublic.Visibility = model.WorkoutVisibilityPublic
	require.NoError(t, rsPublic.Save(ctrl.db))

	e := echo.New()
	e.Validator = validator.New()

	testAccess := func(rsID uint64, viewer *model.User, expectedStatus int) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/route-segments/%d", rsID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/route-segments/:id")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: strconv.FormatUint(rsID, 10)}})
		if viewer != nil {
			c.Set("user_info", viewer)
		}

		err := ctrl.GetRouteSegment(c)
		if expectedStatus == http.StatusOK {
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)
		} else {
			assert.Equal(t, expectedStatus, rec.Code)
		}
	}

	// Owner can access all
	testAccess(rsPrivate.ID, userOwner, http.StatusOK)
	testAccess(rsFollowers.ID, userOwner, http.StatusOK)
	testAccess(rsPublic.ID, userOwner, http.StatusOK)

	// Follower can access public and followers, but not private
	testAccess(rsPrivate.ID, userFollower, http.StatusNotFound)
	testAccess(rsFollowers.ID, userFollower, http.StatusOK)
	testAccess(rsPublic.ID, userFollower, http.StatusOK)

	// Non-follower viewer can only access public
	testAccess(rsPrivate.ID, userViewer, http.StatusNotFound)
	testAccess(rsFollowers.ID, userViewer, http.StatusNotFound)
	testAccess(rsPublic.ID, userViewer, http.StatusOK)

	// Anonymous viewer can only access public
	testAccess(rsPrivate.ID, nil, http.StatusNotFound)
	testAccess(rsFollowers.ID, nil, http.StatusNotFound)
	testAccess(rsPublic.ID, nil, http.StatusOK)
}

func TestRouteSegmentController_Visibility_ListRouteSegments(t *testing.T) {
	ctrl, userOwner := setupRouteSegmentTestController(t)

	userViewer := &model.User{
		UserData: model.UserData{Active: true},
		UserSecrets: model.UserSecrets{
			Email:    "viewer_list@example.com",
			Password: "pass",
		},
		Profile: model.Profile{Username: "viewer_list", DisplayName: "Viewer List User"},
	}
	userViewer.SetDB(ctrl.db)
	require.NoError(t, userViewer.Create(ctrl.db))

	// 1. Private route segment
	rsPrivate, err := ctrl.routeSegmentRepo.CreateFromContent("Private note", "private_list.gpx", []byte(sampleRouteSegmentGPX))
	require.NoError(t, err)
	rsPrivate.ProfileID = userOwner.Profile.ID
	rsPrivate.Visibility = model.WorkoutVisibilityPrivate
	require.NoError(t, rsPrivate.Save(ctrl.db))

	// 2. Public route segment
	gpxPublic := bytes.ReplaceAll([]byte(sampleRouteSegmentGPX), []byte("50.95786"), []byte("50.95789"))
	rsPublic, err := ctrl.routeSegmentRepo.CreateFromContent("Public note", "public_list.gpx", gpxPublic)
	require.NoError(t, err)
	rsPublic.ProfileID = userOwner.Profile.ID
	rsPublic.Visibility = model.WorkoutVisibilityPublic
	require.NoError(t, rsPublic.Save(ctrl.db))

	e := echo.New()
	e.Validator = validator.New()

	// Owner list includes both
	{
		req := httptest.NewRequest(http.MethodGet, "/route-segments", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_info", userOwner)

		require.NoError(t, ctrl.GetRouteSegments(c))
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp dto.PaginatedResponse[dto.RouteSegmentResponse]
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		ids := make([]uint64, len(resp.Results))
		for i, r := range resp.Results {
			ids[i] = r.ID
		}
		assert.Contains(t, ids, rsPrivate.ID)
		assert.Contains(t, ids, rsPublic.ID)
	}

	// Other viewer list only includes public
	{
		req := httptest.NewRequest(http.MethodGet, "/route-segments", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_info", userViewer)

		require.NoError(t, ctrl.GetRouteSegments(c))
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp dto.PaginatedResponse[dto.RouteSegmentResponse]
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		ids := make([]uint64, len(resp.Results))
		for i, r := range resp.Results {
			ids[i] = r.ID
		}
		assert.NotContains(t, ids, rsPrivate.ID)
		assert.Contains(t, ids, rsPublic.ID)
	}
}

func TestRouteSegmentController_Visibility_MatchesAndStatsDoNotLeakPrivateWorkouts(t *testing.T) {
	ctrl, userOwner := setupRouteSegmentTestController(t)

	userAthlete := &model.User{
		UserData: model.UserData{Active: true},
		UserSecrets: model.UserSecrets{
			Email:    "athlete@example.com",
			Password: "pass",
		},
		Profile: model.Profile{Username: "athlete", DisplayName: "Athlete Private User"},
	}
	userAthlete.SetDB(ctrl.db)
	require.NoError(t, userAthlete.Create(ctrl.db))

	userViewer := &model.User{
		UserData: model.UserData{Active: true},
		UserSecrets: model.UserSecrets{
			Email:    "viewer_stats@example.com",
			Password: "pass",
		},
		Profile: model.Profile{Username: "viewer_stats", DisplayName: "Viewer Stats User"},
	}
	userViewer.SetDB(ctrl.db)
	require.NoError(t, userViewer.Create(ctrl.db))

	// Public route segment
	rsPublic, err := ctrl.routeSegmentRepo.CreateFromContent("Public note", "public_segment.gpx", []byte(sampleRouteSegmentGPX))
	require.NoError(t, err)
	rsPublic.ProfileID = userOwner.Profile.ID
	rsPublic.Visibility = model.WorkoutVisibilityPublic
	require.NoError(t, rsPublic.Save(ctrl.db))

	// Private workout by userAthlete
	workoutPrivate := &model.Workout{
		ProfileID:  userAthlete.Profile.ID,
		Type:       model.WorkoutTypeRunning,
		Visibility: model.WorkoutVisibilityPrivate,
		Name:       "Secret Morning Run",
		Date:       time.Now().UTC(),
		Records: []model.WorkoutRecord{
			{Point: &gogis.Point{Lat: 50.95786, Lng: 4.72410}, SortOrder: 0},
			{Point: &gogis.Point{Lat: 50.95816, Lng: 4.72391}, SortOrder: 1},
			{Point: &gogis.Point{Lat: 50.95900, Lng: 4.72500}, SortOrder: 2},
		},
	}
	require.NoError(t, workoutPrivate.Save(ctrl.db))

	// Create match for private workout
	match := &model.RouteSegmentMatch{
		WorkoutID:      workoutPrivate.ID,
		RouteSegmentID: rsPublic.ID,
		Duration:       60 * time.Second,
		Distance:       1000.0,
	}
	require.NoError(t, ctrl.db.Create(match).Error)

	e := echo.New()
	e.Validator = validator.New()

	// userAthlete (owner of private workout) can see the match
	{
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/route-segments/%d/matches", rsPublic.ID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/route-segments/:id/matches")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: strconv.FormatUint(rsPublic.ID, 10)}})
		c.Set("user_info", userAthlete)

		require.NoError(t, ctrl.GetRouteSegmentMatches(c))
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp dto.PaginatedResponse[dto.RouteSegmentMatch]
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Len(t, resp.Results, 1)
		assert.Equal(t, "Secret Morning Run", resp.Results[0].WorkoutName)
	}

	// userViewer CANNOT see the match of the private workout
	{
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/route-segments/%d/matches", rsPublic.ID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/route-segments/:id/matches")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: strconv.FormatUint(rsPublic.ID, 10)}})
		c.Set("user_info", userViewer)

		require.NoError(t, ctrl.GetRouteSegmentMatches(c))
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp dto.PaginatedResponse[dto.RouteSegmentMatch]
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Empty(t, resp.Results)
	}

	// userViewer GetRouteSegment stats do not leak private course record
	{
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/route-segments/%d", rsPublic.ID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/route-segments/:id")
		c.SetPathValues(echo.PathValues{{Name: "id", Value: strconv.FormatUint(rsPublic.ID, 10)}})
		c.Set("user_info", userViewer)

		require.NoError(t, ctrl.GetRouteSegment(c))
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp dto.Response[dto.RouteSegmentDetailResponse]
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Empty(t, resp.Results.Matches)
		if resp.Results.Stats != nil {
			assert.Equal(t, int64(0), resp.Results.Stats.TotalEfforts)
			assert.Nil(t, resp.Results.Stats.CourseRecord)
		}
	}
}

func TestRouteSegmentController_CreateWithCustomName(t *testing.T) {
	ctrl, user := setupRouteSegmentTestController(t)

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	part, err := w.CreateFormFile("file", "test_track.gpx")
	require.NoError(t, err)
	_, err = part.Write([]byte(sampleRouteSegmentGPX))
	require.NoError(t, err)

	require.NoError(t, w.WriteField("name", "My Custom Segment Name"))
	require.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPost, "/route-segments", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()

	e := echo.New()
	e.Validator = validator.New()
	c := e.NewContext(req, rec)
	c.Set("user_info", user)

	err = ctrl.CreateRouteSegment(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp dto.Response[dto.RouteSegmentsDetailResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Results, 1)
	assert.Equal(t, "My Custom Segment Name", resp.Results[0].Name)
}
