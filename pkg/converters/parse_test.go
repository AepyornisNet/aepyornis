package converters

import (
	"testing"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestSynthesizeTimerEventsFromRecords_EmitsPauseEvents(t *testing.T) {
	base := time.Date(2026, time.April, 16, 10, 0, 0, 0, time.UTC)

	records := []model.WorkoutRecord{
		{Time: base},
		{Time: base.Add(10 * time.Second), Duration: 10 * time.Second, Distance: 35},
		{Time: base.Add(20 * time.Second), Duration: 10 * time.Second, Distance: 0},
		{Time: base.Add(30 * time.Second), Duration: 10 * time.Second, Distance: 0},
		{Time: base.Add(40 * time.Second), Duration: 10 * time.Second, Distance: 40},
	}

	events := synthesizeTimerEventsFromRecords(records)

	if assert.Len(t, events, 2) {
		assert.Equal(t, "timer", events[0].Event)
		assert.Equal(t, "stop", events[0].EventType)
		assert.Equal(t, base.Add(10*time.Second), events[0].Timestamp)
		assert.Nil(t, events[0].Payload)

		assert.Equal(t, "timer", events[1].Event)
		assert.Equal(t, "start", events[1].EventType)
		assert.Equal(t, base.Add(40*time.Second), events[1].Timestamp)
		assert.Nil(t, events[1].Payload)
	}
}

func TestSynthesizeTimerEventsFromRecords_IgnoresShortPauseNoise(t *testing.T) {
	base := time.Date(2026, time.April, 16, 11, 0, 0, 0, time.UTC)

	records := []model.WorkoutRecord{
		{Time: base},
		{Time: base.Add(3 * time.Second), Duration: 3 * time.Second, Distance: 0},
		{Time: base.Add(10 * time.Second), Duration: 7 * time.Second, Distance: 30},
	}

	events := synthesizeTimerEventsFromRecords(records)
	assert.Nil(t, events)
}
