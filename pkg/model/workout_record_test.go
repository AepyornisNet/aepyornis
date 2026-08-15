package model

import (
	"math"
	"testing"

	"github.com/restayway/gogis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkoutRecord_Point(t *testing.T) {
	t.Run("Getters and Orb Point conversion with valid point", func(t *testing.T) {
		wr := &WorkoutRecord{
			Point: &gogis.Point{Lat: 50.123, Lng: 8.456},
		}
		assert.Equal(t, 50.123, wr.Lat())
		assert.Equal(t, 8.456, wr.Lng())

		orbPt := wr.ToOrbPoint()
		require.NotNil(t, orbPt)
		assert.Equal(t, 8.456, orbPt[0])
		assert.Equal(t, 50.123, orbPt[1])
	})

	t.Run("Getters and Orb Point conversion with nil point", func(t *testing.T) {
		wr := &WorkoutRecord{
			Point: nil,
		}
		assert.Equal(t, 0.0, wr.Lat())
		assert.Equal(t, 0.0, wr.Lng())
		assert.Nil(t, wr.ToOrbPoint())
	})

	t.Run("Point Value returns WKT", func(t *testing.T) {
		p := gogis.Point{Lat: 50.123, Lng: 8.456}
		val, err := p.Value()
		require.NoError(t, err)
		assert.Equal(t, "SRID=4326;POINT(8.456 50.123)", val)
	})
	t.Run("PointDistance calculates distance correctly using orb", func(t *testing.T) {
		p1 := gogis.Point{Lat: 50.0, Lng: 8.0}
		p2 := gogis.Point{Lat: 50.001, Lng: 8.0}
		d := PointDistance(p1, p2)
		assert.InDelta(t, 111.19, d, 0.5)
	})

	t.Run("DistanceTo and DistanceToPoint with valid points", func(t *testing.T) {
		wr1 := &WorkoutRecord{Point: &gogis.Point{Lat: 50.0, Lng: 8.0}}
		wr2 := &WorkoutRecord{Point: &gogis.Point{Lat: 50.001, Lng: 8.0}}
		assert.InDelta(t, 111.19, wr1.DistanceTo(wr2), 0.5)
		assert.InDelta(t, 111.19, wr1.DistanceToPoint(*wr2.Point), 0.5)
	})

	t.Run("DistanceTo and DistanceToPoint with nil points", func(t *testing.T) {
		wr1 := &WorkoutRecord{Point: nil}
		wr2 := &WorkoutRecord{Point: &gogis.Point{Lat: 50.0, Lng: 8.0}}
		assert.True(t, math.IsInf(wr1.DistanceTo(wr2), 1))
		assert.True(t, math.IsInf(wr2.DistanceTo(wr1), 1))
		assert.True(t, math.IsInf(wr1.DistanceToPoint(*wr2.Point), 1))
	})
}
