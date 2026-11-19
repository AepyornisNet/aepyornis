package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/aputil"
	"github.com/AepyornisNet/aepyornis/pkg/config"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/model/dto"
	"github.com/AepyornisNet/aepyornis/pkg/notification"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	vocab "github.com/go-ap/activitypub"
	"github.com/samber/do/v2"
	"gorm.io/gorm"
)

var (
	ErrCannotFollowSelf            = errors.New("cannot follow yourself")
	ErrCannotUnfollowSelf          = errors.New("cannot unfollow yourself")
	ErrActivityPubRequiredToFollow = errors.New("activitypub must be enabled to follow users")
	ErrTargetNoActivityPub         = errors.New("target user does not have activitypub enabled")
	ErrRemoteInboxNotFound         = errors.New("remote actor inbox not found")
)

type UserService interface {
	GetTotals(ctx context.Context, targetUser, viewer *model.User, startDate, endDate *time.Time) ([]model.Bucket, error)
	GetVisibleRecords(ctx context.Context, targetUser, viewer *model.User, startDate, endDate *time.Time) ([]*model.WorkoutPersonalRecord, error)
	GetVisibleRecordForType(targetUser *model.User, viewerProfileID uint64, t model.WorkoutType, startDate, endDate *time.Time) (*model.WorkoutPersonalRecord, error)
	GetVisibleStoredDistanceRecords(targetUser *model.User, viewerProfileID uint64, t model.WorkoutType, startDate, endDate *time.Time) ([]model.DistanceRecord, error)
	GetVisibleStoredPowerRecords(targetUser *model.User, viewerProfileID uint64, t model.WorkoutType, startDate, endDate *time.Time) ([]model.PowerRecord, error)
	GetDistanceRanking(ctx context.Context, targetUser, viewer *model.User, wt model.WorkoutType, label string, startDate, endDate *time.Time, limit, offset int) ([]model.DistanceRecord, int64, error)
	GetPowerRanking(ctx context.Context, targetUser, viewer *model.User, wt model.WorkoutType, label string, startDate, endDate *time.Time, limit, offset int) ([]model.PowerRecord, int64, error)
	GetClimbRanking(ctx context.Context, targetUser, viewer *model.User, wt model.WorkoutType, startDate, endDate *time.Time, limit, offset int) ([]model.ClimbRecord, int64, error)
	SearchProfiles(ctx context.Context, viewer *model.User, query, requestHost, fallbackScheme string) ([]dto.ActivityPubProfileSummaryResponse, error)
	GetUserProfileByHandle(ctx context.Context, viewer *model.User, handle, requestHost, fallbackScheme string) (dto.ActivityPubProfileSummaryResponse, error)
	FollowUserByHandle(ctx context.Context, viewer *model.User, handle, requestHost, fallbackScheme string) (dto.ActivityPubProfileSummaryResponse, error)
	UnfollowUserByHandle(ctx context.Context, viewer *model.User, handle, requestHost, fallbackScheme string) (dto.ActivityPubProfileSummaryResponse, error)
	LocalHost(requestHost string) string
	ResolveTargetUser(handle, requestHost string, viewer *model.User) (*model.User, error)
}

type userService struct {
	cfg          *config.Config
	db           *gorm.DB
	logger       *slog.Logger
	apProfileSvc ActivityPubProfileService
	actorService ActivityPubActorService
	notify       NotificationService
	followerRepo repository.Follower
	userRepo     repository.User
}

func NewUserService(injector do.Injector) (UserService, error) {
	return &userService{
		cfg:          do.MustInvoke[*config.Config](injector),
		db:           do.MustInvoke[*gorm.DB](injector),
		logger:       do.MustInvoke[*slog.Logger](injector),
		apProfileSvc: do.MustInvoke[ActivityPubProfileService](injector),
		actorService: do.MustInvoke[ActivityPubActorService](injector),
		notify:       do.MustInvoke[NotificationService](injector),
		followerRepo: do.MustInvoke[repository.Follower](injector),
		userRepo:     do.MustInvoke[repository.User](injector),
	}, nil
}

func (s *userService) LocalHost(requestHost string) string {
	if s.cfg.Host != "" {
		return s.cfg.Host
	}
	return requestHost
}

func (s *userService) isLocalHost(requestHost, host string) bool {
	return strings.EqualFold(strings.TrimSpace(host), strings.TrimSpace(s.LocalHost(requestHost)))
}

func (s *userService) renderHandle(requestHost, username string) string {
	return fmt.Sprintf("@%s@%s", username, s.LocalHost(requestHost))
}

func (s *userService) localActorIRI(requestHost, fallbackScheme string, user *model.User) string {
	if user == nil {
		return ""
	}
	return aputil.LocalActorURL(aputil.LocalActorURLConfig{
		Host:           s.cfg.Host,
		WebRoot:        s.cfg.WebRoot,
		FallbackHost:   requestHost,
		FallbackScheme: fallbackScheme,
	}, user.Profile.Username)
}

func (s *userService) parseHandleWithHost(requestHost, handle string) (string, string, bool, error) {
	username, host, err := aputil.ParseActorHandle(handle)
	if err != nil {
		return "", "", false, gorm.ErrRecordNotFound
	}
	return username, host, host != "" && !s.isLocalHost(requestHost, host), nil
}

