package controller

import (
	"errors"
	"net/http"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/model/dto"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/labstack/echo/v5"
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
	GetWorkoutCoordinates(c *echo.Context) error
	GetWorkoutCenters(c *echo.Context) error
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
	Lat    float64 `gorm:"column:lat"`
	Lng    float64 `gorm:"column:lng"`
	Weight int64   `gorm:"column:weight"`
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
// @Failure      400  {object}  dto.Response[string]
// @Failure      500  {object}  dto.Response[string]
// @Router       /workouts/coordinates [get]
func (hc *heatmapController) GetWorkoutCoordinates(c *echo.Context) error {
	var req dto.HeatmapCoordinatesRequest
	if err := c.Bind(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	cellSize := defaultHeatmapCellSize
	hasCellSize := false
	if req.CellSize != nil {
		cellSize = *req.CellSize
		hasCellSize = true
	}

	bounds, err := parseHeatmapBoundsFromRequest(&req)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	u := currentUser(c)

	filters, err := model.GetWorkoutsFilters(c)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	query := hc.db.Table("workout_records AS wr").
		Joins("JOIN workouts ON workouts.id = wr.workout_id").
		Where("workouts.profile_id = ?", u.Profile.ID).
		Where("wr.point IS NOT NULL")

	query = filters.ToQueryWithoutOrder(query)

	if bounds != nil {
		query = query.Where(
			"ST_Intersects(wr.point, ST_MakeEnvelope(?, ?, ?, ?, 4326))",
			bounds.minLng, bounds.minLat, bounds.maxLng, bounds.maxLat,
		)
	}

	if !hasCellSize {
		rows := make([]rawCoordinateRow, 0)
		if err := query.Select("ST_Y(wr.point) AS lat, ST_X(wr.point) AS lng").Find(&rows).Error; err != nil {
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
		Select("ST_Y(ST_SnapToGrid(wr.point, ?)) AS lat, ST_X(ST_SnapToGrid(wr.point, ?)) AS lng, count(*) AS weight", cellSize, cellSize).
		Group("1, 2").
		Find(&rows).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	coords := make([][]float64, 0, len(rows))
	for _, row := range rows {
		coords = append(coords, []float64{row.Lat, row.Lng, float64(row.Weight)})
	}

	resp := dto.Response[[][]float64]{
		Results: coords,
	}

	return c.JSON(http.StatusOK, resp)
}

func parseHeatmapBoundsFromRequest(req *dto.HeatmapCoordinatesRequest) (*heatmapBounds, error) {
	if req.MinLat == nil && req.MinLng == nil && req.MaxLat == nil && req.MaxLng == nil {
		return nil, nil
	}
	if req.MinLat == nil || req.MinLng == nil || req.MaxLat == nil || req.MaxLng == nil {
		return nil, errors.New("invalid viewport bounds")
	}

	minLat, minLng, maxLat, maxLng := *req.MinLat, *req.MinLng, *req.MaxLat, *req.MaxLng
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
// @Failure      500  {object}  dto.Response[string]
// @Router       /workouts/centers [get]
func (hc *heatmapController) GetWorkoutCenters(c *echo.Context) error {
	coords := geojson.NewFeatureCollection()
	u := currentUser(c)

	filters, err := model.GetWorkoutsFilters(c)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	wos, err := hc.workoutRepo.ListByProfileAndFilters(u.Profile.ID, filters, 0, 0)
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
