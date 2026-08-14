package model

import (
	"testing"
	"time"

	"github.com/restayway/gogis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkoutRecord_Point(t *testing.T) {
	t.Run("Getters and Orb Point conversion", func(t *testing.T) {
		wr := &WorkoutRecord{
			Point: gogis.Point{Lat: 50.123, Lng: 8.456},
		}
		assert.Equal(t, 50.123, wr.Lat())
		assert.Equal(t, 8.456, wr.Lng())

		orbPt := wr.ToOrbPoint()
		assert.Equal(t, 8.456, orbPt[0])
		assert.Equal(t, 50.123, orbPt[1])
	})

	t.Run("Point Value returns WKT", func(t *testing.T) {
		p := gogis.Point{Lat: 50.123, Lng: 8.456}
		val, err := p.Value()
		require.NoError(t, err)
		assert.Equal(t, "SRID=4326;POINT(8.456 50.123)", val)
	})
}

func TestWorkoutRecord_GPXPoint(t *testing.T) {
	wr := WorkoutRecord{
		Point:     gogis.Point{Lat: 52.52, Lng: 13.405},
		Elevation: 35.5,
		Time:      time.Now(),
	}

	gpxPt := wr.AsGPXPoint()
	require.NotNil(t, gpxPt)
	assert.Equal(t, 52.52, gpxPt.Latitude)
	assert.Equal(t, 13.405, gpxPt.Longitude)
	assert.Equal(t, 35.5, gpxPt.Elevation.Value())
}