func (s *userService) ResolveTargetUser(handle, requestHost string, viewer *model.User) (*model.User, error) {
	if handle != "" {
		return s.userRepo.GetByHandle(handle, s.LocalHost(requestHost))
	}
	if viewer == nil || viewer.IsAnonymous() {
		return nil, dto.ErrNotAuthorized
	}
	return viewer, nil
}

func (s *userService) GetTotals(_ context.Context, targetUser, viewer *model.User, startDate, endDate *time.Time) ([]model.Bucket, error) {
	totalsQuery := model.ScopeVisibleWorkouts(
		s.db.Table("workouts").
			Select(
				"count(*) as workouts",
				"max(workouts.type) as workout_type",
				"sum(total_duration) as duration",
				"sum(total_distance) as distance",
				"sum(coalesce(workout_stats.total_up, 0)) as up",
				"'all' as bucket",
			).
			Joins("left join workout_stats on workouts.stats_id = workout_stats.id").
			Joins("left join workout_geo_meta on workouts.id = workout_geo_meta.workout_id").
			Group("workouts.type"),
		targetUser.Profile.ID,
		viewer.Profile.ID,
	)

	totalsShow := targetUser.TotalsShow
	if totalsShow == model.WorkoutTypeAutoDetect {
		totalsShow = model.WorkoutTypeRunning
	}
	if totalsShow != "all" {
		totalsQuery = totalsQuery.Where("workouts.type = ?", totalsShow)
	}
	if startDate != nil {
		totalsQuery = totalsQuery.Where("workouts.date >= ?", *startDate)
	}
	if endDate != nil {
		totalsQuery = totalsQuery.Where("workouts.date <= ?", *endDate)
	}

	rows, err := totalsQuery.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var totals []model.Bucket
	for rows.Next() {
		var total model.Bucket
		if err := s.db.ScanRows(rows, &total); err != nil {
			return nil, err
		}
		totals = append(totals, total)
	}
	return totals, nil
}

func (s *userService) GetVisibleRecords(_ context.Context, targetUser, viewer *model.User, startDate, endDate *time.Time) ([]*model.WorkoutPersonalRecord, error) {
	if targetUser == nil || targetUser.IsAnonymous() {
		return nil, model.ErrAnonymousUser
	}

	if viewer != nil && viewer.ID != 0 && viewer.ID == targetUser.ID {
		return targetUser.GetAllPersonalRecords(startDate, endDate)
	}

	rs := make([]*model.WorkoutPersonalRecord, 0, len(model.DistanceWorkoutTypes()))
	for _, w := range model.DistanceWorkoutTypes() {
		r, err := s.GetVisibleRecordForType(targetUser, viewer.Profile.ID, w, startDate, endDate)
		if err != nil {
			return nil, err
		}
		if r != nil {
			rs = append(rs, r)
		}
	}
	return rs, nil
}

func (s *userService) GetVisibleRecordForType(targetUser *model.User, viewerProfileID uint64, t model.WorkoutType, startDate, endDate *time.Time) (*model.WorkoutPersonalRecord, error) {
	if t == "" {
		t = model.WorkoutTypeRunning
		if targetUser != nil {
			t = targetUser.TotalsShow
		}
	}

	r := &model.WorkoutPersonalRecord{WorkoutType: t}
	mapping := map[*model.Float64Record]string{
		&r.Distance:            "max(total_distance)",
		&r.MaxSpeed:            "max(workout_stats.max_speed)",
		&r.TotalUp:             "max(workout_stats.total_up)",
		&r.AverageSpeed:        "max(workout_stats.average_speed)",
		&r.AverageSpeedNoPause: "max(workout_stats.average_speed_no_pause)",
	}

	for k, v := range mapping {
		query := model.ScopeExcludeRecordWorkouts(model.ScopeVisibleWorkouts(
			s.db.Table("workouts").Joins("left join workout_stats on workouts.stats_id = workout_stats.id").Joins("left join workout_geo_meta on workouts.id = workout_geo_meta.workout_id"),
			targetUser.Profile.ID,
			viewerProfileID,
		)).
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
		if err := query.Scan(k).Error; err != nil {
			return nil, err
		}
	}

	durationQuery := model.ScopeVisibleWorkouts(
		s.db.Table("workouts").Joins("left join workout_geo_meta on workouts.id = workout_geo_meta.workout_id"),
		targetUser.Profile.ID,
		viewerProfileID,
	).Where("workouts.type = ?", t).
		Select("workouts.id as id", "max(total_duration) as value", "workouts.date as date").
		Order("max(total_duration) DESC").
		Group("workouts.id").
		Limit(1)

	if startDate != nil {
		durationQuery = durationQuery.Where("workouts.date >= ?", *startDate)
	}
	if endDate != nil {
		durationQuery = durationQuery.Where("workouts.date <= ?", *endDate)
	}
	if err := durationQuery.Scan(&r.Duration).Error; err != nil {
		return nil, err
	}

	targets := model.DistanceRecordTargetsFor(t)
	if len(targets) > 0 {
		dr, derr := s.GetVisibleStoredDistanceRecords(targetUser, viewerProfileID, t, startDate, endDate)
		if derr != nil {
			return nil, derr
		}
		r.DistanceRecords = dr
	}

	powerTargets := model.PowerRecordTargetsFor(t)
	if len(powerTargets) > 0 {
		pr, perr := s.GetVisibleStoredPowerRecords(targetUser, viewerProfileID, t, startDate, endDate)
		if perr != nil {
			return nil, perr
		}
		r.PowerRecords = pr
	}

	r.Active = r.Distance.Value > 0 ||
		r.MaxSpeed.Value > 0 ||
		r.TotalUp.Value > 0 ||
		r.Duration.Value > 0 ||
		len(r.DistanceRecords) > 0 ||
		len(r.PowerRecords) > 0

	return r, nil
}

