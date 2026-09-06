package migrations

import (
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"gorm.io/gorm"
)

func init() {
	model.RegisterMigration(
		202609061800,
		"ensure route_segment_matches has surrogate id primary key and drops composite unique constraints",
		preRouteSegmentMatchesSurrogatePKFixMigrate,
		nil,
		nil,
		nil,
	)
}

func preRouteSegmentMatchesSurrogatePKFixMigrate(db *gorm.DB) error {
	if !db.Migrator().HasTable("route_segment_matches") {
		return nil
	}

	if db.Dialector.Name() == "postgres" {
		sql := `
		DO $$
		BEGIN
			-- 1. Ensure id column exists
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'route_segment_matches' AND column_name = 'id'
			) THEN
				ALTER TABLE route_segment_matches ADD COLUMN id BIGSERIAL;
			END IF;

			-- 2. Drop existing primary key constraint if it is not exclusively on id
			IF EXISTS (
				SELECT 1
				FROM information_schema.table_constraints tc
				JOIN information_schema.key_column_usage kcu
				  ON tc.constraint_name = kcu.constraint_name
				 AND tc.table_schema = kcu.table_schema
				WHERE tc.table_name = 'route_segment_matches'
				  AND tc.constraint_type = 'PRIMARY KEY'
				  AND kcu.column_name != 'id'
			) THEN
				ALTER TABLE route_segment_matches DROP CONSTRAINT IF EXISTS route_segment_matches_pkey;
			END IF;

			-- 3. If primary key constraint does not exist, add it on id
			IF NOT EXISTS (
				SELECT 1
				FROM information_schema.table_constraints
				WHERE table_name = 'route_segment_matches'
				  AND constraint_type = 'PRIMARY KEY'
			) THEN
				-- Ensure id sequence exists and default is nextval
				IF NOT EXISTS (
					SELECT 1 FROM pg_sequences WHERE sequencename = 'route_segment_matches_id_seq'
				) THEN
					CREATE SEQUENCE IF NOT EXISTS route_segment_matches_id_seq;
					ALTER TABLE route_segment_matches ALTER COLUMN id SET DEFAULT nextval('route_segment_matches_id_seq');
					ALTER SEQUENCE route_segment_matches_id_seq OWNED BY route_segment_matches.id;
				END IF;
				PERFORM setval('route_segment_matches_id_seq', COALESCE((SELECT MAX(id) FROM route_segment_matches), 0) + 1, false);
				ALTER TABLE route_segment_matches ADD PRIMARY KEY (id);
			END IF;

			-- 4. Drop any unique index/constraint on (route_segment_id, workout_id)
			DROP INDEX IF EXISTS idx_route_segment_matches_segment_workout;
			CREATE INDEX IF NOT EXISTS idx_route_segment_matches_segment_workout ON route_segment_matches(route_segment_id, workout_id);
			CREATE INDEX IF NOT EXISTS idx_route_segment_matches_route_segment_id ON route_segment_matches(route_segment_id);
			CREATE INDEX IF NOT EXISTS idx_route_segment_matches_workout_id ON route_segment_matches(workout_id);
		END $$;
		`
		if err := db.Exec(sql).Error; err != nil {
			return err
		}
	} else {
		_ = db.Exec("CREATE INDEX IF NOT EXISTS idx_route_segment_matches_route_segment_id ON route_segment_matches(route_segment_id)").Error
		_ = db.Exec("CREATE INDEX IF NOT EXISTS idx_route_segment_matches_workout_id ON route_segment_matches(workout_id)").Error
		_ = db.Exec("CREATE INDEX IF NOT EXISTS idx_route_segment_matches_segment_workout ON route_segment_matches(route_segment_id, workout_id)").Error
	}

	return nil
}
