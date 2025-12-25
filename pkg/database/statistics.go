package database

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

const postgresDialect = "postgres"

var ErrAnonymousUser = errors.New("no statistics available for anonymous user")

type StatConfig struct {
	Since string `query:"since"`
	Per   string `query:"per"`
}

func loadWorkoutsForRecords(db *gorm.DB, userID uint64, t WorkoutType, startDate, endDate *time.Time) ([]*Workout, error) {
	var workouts []*Workout

	query := db.Preload("Data").Where("user_id = ?", userID).Where("workouts.type = ?", t)

	if startDate != nil {
		query = query.Where("workouts.date >= ?", *startDate)
	}

	if endDate != nil {
		query = query.Where("workouts.date <= ?", *endDate)
	}

	if err := query.Find(&workouts).Error; err != nil {
		return nil, err
	}

	return workouts, nil
}

func (sc *StatConfig) GetBucketString(sqlDialect string) string {
	switch sqlDialect {
	case postgresDialect:
		switch sc.Per {
		case "year":
			return "YYYY"
		case "week":
			return "YYYY-WW"
		case "day":
			return "YYYY-MM-DD"
		default:
			return "YYYY-MM"
		}
	default:
		switch sc.Per {
		case "year":
			return "%Y"
		case "week":
			return "%Y-%W"
		case "day":
			return "%Y-%m-%d"
		default:
			return "%Y-%m"
		}
	}
}

func (sc *StatConfig) GetDayBucketFormatExpression(sqlDialect string) string {
	switch sqlDialect {
	case postgresDialect:
		return "min(to_char(workouts.date, 'YYYY-MM-DD')) as bucket"
	default:
		return "min(strftime('%Y-%m-%d', workouts.date)) as bucket"
	}
}

func (sc *StatConfig) GetBucketFormatExpression(sqlDialect string) string {
	switch sqlDialect {
	case postgresDialect:
		return fmt.Sprintf("to_char(workouts.date, '%s') as raw_bucket", sc.GetBucketString(sqlDialect))
	default:
		return fmt.Sprintf("strftime('%s', workouts.date) as raw_bucket", sc.GetBucketString(sqlDialect))
	}
}

func GetDateLimitExpression(sqlDialect string) string {
	switch sqlDialect {
	case postgresDialect:
		return "workouts.date > CURRENT_DATE + cast(? as interval)"
	default:
		return "workouts.date > DATE(CURRENT_DATE, ?)"
	}
}

func (sc *StatConfig) GetSince() string {
	s := sc.Since
	if s == "" {
		s = "1 year"
	}

	return s
}

func (u *User) GetDefaultStatistics() (*Statistics, error) {
	return u.GetStatisticsFor("misc.years_1", "misc.month")
}

func (u *User) GetStatisticsFor(since, per string) (*Statistics, error) {
	return u.GetStatistics(StatConfig{
		Since: since,
		Per:   per,
	})
}

func (u *User) GetStatistics(statConfig StatConfig) (*Statistics, error) {
	sqlDialect := u.db.Dialector.Name()

	r := &Statistics{
		UserID:       u.ID,
		BucketFormat: statConfig.GetBucketString(sqlDialect),
		Buckets:      map[WorkoutType]Buckets{},
	}

	q := u.db.
		Table("workouts").
		Select(
			"count(*) as workouts",
			"workouts.type as workout_type",
			"sum(total_duration) as duration",
			"sum(total_distance) as distance",
			"sum(total_up) as up",
			"max(max_speed) as max_speed",
			"sum(total_duration * average_speed) / sum(total_duration) as average_speed",
			"sum((total_duration - pause_duration) * average_speed_no_pause) / NULLIF(sum(total_duration - pause_duration), 0) as average_speed_no_pause",
			statConfig.GetBucketFormatExpression(sqlDialect),
			statConfig.GetDayBucketFormatExpression(sqlDialect),
		).
		Joins("join map_data on workouts.id = map_data.workout_id").
		Where("user_id = ?", u.ID)

	if statConfig.Since != "" && statConfig.Since != "forever" {
		q = q.Where(GetDateLimitExpression(sqlDialect), "-"+statConfig.GetSince())
	}

	// Grouping by `raw_bucket` instead of `bucket` ensures that the data is grouped
	// based on the raw, unprocessed bucket values as defined in the database schema.
	// This is necessary to maintain consistency with the bucket format expressions
	// used in the SELECT clause and to avoid potential mismatches caused by
	// transformations or processing applied to `bucket`.
	// The `bucket` field is provided for frontend rendering purposes only.
	rows, err := q.Group("raw_bucket, workout_type").Rows()
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var result Bucket

	units := u.PreferredUnits()

	for rows.Next() {
		if err := u.db.ScanRows(rows, &result); err != nil {
			return nil, err
		}

		if _, ok := r.Buckets[result.WorkoutType]; !ok {
			r.Buckets[result.WorkoutType] = Buckets{
				WorkoutType:      result.WorkoutType,
				LocalWorkoutType: u.I18n(result.WorkoutType.StringT()),
				Buckets:          map[string]Bucket{},
			}
		}

		result.Localize(units)

		r.Buckets[result.WorkoutType].Buckets[result.Bucket] = result
	}

	return r, nil
}