func (s *userService) GetVisibleStoredDistanceRecords(targetUser *model.User, viewerProfileID uint64, t model.WorkoutType, startDate, endDate *time.Time) ([]model.DistanceRecord, error) {
	targets := model.DistanceRecordTargetsFor(t)
	if len(targets) == 0 {
		return nil, nil
	}

	validLabels := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		validLabels[target.Label] = struct{}{}
	}

	type intervalRow struct {
		model.WorkoutIntervalBest
		Date time.Time
	}
	var rows []intervalRow

	q := model.ScopeExcludeRecordWorkouts(model.ScopeVisibleWorkouts(
		s.db.Table("workout_interval_records").
			Select("workout_interval_records.*, workouts.date as date").
			Joins("join workouts on workouts.id = workout_interval_records.workout_id"),
		targetUser.Profile.ID,
		viewerProfileID,
	)).Where("workouts.type = ?", t).
		Where("workout_interval_records.type = ?", model.WorkoutIntervalBestTypeSpeed)

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

	best := map[string]model.DistanceRecord{}
	for _, r := range rows {
		if _, ok := validLabels[r.Label]; !ok {
			continue
		}
		candidate := model.DistanceRecord{
			Label:          r.Label,
			TargetDistance: r.TargetDistance,
			Distance:       r.Distance,
			Duration:       time.Duration(r.DurationSeconds * float64(time.Second)),
			AverageSpeed:   r.Average,
			WorkoutID:      r.WorkoutID,
			Date:           r.Date,
			StartIndex:     r.StartIndex,
			EndIndex:       r.EndIndex,
			Active:         true,
		}
		current, ok := best[candidate.Label]
		if !ok || (candidate.Duration < current.Duration || (candidate.Duration == current.Duration && candidate.Distance > current.Distance)) {
			best[candidate.Label] = candidate
		}
	}

	result := make([]model.DistanceRecord, 0, len(targets))
	for _, target := range targets {
		if rec, ok := best[target.Label]; ok {
			result = append(result, rec)
		}
	}
	return result, nil
}

func (s *userService) GetVisibleStoredPowerRecords(targetUser *model.User, viewerProfileID uint64, t model.WorkoutType, startDate, endDate *time.Time) ([]model.PowerRecord, error) {
	targets := model.PowerRecordTargetsFor(t)
	if len(targets) == 0 {
		return nil, nil
	}

	validLabels := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		validLabels[target.Label] = struct{}{}
	}

	type intervalRow struct {
		model.WorkoutIntervalBest
		Date time.Time
	}
	var rows []intervalRow

	q := model.ScopeVisibleWorkouts(
		s.db.Table("workout_interval_records").
			Select("workout_interval_records.*, workouts.date as date").
			Joins("join workouts on workouts.id = workout_interval_records.workout_id"),
		targetUser.Profile.ID,
		viewerProfileID,
	).Where("workouts.type = ?", t).
		Where("workout_interval_records.type = ?", model.WorkoutIntervalBestTypePower)

	if startDate != nil {
		q = q.Where("workouts.date >= ?", *startDate)
	}
	if endDate != nil {
		q = q.Where("workouts.date <= ?", *endDate)
	}

	q = q.Order("workout_interval_records.label asc, workout_interval_records.target_distance asc, workout_interval_records.average desc, workout_interval_records.duration_seconds desc")
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}

	best := map[string]model.PowerRecord{}
	for _, r := range rows {
		if _, ok := validLabels[r.Label]; !ok {
			continue
		}
		speed := 0.0
		if r.DurationSeconds > 0 {
			speed = r.Distance / r.DurationSeconds
		}
		candidate := model.PowerRecord{
			Label:          r.Label,
			TargetDuration: r.TargetDistance,
			Distance:       r.Distance,
			Duration:       time.Duration(r.DurationSeconds * float64(time.Second)),
			AveragePower:   r.Average,
			AverageSpeed:   speed,
			WorkoutID:      r.WorkoutID,
			Date:           r.Date,
			StartIndex:     r.StartIndex,
			EndIndex:       r.EndIndex,
			Active:         true,
		}
		current, ok := best[candidate.Label]
		if !ok || (candidate.AveragePower > current.AveragePower || (candidate.AveragePower == current.AveragePower && candidate.Duration > current.Duration)) {
			best[candidate.Label] = candidate
		}
	}

	result := make([]model.PowerRecord, 0, len(targets))
	for _, target := range targets {
		if rec, ok := best[target.Label]; ok {
			result = append(result, rec)
		}
	}
	return result, nil
}

