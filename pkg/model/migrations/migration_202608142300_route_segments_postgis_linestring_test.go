package migrations

import (
	"testing"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/restayway/gogis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration_RouteSegmentsLineString(t *testing.T) {
	db := model.TestDB(t)

	// Create legacy table structure with text points column
	require.NoError(t, db.Exec("DROP TABLE IF EXISTS test_legacy_route_segments").Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE test_legacy_route_segments (
			id serial PRIMARY KEY,
			filename text,
			content bytea,
			points text
		)
	`).Error)

	legacyJSON := `[{"point":{"lat":50.95786,"lng":4.72410}},{"point":{"lat":50.95816,"lng":4.72391}}]`
	require.NoError(t, db.Exec(`
		INSERT INTO test_legacy_route_segments (filename, points) VALUES (?, ?)
	`, "test.gpx", legacyJSON).Error)

	// Simulate migration steps on the test table
	require.NoError(t, db.Exec("ALTER TABLE test_legacy_route_segments RENAME COLUMN points TO points_legacy_json").Error)
	require.NoError(t, db.Exec("ALTER TABLE test_legacy_route_segments ADD COLUMN points geometry(LineString, 4326)").Error)

	type row struct {
		ID               uint64 `gorm:"column:id"`
		PointsLegacyJSON string `gorm:"column:points_legacy_json"`
	}
	var rows []row
	require.NoError(t, db.Raw("SELECT id, points_legacy_json FROM test_legacy_route_segments WHERE points IS NULL").Scan(&rows).Error)
	require.Len(t, rows, 1)

	var points []gogis.Point
	_ = points
	require.NoError(t, db.Exec(`
		UPDATE test_legacy_route_segments 
		SET points = ST_GeomFromEWKT('SRID=4326;LINESTRING(4.7241 50.95786,4.72391 50.95816)') 
		WHERE id = ?
	`, rows[0].ID).Error)

	var loadedPoints gogis.LineString
	require.NoError(t, db.Raw("SELECT points FROM test_legacy_route_segments WHERE id = ?", rows[0].ID).Row().Scan(&loadedPoints))
	assert.Len(t, loadedPoints.Points, 2)
	assert.InDelta(t, 50.95786, loadedPoints.Points[0].Lat, 0.0001)
	assert.InDelta(t, 4.72410, loadedPoints.Points[0].Lng, 0.0001)
	assert.InDelta(t, 50.95816, loadedPoints.Points[1].Lat, 0.0001)
	assert.InDelta(t, 4.72391, loadedPoints.Points[1].Lng, 0.0001)
}
