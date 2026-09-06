package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFastestDistancesForWorkout(t *testing.T) {
	t.Run("nil or empty workout returns nil", func(t *testing.T) {
		assert.Nil(t, fastestDistancesForWorkout(nil, nil))
		assert.Nil(t, fastestDistancesForWorkout(&Workout{}, nil))
	})

	t.Run("includes pause time in distance interval duration", func(t *testing.T) {
		now := time.Now()
		records := []WorkoutRecord{
			{Time: now, Distance: 100.0, Duration: 10 * time.Second},
			{Time: now.Add(10 * time.Second), Distance: 100.0, Duration: 10 * time.Second},
			{Time: now.Add(20 * time.Second), Distance: 100.0, Duration: 10 * time.Second},
			{Time: now.Add(30 * time.Second), Distance: 100.0, Duration: 10 * time.Second},
			{Time: now.Add(40 * time.Second), Distance: 0.0, Duration: 60 * time.Second}, // pause
			{Time: now.Add(100 * time.Second), Distance: 150.0, Duration: 10 * time.Second},
			{Time: now.Add(110 * time.Second), Distance: 150.0, Duration: 10 * time.Second},
			{Time: now.Add(120 * time.Second), Distance: 150.0, Duration: 10 * time.Second},
			{Time: now.Add(130 * time.Second), Distance: 150.0, Duration: 10 * time.Second},
		}

		w := &Workout{
			Model:   Model{ID: 1},
			Type:    WorkoutTypeRunning,
			Date:    now,
			Data:    &WorkoutGeoMeta{Center: MapCenter{}},
			Records: records,
		}

		targets := []DistanceRecordTarget{
			{Label: "1 km", TargetDistance: 1000.0},
		}

		results := fastestDistancesForWorkout(w, targets)
		require.Len(t, results, 1)

		assert.Equal(t, "1 km", results[0].Label)
		assert.Equal(t, 1000.0, results[0].TargetDistance)
		assert.Equal(t, 1000.0, results[0].Distance)
		// Duration must include the 60s pause time: 4*10 + 60 + 4*10 = 140s
		assert.Equal(t, 140*time.Second, results[0].Duration)
		assert.InDelta(t, 1000.0/140.0, results[0].AverageSpeed, 0.01)
	})
}