func (s *userService) GetDistanceRanking(
	_ context.Context,
	targetUser, viewer *model.User,
	t model.WorkoutType,
	label string,
	startDate, endDate *time.Time,
	limit, offset int,
) ([]model.DistanceRecord, int64, error) {
	type intervalRow struct {
		model.WorkoutIntervalBest
		Date time.Time
	}
	var rows []intervalRow

	base := model.ScopeExcludeRecordWorkouts(model.ScopeVisibleWorkouts(
		s.db.Table("workout_interval_records").
			Select("workout_interval_records.*, workouts.date as date").
			Joins("join workouts on workouts.id = workout_interval_records.workout_id"),
		targetUser.Profile.ID,
		viewer.Profile.ID,
	)).Where("workouts.type = ?", t).
		Where("workout_interval_records.type = ?", model.WorkoutIntervalBestTypeSpeed).
		Where("workout_interval_records.label = ?", label)

	if startDate != nil {
		base = base.Where("workouts.date >= ?", *startDate)
	}
	if endDate != nil {
		base = base.Where("workouts.date <= ?", *endDate)
	}

	ranked := s.db.Table("(?) AS ranked_records", base.
		Select("workout_interval_records.*, workouts.date as date, ROW_NUMBER() OVER (PARTITION BY workout_interval_records.workout_id ORDER BY workout_interval_records.duration_seconds ASC, workout_interval_records.distance DESC) as rn"))

	var totalCount int64
	if err := s.db.Table("(?) AS filtered", ranked.Where("rn = 1")).Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	q := ranked.Where("rn = 1").
		Order("duration_seconds ASC, distance DESC").
		Limit(limit).
		Offset(offset)

	if err := q.Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	records := make([]model.DistanceRecord, len(rows))
	for i, r := range rows {
		records[i] = model.DistanceRecord{
			Label:          r.Label,
			TargetDistance: r.TargetDistance,
			Distance:       r.Distance,
			Duration:       time.Duration(r.DurationSeconds * float64(time.Second)),
			AverageSpeed:   r.Average,
			WorkoutID:      r.WorkoutID,
			Date:           r.Date,
			StartIndex:     r.StartIndex,
			EndIndex:       r.EndIndex,
			Active:         true,
		}
	}
	return records, totalCount, nil
}

func (s *userService) GetPowerRanking(
	_ context.Context,
	targetUser, viewer *model.User,
	t model.WorkoutType,
	label string,
	startDate, endDate *time.Time,
	limit, offset int,
) ([]model.PowerRecord, int64, error) {
	type intervalRow struct {
		model.WorkoutIntervalBest
		Date time.Time
	}
	var rows []intervalRow

	base := model.ScopeExcludeRecordWorkouts(model.ScopeVisibleWorkouts(
		s.db.Table("workout_interval_records").
			Select("workout_interval_records.*, workouts.date as date").
			Joins("join workouts on workouts.id = workout_interval_records.workout_id"),
		targetUser.Profile.ID,
		viewer.Profile.ID,
	)).Where("workouts.type = ?", t).
		Where("workout_interval_records.type = ?", model.WorkoutIntervalBestTypePower).
		Where("workout_interval_records.label = ?", label)

	if startDate != nil {
		base = base.Where("workouts.date >= ?", *startDate)
	}
	if endDate != nil {
		base = base.Where("workouts.date <= ?", *endDate)
	}

	ranked := s.db.Table("(?) AS ranked_records", base.
		Select("workout_interval_records.*, workouts.date as date, ROW_NUMBER() OVER (PARTITION BY workout_interval_records.workout_id ORDER BY workout_interval_records.average DESC, workout_interval_records.duration_seconds DESC) as rn"))

	var totalCount int64
	if err := s.db.Table("(?) AS filtered", ranked.Where("rn = 1")).Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	q := ranked.Where("rn = 1").
		Order("average DESC, duration_seconds DESC").
		Limit(limit).
		Offset(offset)

	if err := q.Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	records := make([]model.PowerRecord, len(rows))
	for i, r := range rows {
		speed := 0.0
		if r.DurationSeconds > 0 {
			speed = r.Distance / r.DurationSeconds
		}
		records[i] = model.PowerRecord{
			Label:          r.Label,
			TargetDuration: r.TargetDistance,
			Distance:       r.Distance,
			Duration:       time.Duration(r.DurationSeconds * float64(time.Second)),
			AveragePower:   r.Average,
			AverageSpeed:   speed,
			WorkoutID:      r.WorkoutID,
			Date:           r.Date,
			StartIndex:     r.StartIndex,
			EndIndex:       r.EndIndex,
			Active:         true,
		}
	}
	return records, totalCount, nil
}

