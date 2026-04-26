package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/AepyornisNet/aepyornis/pkg/model/dto"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/labstack/echo/v4"
	geojson "github.com/paulmach/orb/geojson"
	"github.com/samber/do/v2"
	"gorm.io/gorm"
)

const (
	defaultHeatmapCellSize = 0.0015
	minHeatmapCellSize     = 0.0001
	maxHeatmapCellSize     = 0.1
)

type HeatmapController interface {
	GetWorkoutCoordinates(c echo.Context) error
	GetWorkoutCenters(c echo.Context) error
}

type heatmapController struct {
	db          *gorm.DB
	workoutRepo repository.Workout
}

type heatmapBounds struct {
	minLat float64
	minLng float64
	maxLat float64
	maxLng float64
}

type aggregatedCoordinateRow struct {
	LatCell float64 `gorm:"column:lat_cell"`
	LngCell float64 `gorm:"column:lng_cell"`
	Weight  int64   `gorm:"column:weight"`
}

type rawCoordinateRow struct {
	Lat float64 `gorm:"column:lat"`
	Lng float64 `gorm:"column:lng"`
}

func NewHeatmapController(injector do.Injector) HeatmapController {
	return &heatmapController{
		db:          do.MustInvoke[*gorm.DB](injector),
		workoutRepo: do.MustInvoke[repository.Workout](injector),
	}
}

// GetWorkoutCoordinates returns all coordinates of all workouts of the current user
// @Summary      Get workout coordinates
// @Tags         heatmap
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Param        cell_size  query  number  false  "Grid cell size in degrees used for server-side aggregation"  default(0.0015)
// @Param        min_lat    query  number  false  "Minimum latitude for viewport filtering"
// @Param        min_lng    query  number  false  "Minimum longitude for viewport filtering"
// @Param        max_lat    query  number  false  "Maximum latitude for viewport filtering"
// @Param        max_lng    query  number  false  "Maximum longitude for viewport filtering"
// @Success      200  {object}  dto.Response[[][]float64]
// @Failure      400  {object}  dto.Response[any]
// @Failure      500  {object}  dto.Response[any]
// @Router       /workouts/coordinates [get]
func (hc *heatmapController) GetWorkoutCoordinates(c echo.Context) error {
	hasCellSize := false
	cellSize := defaultHeatmapCellSize
	if rawCellSize := c.QueryParam("cell_size"); rawCellSize != "" {
		parsedCellSize, err := strconv.ParseFloat(rawCellSize, 64)
		if err != nil || parsedCellSize < minHeatmapCellSize || parsedCellSize > maxHeatmapCellSize {
			return renderApiError(c, http.StatusBadRequest, errors.New("invalid cell_size"))
		}
		cellSize = parsedCellSize
		hasCellSize = true
	}

	bounds, err := parseHeatmapBounds(c)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	u := currentUser(c)

	query := hc.db.Table("workout_records AS wr").
		Joins("JOIN workouts ON workouts.id = wr.workout_id").
		Where("workouts.profile_id = ?", u.Profile.ID)

	if bounds != nil {
		query = query.Where(
			"wr.lat >= ? AND wr.lat <= ? AND wr.lng >= ? AND wr.lng <= ?",
			bounds.minLat, bounds.maxLat, bounds.minLng, bounds.maxLng,
		)
	}

	if !hasCellSize {
		rows := make([]rawCoordinateRow, 0)
		if err := query.Select("wr.lat AS lat, wr.lng AS lng").Find(&rows).Error; err != nil {
			return renderApiError(c, http.StatusInternalServerError, err)
		}

		coords := make([][]float64, 0, len(rows))
		for _, row := range rows {
			coords = append(coords, []float64{row.Lat, row.Lng, 1})
		}

		resp := dto.Response[[][]float64]{
			Results: coords,
		}
		return c.JSON(http.StatusOK, resp)
	}

	rows := make([]aggregatedCoordinateRow, 0)
	if err := query.
		Select("floor(wr.lat / ?) AS lat_cell, floor(wr.lng / ?) AS lng_cell, count(*) AS weight", cellSize, cellSize).
		Group("lat_cell, lng_cell").
		Find(&rows).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	coords := make([][]float64, 0, len(rows))
	for _, row := range rows {
		lat := (row.LatCell + 0.5) * cellSize
		lng := (row.LngCell + 0.5) * cellSize
		coords = append(coords, []float64{lat, lng, float64(row.Weight)})
	}

	resp := dto.Response[[][]float64]{
		Results: coords,
	}

	return c.JSON(http.StatusOK, resp)
}

func parseHeatmapBounds(c echo.Context) (*heatmapBounds, error) {
	minLatRaw := c.QueryParam("min_lat")
	minLngRaw := c.QueryParam("min_lng")
	maxLatRaw := c.QueryParam("max_lat")
	maxLngRaw := c.QueryParam("max_lng")

	if minLatRaw == "" && minLngRaw == "" && maxLatRaw == "" && maxLngRaw == "" {
		return nil, nil
	}

	minLat, err := strconv.ParseFloat(minLatRaw, 64)
	if err != nil {
		return nil, errors.New("invalid min_lat")
	}
	minLng, err := strconv.ParseFloat(minLngRaw, 64)
	if err != nil {
		return nil, errors.New("invalid min_lng")
	}
	maxLat, err := strconv.ParseFloat(maxLatRaw, 64)
	if err != nil {
		return nil, errors.New("invalid max_lat")
	}
	maxLng, err := strconv.ParseFloat(maxLngRaw, 64)
	if err != nil {
		return nil, errors.New("invalid max_lng")
	}

	withingBounds := minLat < -90 || maxLat > 90 || minLng < -180 || maxLng > 180
	if withingBounds || minLat > maxLat || minLng > maxLng {
		return nil, errors.New("invalid viewport bounds")
	}

	return &heatmapBounds{
		minLat: minLat,
		minLng: minLng,
		maxLat: maxLat,
		maxLng: maxLng,
	}, nil
}

// GetWorkoutCenters returns the center of all workouts of the current user
// @Summary      Get workout centers
// @Tags         heatmap
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  dto.Response[geojson.FeatureCollection]
// @Failure      500  {object}  dto.Response[any]
// @Router       /workouts/centers [get]
func (hc *heatmapController) GetWorkoutCenters(c echo.Context) error {
	coords := geojson.NewFeatureCollection()
	u := currentUser(c)

	wos, err := hc.workoutRepo.ListByUserID(u.ID)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	for _, w := range wos {
		if w.Data == nil {
			continue
		}

		p := w.Data.Center
		if p.IsZero() {
			continue
		}

		f := geojson.NewFeature(p.ToOrbPoint())
		f.Properties["popup_data"] = dto.NewWorkoutPopupData(w)

		coords.Append(f)
	}

	resp := dto.Response[*geojson.FeatureCollection]{
		Results: coords,
	}

	return c.JSON(http.StatusOK, resp)
}