func (u *User) GetHighestWorkoutType() (*WorkoutType, error) {
	r := ""

	err := u.db.
		Table("workouts").
		Select("workouts.type").
		Where("user_id = ?", u.ID).
		Group("workouts.type").
		Order("count(*) DESC").
		Limit(1).
		Pluck("workouts.type", &r).Error
	if err != nil {
		return nil, err
	}

	wt := AsWorkoutType(r)

	return &wt, nil
}

func (u *User) GetDefaultTotals(startDate, endDate *time.Time) (*Bucket, error) {
	if u.IsAnonymous() {
		return nil, ErrAnonymousUser
	}

	t := u.Profile.TotalsShow
	if t == WorkoutTypeAutoDetect {
		ht, err := u.GetHighestWorkoutType()
		if err != nil {
			return nil, err
		}

		t = *ht
	}

	return u.GetTotals(t, startDate, endDate)
}

func (u *User) GetTotals(t WorkoutType, startDate, endDate *time.Time) (*Bucket, error) {
	if t == "" {
		t = WorkoutTypeRunning
	}

	r := &Bucket{}

	query := u.db.
		Table("workouts").
		Select(
			"count(*) as workouts",
			"max(workouts.type) as workout_type",
			"sum(total_duration) as duration",
			"sum(total_distance) as distance",
			"sum(total_up) as up",
			"'all' as bucket",
		).
		Joins("join map_data on workouts.id = map_data.workout_id").
		Where("user_id = ?", u.ID).
		Where("workouts.type = ?", t)

	if startDate != nil {
		query = query.Where("workouts.date >= ?", *startDate)
	}

	if endDate != nil {
		query = query.Where("workouts.date <= ?", *endDate)
	}

	err := query.Scan(r).Error
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (u *User) GetAllRecords(startDate, endDate *time.Time) ([]*WorkoutRecord, error) {
	if u.IsAnonymous() {
		return nil, ErrAnonymousUser
	}

	rs := []*WorkoutRecord{}

	for _, w := range DistanceWorkoutTypes() {
		r, err := u.GetRecords(w, startDate, endDate)
		if err != nil {
			return nil, err
		}

		if r != nil {
			rs = append(rs, r)
		}
	}

	return rs, nil
}

func (u *User) getStoredDistanceRecords(t WorkoutType, startDate, endDate *time.Time) ([]DistanceRecord, error) {
	targets := distanceRecordTargetsFor(t)
	if len(targets) == 0 {
		return nil, nil
	}

	validLabels := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		validLabels[target.Label] = struct{}{}
	}

	rows := []struct {
		WorkoutIntervalRecord
		Date time.Time
	}{}

	q := u.db.Table("workout_interval_records").
		Select("workout_interval_records.*, workouts.date as date").
		Joins("join workouts on workouts.id = workout_interval_records.workout_id").
		Where("workouts.user_id = ?", u.ID).
		Where("workouts.type = ?", t)

	if startDate != nil {
		q = q.Where("workouts.date >= ?", *startDate)
	}

	if endDate != nil {
		q = q.Where("workouts.date <= ?", *endDate)
	}

	q = q.Order("workout_interval_records.label asc, workout_interval_records.target_distance asc, workout_interval_records.duration_seconds asc, workout_interval_records.distance desc")

	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}

	best := map[string]DistanceRecord{}

	for _, r := range rows {
		if _, ok := validLabels[r.Label]; !ok {
			continue
		}

		candidate := DistanceRecord{
			Label:          r.Label,
			TargetDistance: r.TargetDistance,
			Distance:       r.Distance,
			Duration:       time.Duration(r.DurationSeconds * float64(time.Second)),
			AverageSpeed:   r.AverageSpeed,
			WorkoutID:      r.WorkoutID,
			Date:           r.Date,
			StartIndex:     r.StartIndex,
			EndIndex:       r.EndIndex,
			Active:         true,
		}

		current, ok := best[candidate.Label]
		if !ok || betterDistanceRecord(candidate, current) {
			best[candidate.Label] = candidate
		}
	}

	result := make([]DistanceRecord, 0, len(targets))
	for _, target := range targets {
		if rec, ok := best[target.Label]; ok {
			result = append(result, rec)
		}
	}

	return result, nil
}