func (s *userService) GetClimbRanking(
	_ context.Context,
	targetUser, viewer *model.User,
	t model.WorkoutType,
	startDate, endDate *time.Time,
	limit, offset int,
) ([]model.ClimbRecord, int64, error) {
	if !t.IsDistance() {
		return nil, 0, fmt.Errorf("climb ranking is only supported for distance workout types: %s", t)
	}

	var workouts []*model.Workout
	q := model.ScopeExcludeRecordWorkouts(model.ScopeVisibleWorkouts(
		model.PreloadWorkoutData(s.db),
		targetUser.Profile.ID,
		viewer.Profile.ID,
	)).Where("workouts.type = ?", t)

	if startDate != nil {
		q = q.Where("workouts.date >= ?", *startDate)
	}

	if endDate != nil {
		q = q.Where("workouts.date <= ?", *endDate)
	}

	if err := q.Find(&workouts).Error; err != nil {
		return nil, 0, err
	}

	records := make([]model.ClimbRecord, 0)
	for _, workout := range workouts {
		if workout == nil || workout.Data == nil || workout.ExcludeFromRecords() {
			continue
		}

		for _, climb := range workout.Climbs {
			if climb.Type != "climb" {
				continue
			}

			records = append(records, model.ClimbRecord{
				ElevationGain: climb.Gain,
				Distance:      climb.Length,
				AverageSlope:  climb.AvgSlope,
				WorkoutID:     workout.ID,
				Date:          workout.Date,
				StartIndex:    climb.StartIdx,
				EndIndex:      climb.EndIdx,
				Active:        true,
			})
		}
	}

	sort.SliceStable(records, func(i, j int) bool {
		a := records[i]
		b := records[j]

		if a.ElevationGain != b.ElevationGain {
			return a.ElevationGain > b.ElevationGain
		}

		if a.Distance != b.Distance {
			return a.Distance > b.Distance
		}

		if a.Date.Equal(b.Date) {
			return false
		}

		return a.Date.Before(b.Date)
	})

	totalCount := int64(len(records))
	start := offset
	if start > len(records) {
		start = len(records)
	}

	end := start + limit
	if limit <= 0 || end > len(records) {
		end = len(records)
	}

	return records[start:end], totalCount, nil
}

func (s *userService) buildLocalProfileSummary(
	viewer, targetUser *model.User,
	requestHost, fallbackScheme string,
) (dto.ActivityPubProfileSummaryResponse, error) {
	postsQuery := model.ScopeVisibleWorkouts(
		s.db.Model(&model.Workout{}),
		targetUser.Profile.ID,
		viewer.Profile.ID,
	)

	var postsCount int64
	if err := postsQuery.Count(&postsCount).Error; err != nil {
		return dto.ActivityPubProfileSummaryResponse{}, err
	}

	followersCount, err := s.followerRepo.CountApprovedFollowers(targetUser.Profile.ID)
	if err != nil {
		return dto.ActivityPubProfileSummaryResponse{}, err
	}

	followingCount, err := s.followerRepo.CountApprovedFollowing(targetUser.Profile.ID)
	if err != nil {
		return dto.ActivityPubProfileSummaryResponse{}, err
	}

	isFollowing := false
	if viewer.ID != targetUser.ID {
		isFollowing, err = s.followerRepo.IsFollowingApproved(viewer.Profile.ID, targetUser.Profile.ID)
		if err != nil {
			return dto.ActivityPubProfileSummaryResponse{}, err
		}
	}

	return dto.ActivityPubProfileSummaryResponse{
		ID:             targetUser.ID,
		Username:       targetUser.Profile.Username,
		Name:           targetUser.Profile.DisplayName,
		Handle:         s.renderHandle(requestHost, targetUser.Profile.Username),
		ActorURL:       s.localActorIRI(requestHost, fallbackScheme, targetUser),
		IconURL:        "",
		IsExternal:     false,
		IsOwn:          viewer.ID == targetUser.ID,
		IsFollowing:    isFollowing,
		PostsCount:     postsCount,
		FollowersCount: followersCount,
		FollowingCount: followingCount,
		MemberSince:    targetUser.CreatedAt.UTC(),
	}, nil
}

func (s *userService) buildRemoteProfileSummary(
	ctx context.Context,
	viewer *model.User,
	username, host string,
) (dto.ActivityPubProfileSummaryResponse, error) {
	actorIRI, err := aputil.ResolveActorIRIFromWebFinger(ctx, username, host)
	if err != nil {
		return dto.ActivityPubProfileSummaryResponse{}, err
	}

	actor, err := aputil.LoadRemoteActor(ctx, actorIRI)
	if err != nil {
		return dto.ActivityPubProfileSummaryResponse{}, err
	}

	actorURL := actorIRI
	if actor != nil && actor.ID != "" {
		actorURL = actor.ID.String()
	}

	remoteProfile, err := s.apProfileSvc.GetByActorIRI(ctx, actorURL)
	if err != nil {
		return dto.ActivityPubProfileSummaryResponse{}, err
	}

	followersCount, _ := aputil.LoadCollectionTotalItems(ctx, itemIRIString(actor.Followers))
	followingCount, _ := aputil.LoadCollectionTotalItems(ctx, itemIRIString(actor.Following))
	postsCount, _ := aputil.LoadCollectionTotalItems(ctx, itemIRIString(actor.Outbox))

	name := username
	if actor != nil && actor.Name.String() != "" {
		name = actor.Name.String()
	}

	memberSince := time.Time{}
	if actor != nil {
		memberSince = actor.Published
	}

	iconURL := actorIconURL(actor)
	if iconURL == "" && remoteProfile.AvatarRemoteURL != nil {
		iconURL = *remoteProfile.AvatarRemoteURL
	}

	isFollowing := false
	if viewer != nil && !viewer.IsAnonymous() {
		isFollowing, err = s.followerRepo.IsFollowingActive(viewer.Profile.ID, remoteProfile.ID)
		if err != nil {
			return dto.ActivityPubProfileSummaryResponse{}, err
		}
	}

	return dto.ActivityPubProfileSummaryResponse{
		ID:             0,
		Username:       username,
		Name:           name,
		Handle:         fmt.Sprintf("@%s@%s", username, host),
		ActorURL:       actorURL,
		IconURL:        iconURL,
		IsExternal:     true,
		IsOwn:          false,
		IsFollowing:    isFollowing,
		PostsCount:     postsCount,
		FollowersCount: followersCount,
		FollowingCount: followingCount,
		MemberSince:    memberSince,
	}, nil
}

