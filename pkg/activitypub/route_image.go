package activitypub

import (
	"bytes"
	"errors"
	"fmt"
	"image/color"
	"image/png"
	"math"

	sm "github.com/flopp/go-staticmaps"
	"github.com/golang/geo/s2"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
)

const RouteImageMIMEType = "image/png"

const (
	routeImageWidth       = 1200
	routeImageHeight      = 630
	routeImageMaxPoints   = 120
	routeImagePadding     = 0.05
	routeRouteStrokeWidth = 5.0
)

var ErrWorkoutMissingCoordinates = errors.New("workout has no usable coordinates")

type routePoint struct {
	Lat float64
	Lng float64
}

func GenerateWorkoutRouteImage(workout *model.Workout) ([]byte, error) {
	points := routePointsFromWorkout(workout)
	if len(points) < 2 {
		return nil, ErrWorkoutMissingCoordinates
	}

	points = downsampleRoutePoints(points, routeImageMaxPoints)

	bbox := routeBoundingBox(points)
	bbox = padRouteBoundingBox(bbox, routeImagePadding)

	ctx := sm.NewContext()
	ctx.SetSize(routeImageWidth, routeImageHeight)
	ctx.SetUserAgent("workout-tracker/route-image")
	ctx.SetTileProvider(sm.NewTileProviderOpenStreetMaps())

	box, err := sm.CreateBBox(bbox.maxLat, bbox.minLng, bbox.minLat, bbox.maxLng)
	if err != nil {
		return nil, err
	}
	ctx.SetBoundingBox(*box)

	positions := make([]s2.LatLng, 0, len(points))
	for _, p := range points {
		positions = append(positions, s2.LatLngFromDegrees(p.Lat, p.Lng))
	}

	ctx.AddPath(sm.NewPath(positions, color.RGBA{R: 0, G: 85, B: 255, A: 255}, routeRouteStrokeWidth))

	img, err := ctx.Render()
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(nil)
	if err := png.Encode(buf, img); err != nil {
		return nil, err
	}

	if buf.Len() == 0 {
		return nil, errors.New("generated route image is empty")
	}

	return buf.Bytes(), nil
}

func WorkoutRouteImageFilename(workout *model.Workout) string {
	if workout == nil {
		return "workout-route.png"
	}

	return fmt.Sprintf("workout-%d-route.png", workout.ID)
}

func routePointsFromWorkout(workout *model.Workout) []routePoint {
	if workout == nil || workout.Data == nil || workout.Data.Details == nil {
		return nil
	}

	points := make([]routePoint, 0, len(workout.Data.Details.Points))
	for _, p := range workout.Data.Details.Points {
		if math.IsNaN(p.Lat) || math.IsNaN(p.Lng) || (p.Lat == 0 && p.Lng == 0) {
			continue
		}

		if p.Lat < -90 || p.Lat > 90 || p.Lng < -180 || p.Lng > 180 {
			continue
		}

		if len(points) > 0 {
			last := points[len(points)-1]
			if last.Lat == p.Lat && last.Lng == p.Lng {
				continue
			}
		}

		points = append(points, routePoint{Lat: p.Lat, Lng: p.Lng})
	}

	return points
}

type routeBBox struct {
	minLat float64
	minLng float64
	maxLat float64
	maxLng float64
}

func routeBoundingBox(points []routePoint) routeBBox {
	bbox := routeBBox{
		minLat: points[0].Lat,
		minLng: points[0].Lng,
		maxLat: points[0].Lat,
		maxLng: points[0].Lng,
	}

	for _, p := range points[1:] {
		bbox.minLat = min(bbox.minLat, p.Lat)
		bbox.maxLat = max(bbox.maxLat, p.Lat)
		bbox.minLng = min(bbox.minLng, p.Lng)
		bbox.maxLng = max(bbox.maxLng, p.Lng)
	}

	return bbox
}

func padRouteBoundingBox(bbox routeBBox, factor float64) routeBBox {
	latSpan := bbox.maxLat - bbox.minLat
	lngSpan := bbox.maxLng - bbox.minLng

	if latSpan == 0 {
		latSpan = 0.01
	}

	if lngSpan == 0 {
		lngSpan = 0.01
	}

	latPad := latSpan * factor
	lngPad := lngSpan * factor

	bbox.minLat = max(-90.0, bbox.minLat-latPad)
	bbox.maxLat = min(90.0, bbox.maxLat+latPad)
	bbox.minLng = max(-180.0, bbox.minLng-lngPad)
	bbox.maxLng = min(180.0, bbox.maxLng+lngPad)

	return bbox
}

func downsampleRoutePoints(points []routePoint, maxPoints int) []routePoint {
	if len(points) <= maxPoints || maxPoints < 2 {
		return points
	}

	result := make([]routePoint, 0, maxPoints)
	result = append(result, points[0])

	step := float64(len(points)-1) / float64(maxPoints-1)
	for i := 1; i < maxPoints-1; i++ {
		idx := int(math.Round(float64(i) * step))
		idx = min(idx, len(points)-2)
		result = append(result, points[idx])
	}

	result = append(result, points[len(points)-1])
	return result
}
