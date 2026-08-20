package converters

import (
	"math"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/muktihari/fit/kit/semicircles"
	"github.com/muktihari/fit/profile/filedef"
	"github.com/muktihari/fit/profile/mesgdef"
	"github.com/restayway/gogis"
)

// mapDataFromActivity converts a FIT activity into MapData, falling back to
// non-positional record data when coordinates are missing so charts and
// breakdowns remain available even without a map.
func mapDataFromActivity(act *filedef.Activity) (*model.WorkoutGeoMeta, []model.WorkoutRecord) {
	if act == nil || len(act.Records) == 0 {
		return nil, nil
	}

	points := make([]model.WorkoutRecord, 0, len(act.Records))

	var (
		totalDuration time.Duration
		prevDistance  float64
	)

	for i, r := range act.Records {
		if r == nil {
			continue
		}

		ts := r.Timestamp.Local()
		if ts.IsZero() {
			continue
		}

		dist := 0.0
		if r.Distance != math.MaxUint32 {
			dist = r.DistanceScaled()
		}

		deltaDist := 0.0
		if i == 0 {
			prevDistance = dist
		} else if dist >= prevDistance {
			deltaDist = dist - prevDistance
			prevDistance = dist
		}

		dt := time.Duration(0)
		if i+1 < len(act.Records) && act.Records[i+1] != nil {
			dt = max(act.Records[i+1].Timestamp.Sub(ts), 0)
		}

		totalDuration += dt

		elevation := extractRecordElevation(r)
		extra := extractRecordExtraMetrics(r, elevation)

		elevationValue := elevation
		if math.IsNaN(elevationValue) {
			elevationValue = 0
		}

		points = append(points, model.WorkoutRecord{
			Time:          ts,
			Point:         extractRecordPoint(r),
			Elevation:     elevationValue,
			Distance:      deltaDist,
			TotalDistance: dist,
			Duration:      dt,
			TotalDuration: totalDuration,
			ExtraMetrics:  extra,
		})
	}

	if len(points) == 0 {
		return nil, nil
	}

	data := &model.WorkoutGeoMeta{Center: model.MapCenter{}}
	data.UpdateExtraMetrics(points)

	return data, points
}

func extractRecordElevation(r *mesgdef.Record) float64 {
	if r.EnhancedAltitude != math.MaxUint32 {
		return r.EnhancedAltitudeScaled()
	} else if r.Altitude != math.MaxUint16 {
		return r.AltitudeScaled()
	}
	return math.NaN()
}

func extractRecordPoint(r *mesgdef.Record) *gogis.Point {
	lat := semicircles.ToDegrees(r.PositionLat)
	lng := semicircles.ToDegrees(r.PositionLong)
	if !math.IsNaN(lat) && !math.IsNaN(lng) && (lat != 0 || lng != 0) {
		return &gogis.Point{Lat: lat, Lng: lng}
	}
	return nil
}

func extractRecordExtraMetrics(r *mesgdef.Record, elevation float64) model.ExtraMetrics {
	extra := model.ExtraMetrics{}
	if !math.IsNaN(elevation) {
		extra.Set("elevation", elevation)
	}
	if r.Cadence != math.MaxUint8 {
		extra.Set("cadence", float64(r.Cadence))
	}
	if r.HeartRate != math.MaxUint8 {
		extra.Set("heart-rate", float64(r.HeartRate))
	}
	if r.EnhancedRespirationRate != math.MaxUint16 {
		extra.Set("respiration-rate", float64(r.EnhancedRespirationRateScaled()))
	} else if r.RespirationRate != math.MaxUint8 {
		extra.Set("respiration-rate", float64(r.RespirationRate))
	}
	if r.Power != math.MaxUint16 {
		extra.Set("power", float64(r.Power))
	}
	if r.Temperature != math.MaxInt8 {
		extra.Set("temperature", float64(r.Temperature))
	}
	if r.EnhancedSpeed != math.MaxUint32 {
		extra.Set("speed", r.EnhancedSpeedScaled())
	} else if r.Speed != math.MaxUint16 {
		extra.Set("speed", r.SpeedScaled())
	}
	return extra
}