func (s *userService) SearchProfiles(ctx context.Context, viewer *model.User, query, requestHost, fallbackScheme string) ([]dto.ActivityPubProfileSummaryResponse, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []dto.ActivityPubProfileSummaryResponse{}, nil
	}

	results := make([]dto.ActivityPubProfileSummaryResponse, 0, 20)
	seenHandles := make(map[string]struct{})
	localQuery := query

	if username, host, parsedAsRemote, err := s.parseHandleWithHost(requestHost, query); err == nil {
		localQuery = username
		if parsedAsRemote {
			if summary, err := s.buildRemoteProfileSummary(ctx, viewer, username, host); err == nil {
				results, seenHandles = appendUniqueProfileSummary(results, seenHandles, summary)
			}
		} else if targetUser, err := s.userRepo.GetByHandle(query, s.LocalHost(requestHost)); err == nil {
			if viewer.ID != targetUser.ID && targetUser.ActivityPubEnabled() {
				summary, err := s.buildLocalProfileSummary(viewer, targetUser, requestHost, fallbackScheme)
				if err != nil {
					return nil, err
				}
				results, seenHandles = appendUniqueProfileSummary(results, seenHandles, summary)
			}
		}
	}

	localUsers, err := s.userRepo.SearchProfiles(localQuery)
	if err != nil {
		return nil, err
	}

	for _, targetUser := range localUsers {
		if viewer.ID == targetUser.ID {
			continue
		}
		summary, err := s.buildLocalProfileSummary(viewer, targetUser, requestHost, fallbackScheme)
		if err != nil {
			return nil, err
		}
		results, seenHandles = appendUniqueProfileSummary(results, seenHandles, summary)
	}

	sort.Slice(results, func(i, j int) bool {
		return strings.ToLower(results[i].Name) < strings.ToLower(results[j].Name)
	})

	return results, nil
}

func (s *userService) GetUserProfileByHandle(ctx context.Context, viewer *model.User, handle, requestHost, fallbackScheme string) (dto.ActivityPubProfileSummaryResponse, error) {
	handle = strings.TrimSpace(handle)
	if handle != "" {
		username, host, parsedAsRemote, err := s.parseHandleWithHost(requestHost, handle)
		if err != nil {
			return dto.ActivityPubProfileSummaryResponse{}, err
		}
		if parsedAsRemote {
			return s.buildRemoteProfileSummary(ctx, viewer, username, host)
		}
	}

	targetUser := viewer
	if handle != "" {
		var err error
		targetUser, err = s.userRepo.GetByHandle(handle, s.LocalHost(requestHost))
		if err != nil {
			return dto.ActivityPubProfileSummaryResponse{}, err
		}
	} else if viewer == nil || viewer.IsAnonymous() {
		return dto.ActivityPubProfileSummaryResponse{}, dto.ErrNotAuthorized
	}

	if viewer.ID != targetUser.ID && !targetUser.ActivityPubEnabled() {
		return dto.ActivityPubProfileSummaryResponse{}, gorm.ErrRecordNotFound
	}

	return s.buildLocalProfileSummary(viewer, targetUser, requestHost, fallbackScheme)
}

