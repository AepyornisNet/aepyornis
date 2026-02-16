package activitypub

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
)

const RouteImageMIMEType = "image/png"

const (
	routeImageWidth        = 1200
	routeImageHeight       = 630
	routeImageMaxPoints    = 120
	routeImagePadding      = 0.05
	routeTileSize          = 256
	routeMinZoom           = 1
	routeMaxZoom           = 17
	routeRouteStrokeWidth  = 5
	routeRouteStrokeRadius = routeRouteStrokeWidth / 2
	routeTileURLPattern    = "https://tile.openstreetmap.org/%d/%d/%d.png"
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

	client := &http.Client{Timeout: 8 * time.Second}
	zoom := zoomForBounds(bbox, routeImageWidth, routeImageHeight)

	canvas, originX, originY, err := drawOSMBaseMap(client, bbox, zoom, routeImageWidth, routeImageHeight)
	if err != nil {
		return nil, err
	}

	renderRouteOnCanvas(canvas, points, zoom, originX, originY)

	buf := bytes.NewBuffer(nil)
	if err := png.Encode(buf, canvas); err != nil {
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

func zoomForBounds(bbox routeBBox, width, height int) int {
	for zoom := routeMaxZoom; zoom >= routeMinZoom; zoom-- {
		minX, minY := projectLatLngToWorldPixels(bbox.maxLat, bbox.minLng, zoom)
		maxX, maxY := projectLatLngToWorldPixels(bbox.minLat, bbox.maxLng, zoom)

		spanX := math.Abs(maxX - minX)
		spanY := math.Abs(maxY - minY)

		if spanX <= float64(width)*0.85 && spanY <= float64(height)*0.85 {
			return zoom
		}
	}

	return routeMinZoom
}

func drawOSMBaseMap(client *http.Client, bbox routeBBox, zoom, width, height int) (*image.RGBA, float64, float64, error) {
	minX, minY := projectLatLngToWorldPixels(bbox.maxLat, bbox.minLng, zoom)
	maxX, maxY := projectLatLngToWorldPixels(bbox.minLat, bbox.maxLng, zoom)

	centerX := (minX + maxX) / 2
	centerY := (minY + maxY) / 2

	originX := centerX - float64(width)/2
	originY := centerY - float64(height)/2

	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.RGBA{R: 245, G: 245, B: 245, A: 255}), image.Point{}, draw.Src)

	minTileX := int(math.Floor(originX / routeTileSize))
	minTileY := int(math.Floor(originY / routeTileSize))
	maxTileX := int(math.Floor((originX + float64(width-1)) / routeTileSize))
	maxTileY := int(math.Floor((originY + float64(height-1)) / routeTileSize))

	tilesPerAxis := 1 << zoom
	loadedTiles := 0

	for tileY := minTileY; tileY <= maxTileY; tileY++ {
		if tileY < 0 || tileY >= tilesPerAxis {
			continue
		}

		for tileX := minTileX; tileX <= maxTileX; tileX++ {
			normalizedX := ((tileX % tilesPerAxis) + tilesPerAxis) % tilesPerAxis

			tile, err := fetchOSMTile(client, zoom, normalizedX, tileY)
			if err != nil {
				continue
			}

			dst := image.Rect(
				int(float64(tileX*routeTileSize)-originX),
				int(float64(tileY*routeTileSize)-originY),
				int(float64((tileX+1)*routeTileSize)-originX),
				int(float64((tileY+1)*routeTileSize)-originY),
			)

			draw.Draw(canvas, dst, tile, image.Point{}, draw.Over)
			loadedTiles++
		}
	}

	if loadedTiles == 0 {
		return nil, 0, 0, errors.New("could not load map tiles")
	}

	return canvas, originX, originY, nil
}

func fetchOSMTile(client *http.Client, zoom, tileX, tileY int) (image.Image, error) {
	url := fmt.Sprintf(routeTileURLPattern, zoom, tileX, tileY)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "workout-tracker/route-image")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tile request failed: %d", resp.StatusCode)
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	return img, nil
}

func renderRouteOnCanvas(canvas *image.RGBA, points []routePoint, zoom int, originX, originY float64) {
	if canvas == nil || len(points) < 2 {
		return
	}

	stroke := color.RGBA{R: 0, G: 85, B: 255, A: 255}

	prevX, prevY := routePointToCanvas(points[0], zoom, originX, originY)
	for i := 1; i < len(points); i++ {
		x, y := routePointToCanvas(points[i], zoom, originX, originY)
		drawThickLine(canvas, prevX, prevY, x, y, stroke, routeRouteStrokeWidth)
		prevX, prevY = x, y
	}
}

func routePointToCanvas(p routePoint, zoom int, originX, originY float64) (float64, float64) {
	x, y := projectLatLngToWorldPixels(p.Lat, p.Lng, zoom)
	return x - originX, y - originY
}

func projectLatLngToWorldPixels(lat, lng float64, zoom int) (float64, float64) {
	lat = min(max(lat, -85.05112878), 85.05112878)
	lng = math.Mod(lng+180.0, 360.0)
	if lng < 0 {
		lng += 360
	}
	lng -= 180

	scale := float64(routeTileSize) * math.Pow(2, float64(zoom))
	x := (lng + 180.0) / 360.0 * scale
	latRad := lat * math.Pi / 180.0
	y := (1 - math.Log(math.Tan(latRad)+1/math.Cos(latRad))/math.Pi) / 2 * scale

	return x, y
}

func drawThickLine(img *image.RGBA, x1, y1, x2, y2 float64, col color.RGBA, width int) {
	steps := int(math.Max(math.Abs(x2-x1), math.Abs(y2-y1)))
	if steps < 1 {
		drawFilledCircle(img, int(math.Round(x1)), int(math.Round(y1)), routeRouteStrokeRadius, col)
		return
	}

	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(math.Round(x1 + (x2-x1)*t))
		y := int(math.Round(y1 + (y2-y1)*t))
		drawFilledCircle(img, x, y, max(1, width/2), col)
	}
}

func drawFilledCircle(img *image.RGBA, cx, cy, radius int, col color.RGBA) {
	if radius < 1 {
		radius = 1
	}

	bounds := img.Bounds()
	for y := cy - radius; y <= cy+radius; y++ {
		if y < bounds.Min.Y || y >= bounds.Max.Y {
			continue
		}

		for x := cx - radius; x <= cx+radius; x++ {
			if x < bounds.Min.X || x >= bounds.Max.X {
				continue
			}

			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= radius*radius {
				img.SetRGBA(x, y, col)
			}
		}
	}
}
