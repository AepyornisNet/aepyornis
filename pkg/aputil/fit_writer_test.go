package aputil

import (
	"testing"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/converters"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateWorkoutFIT_PreservesPerRecordSpeedMetrics(t *testing.T) {
	workout := &model.Workout{
		Date:          time.Date(2026, 4, 26, 18, 0, 0, 0, time.UTC),
		TotalDuration: 10 * time.Second,
		Data:          &model.WorkoutGeoMeta{},
		Records: []model.WorkoutRecord{
			{
				Time:          time.Date(2026, 4, 26, 18, 0, 0, 0, time.UTC),
				Lat:           50.0,
				Lng:           8.0,
				Elevation:     100,
				TotalDuration: 0,
				ExtraMetrics:  model.ExtraMetrics{"speed": 0.0},
			},
			{
				Time:          time.Date(2026, 4, 26, 18, 0, 5, 0, time.UTC),
				Lat:           50.0,
				Lng:           8.0,
				Elevation:     100,
				Duration:      5 * time.Second,
				TotalDuration: 5 * time.Second,
				ExtraMetrics:  model.ExtraMetrics{"speed": 6.2},
			},
			{
				Time:          time.Date(2026, 4, 26, 18, 0, 10, 0, time.UTC),
				Lat:           50.0,
				Lng:           8.0,
				Elevation:     100,
				Duration:      5 * time.Second,
				TotalDuration: 10 * time.Second,
				ExtraMetrics:  model.ExtraMetrics{"speed": 6.4},
			},
		},
	}

	fitContent, err := GenerateWorkoutFIT(workout)
	require.NoError(t, err)

	parsed, err := converters.ParseFit(fitContent, "workout.fit")
	require.NoError(t, err)
	require.Len(t, parsed, 1)
	require.Len(t, parsed[0].Records, 3)

	assert.InDelta(t, 6.2, parsed[0].Records[1].ExtraMetrics["speed"], 0.01)
	assert.InDelta(t, 6.4, parsed[0].Records[2].ExtraMetrics["speed"], 0.01)
	assert.Equal(t, 0.0, parsed[0].Records[1].AverageSpeed())
	assert.Equal(t, 0.0, parsed[0].Records[2].AverageSpeed())
}