func (s *userService) FollowUserByHandle(ctx context.Context, viewer *model.User, handle, requestHost, fallbackScheme string) (dto.ActivityPubProfileSummaryResponse, error) {
	if viewer == nil || viewer.IsAnonymous() {
		return dto.ActivityPubProfileSummaryResponse{}, dto.ErrNotAuthorized
	}
	if !viewer.ActivityPubEnabled() {
		return dto.ActivityPubProfileSummaryResponse{}, ErrActivityPubRequiredToFollow
	}

	handle = strings.TrimSpace(handle)
	if handle != "" {
		if username, host, parsedAsRemote, err := s.parseHandleWithHost(requestHost, handle); err == nil && parsedAsRemote {
			return s.followRemoteUserByHandle(ctx, viewer, username, host, requestHost, fallbackScheme)
		}
	}

	targetUser, err := s.ResolveTargetUser(handle, requestHost, viewer)
	if err != nil {
		return dto.ActivityPubProfileSummaryResponse{}, err
	}

	if viewer.ID == targetUser.ID {
		return dto.ActivityPubProfileSummaryResponse{}, ErrCannotFollowSelf
	}
	if !targetUser.ActivityPubEnabled() {
		return dto.ActivityPubProfileSummaryResponse{}, ErrTargetNoActivityPub
	}

	following, err := s.followerRepo.UpsertFollowingRequest(viewer.Profile.ID, &targetUser.Profile)
	if err != nil {
		return dto.ActivityPubProfileSummaryResponse{}, err
	}

	followersCount, err := s.followerRepo.CountApprovedFollowers(targetUser.Profile.ID)
	if err != nil {
		return dto.ActivityPubProfileSummaryResponse{}, err
	}
	followingCount, err := s.followerRepo.CountApprovedFollowing(targetUser.Profile.ID)
	if err != nil {
		return dto.ActivityPubProfileSummaryResponse{}, err
	}

	summary := dto.ActivityPubProfileSummaryResponse{
		ID:             targetUser.ID,
		Username:       targetUser.Profile.Username,
		Name:           targetUser.Profile.DisplayName,
		Handle:         s.renderHandle(requestHost, targetUser.Profile.Username),
		ActorURL:       s.localActorIRI(requestHost, fallbackScheme, targetUser),
		IconURL:        "",
		IsExternal:     false,
		IsOwn:          false,
		IsFollowing:    following.Approved,
		FollowersCount: followersCount,
		FollowingCount: followingCount,
		MemberSince:    targetUser.CreatedAt,
	}

	if err := s.notify.Send(ctx, targetUser, notification.NewFollowRequest(viewer.Profile.DisplayName)); err != nil {
		s.logger.Error("Failed to send notification", "error", err)
	}

	return summary, nil
}

func (s *userService) followRemoteUserByHandle(ctx context.Context, viewer *model.User, username, host, requestHost, fallbackScheme string) (dto.ActivityPubProfileSummaryResponse, error) {
	actorIRI, err := aputil.ResolveActorIRIFromWebFinger(ctx, username, host)
	if err != nil {
		return dto.ActivityPubProfileSummaryResponse{}, err
	}

	actor, err := aputil.LoadRemoteActor(ctx, actorIRI)
	if err != nil {
		return dto.ActivityPubProfileSummaryResponse{}, err
	}

	inbox := itemIRIString(actor.Inbox)
	if inbox == "" {
		return dto.ActivityPubProfileSummaryResponse{}, ErrRemoteInboxNotFound
	}

	viewerActorIRI := s.localActorIRI(requestHost, fallbackScheme, viewer)
	if actorIRI == viewerActorIRI {
		return dto.ActivityPubProfileSummaryResponse{}, ErrCannotFollowSelf
	}

	remoteProfile, err := s.apProfileSvc.GetByActorIRI(ctx, actorIRI)
	if err != nil {
		return dto.ActivityPubProfileSummaryResponse{}, err
	}

	if _, err := s.followerRepo.UpsertFollowingRequest(viewer.Profile.ID, remoteProfile); err != nil {
		return dto.ActivityPubProfileSummaryResponse{}, err
	}

	if err := s.actorService.SendFollow(ctx, &viewer.Profile, inbox, actorIRI); err != nil {
		return dto.ActivityPubProfileSummaryResponse{}, err
	}

	followersCount, _ := aputil.LoadCollectionTotalItems(ctx, itemIRIString(actor.Followers))
	followingCount, _ := aputil.LoadCollectionTotalItems(ctx, itemIRIString(actor.Following))
	postsCount, _ := aputil.LoadCollectionTotalItems(ctx, itemIRIString(actor.Outbox))

	name := username
	if actor != nil && actor.Name.String() != "" {
		name = actor.Name.String()
	}

	memberSince := time.Time{}
	if actor != nil {
		memberSince = actor.Published
	}

	return dto.ActivityPubProfileSummaryResponse{
		ID:             0,
		Username:       username,
		Name:           name,
		Handle:         fmt.Sprintf("@%s@%s", username, host),
		ActorURL:       actorIRI,
		IconURL:        actorIconURL(actor),
		IsExternal:     true,
		IsOwn:          false,
		IsFollowing:    true,
		PostsCount:     postsCount,
		FollowersCount: followersCount,
		FollowingCount: followingCount,
		MemberSince:    memberSince,
	}, nil
}