// GetDistanceRecordRanking returns stored interval efforts for a distance label ordered best-first with pagination.
func (u *User) GetDistanceRecordRanking(t WorkoutType, label string, startDate, endDate *time.Time, limit, offset int) ([]DistanceRecord, int64, error) {
	targets := distanceRecordTargetsFor(t)
	if len(targets) == 0 {
		return nil, 0, nil
	}

	valid := false
	for _, target := range targets {
		if target.Label == label {
			valid = true
			break
		}
	}

	if !valid {
		return nil, 0, fmt.Errorf("unknown distance label %q for workout type %s", label, t)
	}

	rows := []struct {
		WorkoutIntervalRecord
		Date time.Time
	}{}

	base := u.db.Table("workout_interval_records").
		Select("workout_interval_records.*, workouts.date as date").
		Joins("join workouts on workouts.id = workout_interval_records.workout_id").
		Where("workouts.user_id = ?", u.ID).
		Where("workouts.type = ?", t).
		Where("workout_interval_records.label = ?", label)

	if startDate != nil {
		base = base.Where("workouts.date >= ?", *startDate)
	}

	if endDate != nil {
		base = base.Where("workouts.date <= ?", *endDate)
	}

	var totalCount int64
	if err := base.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	q := base

	if limit > 0 {
		q = q.Limit(limit)
	}

	if offset > 0 {
		q = q.Offset(offset)
	}

	q = q.Order("workout_interval_records.duration_seconds asc, workout_interval_records.distance desc, workouts.date asc, workout_interval_records.workout_id asc")

	if err := q.Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	result := make([]DistanceRecord, 0, len(rows))
	for _, r := range rows {
		result = append(result, DistanceRecord{
			Label:          r.Label,
			TargetDistance: r.TargetDistance,
			Distance:       r.Distance,
			Duration:       time.Duration(r.DurationSeconds * float64(time.Second)),
			AverageSpeed:   r.AverageSpeed,
			WorkoutID:      r.WorkoutID,
			Date:           r.Date,
			StartIndex:     r.StartIndex,
			EndIndex:       r.EndIndex,
			Active:         true,
		})
	}

	return result, totalCount, nil
}

//nolint:gocyclo // queries gather several aggregates in one pass
func (u *User) GetRecords(t WorkoutType, startDate, endDate *time.Time) (*WorkoutRecord, error) {
	if t == "" {
		t = u.Profile.TotalsShow
	}

	r := &WorkoutRecord{WorkoutType: t}

	mapping := map[*Float64Record]string{
		&r.Distance:            "max(total_distance)",
		&r.MaxSpeed:            "max(max_speed)",
		&r.TotalUp:             "max(total_up)",
		&r.AverageSpeed:        "max(average_speed)",
		&r.AverageSpeedNoPause: "max(average_speed_no_pause)",
	}

	for k, v := range mapping {
		query := u.db.
			Table("workouts").
			Joins("join map_data on workouts.id = map_data.workout_id").
			Where("user_id = ?", u.ID).
			Where("workouts.type = ?", t).
			Select("workouts.id as id", v+" as value", "workouts.date as date").
			Order(v + " DESC").
			Group("workouts.id").
			Limit(1)

		if startDate != nil {
			query = query.Where("workouts.date >= ?", *startDate)
		}

		if endDate != nil {
			query = query.Where("workouts.date <= ?", *endDate)
		}

		err := query.Scan(k).Error
		if err != nil {
			return nil, err
		}
	}

	query := u.db.
		Table("workouts").
		Joins("join map_data on workouts.id = map_data.workout_id").
		Where("user_id = ?", u.ID).
		Where("workouts.type = ?", t).
		Select("workouts.id as id", "max(total_duration) as value", "workouts.date as date").
		Order("max(total_duration) DESC").
		Group("workouts.id").
		Limit(1)

	if startDate != nil {
		query = query.Where("workouts.date >= ?", *startDate)
	}

	if endDate != nil {
		query = query.Where("workouts.date <= ?", *endDate)
	}

	err := query.Scan(&r.Duration).Error
	if err != nil {
		return nil, err
	}

	targets := distanceRecordTargetsFor(t)

	if len(targets) > 0 {
		dr, derr := u.getStoredDistanceRecords(t, startDate, endDate)
		if derr != nil {
			return nil, derr
		}
		r.DistanceRecords = dr
	}

	if t.IsDistance() {
		workouts, werr := loadWorkoutsForRecords(u.db, u.ID, t, startDate, endDate)
		if werr != nil {
			return nil, werr
		}

		if climb := biggestClimbRecord(workouts); climb != nil && climb.Active {
			r.BiggestClimb = climb
		}
	}

	r.Active = r.Distance.Value > 0 ||
		r.MaxSpeed.Value > 0 ||
		r.TotalUp.Value > 0 ||
		r.Duration.Value > 0 ||
		len(r.DistanceRecords) > 0 ||
		(r.BiggestClimb != nil && r.BiggestClimb.Active)

	return r, nil
}
