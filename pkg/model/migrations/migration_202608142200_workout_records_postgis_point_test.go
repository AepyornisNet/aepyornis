package migrations

import (
	"testing"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/restayway/gogis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration_WorkoutRecordsPostgisPoint(t *testing.T) {
	db := model.TestDB(t)

	// Create legacy table structure with lat/lng columns
	require.NoError(t, db.Exec("DROP TABLE IF EXISTS test_legacy_workout_records").Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE test_legacy_workout_records (
			id serial PRIMARY KEY,
			lat double precision,
			lng double precision,
			point geometry(Point, 4326)
		)
	`).Error)

	// Insert test records: valid point, zero point (indoor), null point
	require.NoError(t, db.Exec(`
		INSERT INTO test_legacy_workout_records (id, lat, lng) VALUES 
			(1, 50.95786, 4.72410),
			(2, 0.0, 0.0),
			(3, NULL, NULL)
	`).Error)

	// Run the migration update query
	require.NoError(t, db.Exec(`
		UPDATE test_legacy_workout_records 
		SET point = ST_SetSRID(ST_MakePoint(lng, lat), 4326) 
		WHERE point IS NULL AND (lat != 0 OR lng != 0)
	`).Error)

	type resultRow struct {
		ID    uint64       `gorm:"column:id"`
		Point *gogis.Point `gorm:"column:point"`
	}

	var results []resultRow
	require.NoError(t, db.Raw("SELECT id, point FROM test_legacy_workout_records ORDER BY id").Scan(&results).Error)
	require.Len(t, results, 3)

	// Record 1 (valid point) should have a non-nil PostGIS Point with correct lat/lng
	require.NotNil(t, results[0].Point)
	assert.InDelta(t, 50.95786, results[0].Point.Lat, 0.0001)
	assert.InDelta(t, 4.72410, results[0].Point.Lng, 0.0001)

	// Record 2 (0,0 indoor workout) must remain NULL
	assert.Nil(t, results[1].Point)

	// Record 3 (NULL) must remain NULL
	assert.Nil(t, results[2].Point)
}