func (s *userService) UnfollowUserByHandle(ctx context.Context, viewer *model.User, handle, requestHost, fallbackScheme string) (dto.ActivityPubProfileSummaryResponse, error) {
	if viewer == nil || viewer.IsAnonymous() {
		return dto.ActivityPubProfileSummaryResponse{}, dto.ErrNotAuthorized
	}
	if !viewer.ActivityPubEnabled() {
		return dto.ActivityPubProfileSummaryResponse{}, ErrActivityPubRequiredToFollow
	}

	handle = strings.TrimSpace(handle)
	if handle != "" {
		if username, host, parsedAsRemote, err := s.parseHandleWithHost(requestHost, handle); err == nil && parsedAsRemote {
			return s.unfollowRemoteUserByHandle(ctx, viewer, username, host)
		}
	}

	targetUser, err := s.ResolveTargetUser(handle, requestHost, viewer)
	if err != nil {
		return dto.ActivityPubProfileSummaryResponse{}, err
	}

	if viewer.ID == targetUser.ID {
		return dto.ActivityPubProfileSummaryResponse{}, ErrCannotUnfollowSelf
	}

	targetActorIRI := s.localActorIRI(requestHost, fallbackScheme, targetUser)
	if err := s.followerRepo.DeleteFollowing(viewer.Profile.ID, targetUser.Profile.ID); err != nil {
		return dto.ActivityPubProfileSummaryResponse{}, err
	}
	followersCount, err := s.followerRepo.CountApprovedFollowers(targetUser.Profile.ID)
	if err != nil {
		return dto.ActivityPubProfileSummaryResponse{}, err
	}
	followingCount, err := s.followerRepo.CountApprovedFollowing(targetUser.Profile.ID)
	if err != nil {
		return dto.ActivityPubProfileSummaryResponse{}, err
	}

	return dto.ActivityPubProfileSummaryResponse{
		ID:             targetUser.ID,
		Username:       targetUser.Profile.Username,
		Name:           targetUser.Profile.DisplayName,
		Handle:         s.renderHandle(requestHost, targetUser.Profile.Username),
		ActorURL:       targetActorIRI,
		IconURL:        "",
		IsExternal:     false,
		IsOwn:          false,
		IsFollowing:    false,
		FollowersCount: followersCount,
		FollowingCount: followingCount,
		MemberSince:    targetUser.CreatedAt,
	}, nil
}

func (s *userService) unfollowRemoteUserByHandle(ctx context.Context, viewer *model.User, username, host string) (dto.ActivityPubProfileSummaryResponse, error) {
	actorIRI, err := aputil.ResolveActorIRIFromWebFinger(ctx, username, host)
	if err != nil {
		return dto.ActivityPubProfileSummaryResponse{}, err
	}

	actor, err := aputil.LoadRemoteActor(ctx, actorIRI)
	if err != nil {
		return dto.ActivityPubProfileSummaryResponse{}, err
	}

	inbox := itemIRIString(actor.Inbox)
	if inbox == "" {
		return dto.ActivityPubProfileSummaryResponse{}, ErrRemoteInboxNotFound
	}

	if err := s.actorService.SendUndoFollow(ctx, &viewer.Profile, inbox, actorIRI); err != nil {
		return dto.ActivityPubProfileSummaryResponse{}, err
	}

	remoteProfile, err := s.apProfileSvc.GetByActorIRI(ctx, actorIRI)
	if err != nil {
		return dto.ActivityPubProfileSummaryResponse{}, err
	}

	if err := s.followerRepo.DeleteFollowing(viewer.Profile.ID, remoteProfile.ID); err != nil {
		return dto.ActivityPubProfileSummaryResponse{}, err
	}

	followersCount, _ := aputil.LoadCollectionTotalItems(ctx, itemIRIString(actor.Followers))
	followingCount, _ := aputil.LoadCollectionTotalItems(ctx, itemIRIString(actor.Following))
	postsCount, _ := aputil.LoadCollectionTotalItems(ctx, itemIRIString(actor.Outbox))

	name := username
	if actor != nil && actor.Name.String() != "" {
		name = actor.Name.String()
	}

	memberSince := time.Time{}
	if actor != nil {
		memberSince = actor.Published
	}

	return dto.ActivityPubProfileSummaryResponse{
		ID:             0,
		Username:       username,
		Name:           name,
		Handle:         fmt.Sprintf("@%s@%s", username, host),
		ActorURL:       actorIRI,
		IconURL:        actorIconURL(actor),
		IsExternal:     true,
		IsOwn:          false,
		IsFollowing:    false,
		PostsCount:     postsCount,
		FollowersCount: followersCount,
		FollowingCount: followingCount,
		MemberSince:    memberSince,
	}, nil
}

func itemIRIString(it vocab.Item) string {
	if vocab.IsNil(it) {
		return ""
	}
	if vocab.IsIRI(it) {
		return it.GetLink().String()
	}
	var iri string
	_ = vocab.OnLink(it, func(link *vocab.Link) error {
		iri = link.Href.String()
		return nil
	})
	return iri
}

func actorIconURL(actor *vocab.Actor) string {
	if actor == nil || vocab.IsNil(actor.Icon) {
		return ""
	}
	if vocab.IsIRI(actor.Icon) {
		return actor.Icon.GetLink().String()
	}
	iconURL := itemIRIString(actor.Icon)
	if iconURL != "" {
		return iconURL
	}
	_ = vocab.OnObject(actor.Icon, func(object *vocab.Object) error {
		if object != nil && !vocab.IsNil(object.URL) {
			iconURL = itemIRIString(object.URL)
		}
		return nil
	})
	return iconURL
}

func appendUniqueProfileSummary(
	results []dto.ActivityPubProfileSummaryResponse,
	seenHandles map[string]struct{},
	summary dto.ActivityPubProfileSummaryResponse,
) ([]dto.ActivityPubProfileSummaryResponse, map[string]struct{}) {
	if _, exists := seenHandles[summary.Handle]; exists {
		return results, seenHandles
	}
	seenHandles[summary.Handle] = struct{}{}
	return append(results, summary), seenHandles
}
