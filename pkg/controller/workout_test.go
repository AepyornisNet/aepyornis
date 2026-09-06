package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/AepyornisNet/aepyornis/pkg/converters"
	"github.com/AepyornisNet/aepyornis/pkg/config"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/model/dto"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/AepyornisNet/aepyornis/pkg/service"
	"github.com/AepyornisNet/aepyornis/pkg/validator"
	"github.com/fsouza/slognil"
	"github.com/labstack/echo/v5"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vgarvardt/gue/v6"
)

const sampleWorkoutGPX = `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="Test" xmlns="http://www.topografix.com/GPX/1/1">
  <trk>
    <name>Sample Workout Track</name>
    <trkseg>
      <trkpt lat="50.95786" lon="4.72410"><ele>25.0</ele><time>2026-01-01T10:00:00Z</time></trkpt>
      <trkpt lat="50.95816" lon="4.72391"><ele>26.0</ele><time>2026-01-01T10:01:00Z</time></trkpt>
    </trkseg>
  </trk>
</gpx>`

func setupWorkoutTestController(t *testing.T) (*workoutController, *model.User) {
	t.Helper()
	db := createTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	gc, err := gue.NewClient(sqlDB)
	require.NoError(t, err)

	cfg := &config.Config{}

	injector := do.New(repository.Package, service.Package)
	do.ProvideValue(injector, db)
	do.ProvideValue(injector, slognil.NewLogger())
	do.ProvideValue(injector, gc)
	do.ProvideValue(injector, cfg)
	ctrl := NewWorkoutController(injector).(*workoutController)

	user := &model.User{
		UserData: model.UserData{
			Active:                   true,
			DefaultWorkoutVisibility: model.WorkoutVisibilityFollowers,
		},
		UserSecrets: model.UserSecrets{
			Email:    fmt.Sprintf("user_%d@example.com", time.Now().UnixNano()),
			Password: "pass",
		},
		Profile: model.Profile{
			Username:    fmt.Sprintf("user_%d", time.Now().UnixNano()),
			DisplayName: "Test User",
		},
	}
	user.SetDB(db)
	require.NoError(t, user.Create(db))

	return ctrl, user
}

func TestWorkoutController_CreateFromFile_Visibility(t *testing.T) {
	ctrl, user := setupWorkoutTestController(t)

	testCases := []struct {
		name               string
		visibilityField    string
		hasVisibilityField bool
		expectedVisibility model.WorkoutVisibility
		timeOffset         time.Duration
	}{
		{
			name:               "Explicit public visibility",
			visibilityField:    "public",
			hasVisibilityField: true,
			expectedVisibility: model.WorkoutVisibilityPublic,
			timeOffset:         1 * time.Hour,
		},
		{
			name:               "Explicit followers visibility",
			visibilityField:    "followers",
			hasVisibilityField: true,
			expectedVisibility: model.WorkoutVisibilityFollowers,
			timeOffset:         2 * time.Hour,
		},
		{
			name:               "Explicit private visibility",
			visibilityField:    "",
			hasVisibilityField: true,
			expectedVisibility: model.WorkoutVisibilityPrivate,
			timeOffset:         3 * time.Hour,
		},
		{
			name:               "No visibility field falls back to default",
			visibilityField:    "",
			hasVisibilityField: false,
			expectedVisibility: model.WorkoutVisibilityFollowers,
			timeOffset:         4 * time.Hour,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			startTime := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC).Add(tc.timeOffset)
			timeStr := startTime.Format(time.RFC3339)
			gpxContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="Test" xmlns="http://www.topografix.com/GPX/1/1">
  <trk>
    <name>%s</name>
    <trkseg>
      <trkpt lat="50.95786" lon="4.72410"><ele>25.0</ele><time>%s</time></trkpt>
      <trkpt lat="50.95816" lon="4.72391"><ele>26.0</ele><time>%s</time></trkpt>
    </trkseg>
  </trk>
</gpx>`, tc.name, timeStr, startTime.Add(time.Minute).Format(time.RFC3339))

			var b bytes.Buffer
			w := multipart.NewWriter(&b)
			part, err := w.CreateFormFile("file", "workout.gpx")
			require.NoError(t, err)
			_, err = part.Write([]byte(gpxContent))
			require.NoError(t, err)

			if tc.hasVisibilityField {
				require.NoError(t, w.WriteField("visibility", tc.visibilityField))
			}
			require.NoError(t, w.Close())

			e := echo.New()
			e.Validator = validator.New()
			req := httptest.NewRequest(http.MethodPost, "/workouts", &b)
			req.Header.Set("Content-Type", w.FormDataContentType())
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_info", user)

			require.NoError(t, ctrl.CreateWorkout(c))
			assert.Equal(t, http.StatusCreated, rec.Code)

			var resp dto.Response[[]dto.WorkoutResponse]
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
			require.Len(t, resp.Results, 1)
			assert.Equal(t, tc.expectedVisibility, resp.Results[0].Visibility)

			// Also verify persisted in database
			var loaded model.Workout
			require.NoError(t, ctrl.db.First(&loaded, resp.Results[0].ID).Error)
			assert.Equal(t, tc.expectedVisibility, loaded.Visibility)
		})
	}
}
