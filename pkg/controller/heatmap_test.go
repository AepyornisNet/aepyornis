package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/model/dto"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/AepyornisNet/aepyornis/pkg/service"
	"github.com/fsouza/slognil"
	"github.com/labstack/echo/v5"
	"github.com/restayway/gogis"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeatmapController_GetWorkoutCoordinates(t *testing.T) {
	db := createTestDB(t)

	injector := do.New(repository.Package, service.Package)
	do.ProvideValue(injector, db)
	do.ProvideValue(injector, slognil.NewLogger())
	ctrl := NewHeatmapController(injector)

	user := &model.User{
		UserData: model.UserData{
			Active: true,
		},
		UserSecrets: model.UserSecrets{
			Email:    "heatmap@example.com",
			Password: "pass",
		},
		Profile: model.Profile{Username: "heatmapuser", DisplayName: "Heatmap User"},
	}
	user.SetDB(db)
	require.NoError(t, user.Create(db))

	workout := &model.Workout{
		ProfileID: user.Profile.ID,
		Type:      model.WorkoutTypeRunning,
		Date:      time.Now().UTC(),
		Records: []model.WorkoutRecord{
			{Point: &gogis.Point{Lat: 50.95786, Lng: 4.72410}, SortOrder: 0},
			{Point: &gogis.Point{Lat: 50.95816, Lng: 4.72391}, SortOrder: 1},
			{Point: &gogis.Point{Lat: 50.95900, Lng: 4.72500}, SortOrder: 2},
			{Point: nil, SortOrder: 3},
		},
	}
	require.NoError(t, workout.Save(db))

	e := echo.New()

	// 1. Without cell_size (raw coordinates)
	{
		req := httptest.NewRequest(http.MethodGet, "/workouts/coordinates", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_info", user)

		require.NoError(t, ctrl.GetWorkoutCoordinates(c))
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp dto.Response[[][]float64]
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Len(t, resp.Results, 3)
		assert.InDelta(t, 50.95786, resp.Results[0][0], 0.0001)
		assert.InDelta(t, 4.72410, resp.Results[0][1], 0.0001)
		assert.Equal(t, float64(1), resp.Results[0][2])
	}

	// 2. With cell_size (ST_SnapToGrid aggregation)
	{
		req := httptest.NewRequest(http.MethodGet, "/workouts/coordinates?cell_size=0.01", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_info", user)

		require.NoError(t, ctrl.GetWorkoutCoordinates(c))
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp dto.Response[[][]float64]
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.NotEmpty(t, resp.Results)
		assert.Equal(t, float64(3), resp.Results[0][2]) // all 3 points in same 0.01 grid
	}

	// 3. With spatial viewport bounding box (ST_Intersects with ST_MakeEnvelope)
	{
		req := httptest.NewRequest(http.MethodGet, "/workouts/coordinates?min_lat=50.9570&max_lat=50.9580&min_lng=4.7230&max_lng=4.7250", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_info", user)

		require.NoError(t, ctrl.GetWorkoutCoordinates(c))
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp dto.Response[[][]float64]
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Len(t, resp.Results, 1) // only 1 point falls inside this bounding box
		assert.InDelta(t, 50.95786, resp.Results[0][0], 0.0001)
	}
}
