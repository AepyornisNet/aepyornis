package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/aputil"
	"github.com/AepyornisNet/aepyornis/pkg/config"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/model/dto"
	"github.com/AepyornisNet/aepyornis/pkg/notification"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/samber/do/v2"
	"github.com/vgarvardt/gue/v6"
	"gorm.io/gorm"
)

type UploadedWorkoutFile struct {
	Filename string
	Content  []byte
}

type WorkoutService interface {
	GetWorkouts(ctx context.Context, viewer *model.User, filters *model.WorkoutFilters, limit, offset int) ([]*model.Workout, int64, error)
	GetWorkoutDetails(ctx context.Context, viewer *model.User, workoutID uint64) (*model.Workout, []model.WorkoutIntervalRecordWithRank, []model.WorkoutIntervalRecordWithRank, error)
	CanReadWorkout(viewer *model.User, workout *model.Workout) (bool, error)
	GetOwnedWorkout(viewer *model.User, workoutID uint64) (*model.Workout, error)
	GetReadableWorkout(viewer *model.User, workoutID uint64, withDetails bool) (*model.Workout, error)
	CreateWorkoutsFromFile(ctx context.Context, viewer *model.User, files []UploadedWorkoutFile, notes string, workoutType model.WorkoutType, visibility *model.WorkoutVisibility, names []string) ([]*model.Workout, []error)
	CreateWorkoutManual(ctx context.Context, viewer *model.User, req *dto.ManualWorkout) (*model.Workout, error)
	UpdateWorkout(ctx context.Context, viewer *model.User, workout *model.Workout, req *dto.ManualWorkout) (*model.Workout, error)
	DeleteWorkout(ctx context.Context, viewer *model.User, workout *model.Workout) error
	ToggleWorkoutLock(ctx context.Context, viewer *model.User, workout *model.Workout) error
	RefreshWorkout(ctx context.Context, workout *model.Workout) error
	GetWorkoutBreakdown(ctx context.Context, workout *model.Workout, preferredUnits string, params dto.WorkoutBreakdownRequest) (dto.WorkoutBreakdownResponse, error)
	GetWorkoutRangeStats(workout *model.Workout, startIndex, endIndex *int) (*model.MapDataRangeStats, int, int, error)
	GetWorkoutCalendar(ctx context.Context, targetUser, viewer *model.User, start, end, timezone *string) ([]dto.CalendarEventResponse, error)
	ExportWorkoutsZip(ctx context.Context, profileID uint64, ids []uint64) ([]byte, error)
	AddEquipmentToWorkouts(ctx context.Context, profileID uint64, workoutIDs, equipmentIDs []uint64) error
	GetWorkoutFilterOptions(ctx context.Context, profileID uint64) ([]string, map[string][]string, error)
	GetWorkoutLikes(ctx context.Context, workoutID uint64) ([]dto.WorkoutLikeResponse, error)
	GetWorkoutReplies(ctx context.Context, workoutID uint64, page, perPage int) ([]dto.WorkoutReplyResponse, int64, error)
	LikeWorkout(ctx context.Context, viewer *model.User, workout *model.Workout) (map[string]any, error)
	LikeWorkoutByObject(ctx context.Context, viewer *model.User, objectID string, localActorIRI string) (map[string]any, int, error)
	CreateReply(ctx context.Context, viewer *model.User, workout *model.Workout, content string) (*dto.WorkoutReplyResponse, error)
	GetRecentWorkouts(ctx context.Context, requester *model.User, req dto.RecentWorkoutsRequest, host string) ([]*model.Workout, error)
	EnrichWorkoutsMetadata(ctx context.Context, viewer *model.User, workouts []*model.Workout) ([]dto.WorkoutResponse, error)
}

type workoutService struct {
	apOutboxRepo     repository.APOutbox
	apProfileService ActivityPubProfileService
	actorService     ActivityPubActorService
	cfg              *config.Config
	client           *gue.Client
	db               *gorm.DB
	equipmentRepo    repository.Equipment
	logger           *slog.Logger
	notify           NotificationService
	userRepo         repository.User
	workoutLikeRepo  repository.WorkoutLike
	workoutReplyRepo repository.WorkoutReply
	workoutRepo      repository.Workout
}

func NewWorkoutService(injector do.Injector) (WorkoutService, error) {
	return &workoutService{
		apOutboxRepo:     do.MustInvoke[repository.APOutbox](injector),
		apProfileService: do.MustInvoke[ActivityPubProfileService](injector),
		actorService:     do.MustInvoke[ActivityPubActorService](injector),
		cfg:              do.MustInvoke[*config.Config](injector),
		client:           do.MustInvoke[*gue.Client](injector),
		db:               do.MustInvoke[*gorm.DB](injector),
		equipmentRepo:    do.MustInvoke[repository.Equipment](injector),
		logger:           do.MustInvoke[*slog.Logger](injector),
		notify:           do.MustInvoke[NotificationService](injector),
		userRepo:         do.MustInvoke[repository.User](injector),
		workoutLikeRepo:  do.MustInvoke[repository.WorkoutLike](injector),
		workoutReplyRepo: do.MustInvoke[repository.WorkoutReply](injector),
		workoutRepo:      do.MustInvoke[repository.Workout](injector),
	}, nil
}

func (s *workoutService) CanReadWorkout(requester *model.User, workout *model.Workout) (bool, error) {
	if workout == nil {
		return false, gorm.ErrRecordNotFound
	}
	return model.CanReadWorkout(s.db, requester, workout)
}

func (s *workoutService) GetOwnedWorkout(viewer *model.User, workoutID uint64) (*model.Workout, error) {
	workout, err := s.workoutRepo.GetByProfileID(viewer.Profile.ID, workoutID)
	if err != nil {
		return nil, err
	}
	workout.Profile = &viewer.Profile
	return workout, nil
}

func (s *workoutService) GetReadableWorkout(viewer *model.User, workoutID uint64, withDetails bool) (*model.Workout, error) {
	var workout *model.Workout
	var err error
	if withDetails {
		workout, err = s.workoutRepo.GetDetailsByID(workoutID)
	} else {
		workout, err = s.workoutRepo.GetByIDForRead(workoutID, true)
	}
	if err != nil {
		return nil, err
	}

	canRead, err := s.CanReadWorkout(viewer, workout)
	if err != nil {
		return nil, err
	}
	if !canRead {
		return nil, gorm.ErrRecordNotFound
	}
	return workout, nil
}

func (s *workoutService) GetWorkouts(_ context.Context, viewer *model.User, filters *model.WorkoutFilters, limit, offset int) ([]*model.Workout, int64, error) {
	totalCount, err := s.workoutRepo.CountByProfileAndFilters(viewer.Profile.ID, filters)
	if err != nil {
		return nil, 0, err
	}

	workouts, err := s.workoutRepo.ListByProfileAndFilters(viewer.Profile.ID, filters, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return workouts, totalCount, nil
}

func (s *workoutService) GetRecentWorkouts(_ context.Context, requester *model.User, req dto.RecentWorkoutsRequest, host string) ([]*model.Workout, error) {
	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}
	handle := strings.TrimSpace(req.Handle)

	query := s.db.
		Scopes(model.PreloadWorkoutData).
		Preload("Profile").
		Preload("Profile.User")

	if handle != "" {
		cfgHost := s.cfg.Host
		if cfgHost == "" {
			cfgHost = host
		}
		targetUser, err := s.userRepo.GetByHandle(handle, cfgHost)
		if err != nil {
			return nil, err
		}
		query = model.ScopeVisibleWorkouts(query, targetUser.Profile.ID, requester.Profile.ID)
	} else {
		scope := req.Scope
		if scope == "" {
			scope = "following"
		}

		if scope == "global" {
			query = query.Where(
				`workouts.profile_id = ? OR workouts.visibility = ? OR (
					workouts.visibility = ? AND
					EXISTS (
						SELECT 1
						FROM followers f
						WHERE f.profile_id = ?
							AND f.following_profile_id = workouts.profile_id
							AND f.approved = ?
					)
				)`,
				requester.Profile.ID,
				model.WorkoutVisibilityPublic,
				model.WorkoutVisibilityFollowers,
				requester.Profile.ID,
				true,
			)
		} else {
			query = query.Where(
				`workouts.profile_id = ? OR (
					(workouts.visibility = ? OR workouts.visibility = ?) AND
					EXISTS (
						SELECT 1
						FROM followers f
						WHERE f.profile_id = ?
							AND f.following_profile_id = workouts.profile_id
							AND f.approved = ?
					)
				)`,
				requester.Profile.ID,
				model.WorkoutVisibilityPublic,
				model.WorkoutVisibilityFollowers,
				requester.Profile.ID,
				true,
			)
		}
	}

	var workouts []*model.Workout
	err := query.
		Order("date DESC").
		Limit(limit).
		Offset(offset).
		Find(&workouts).Error
	if err != nil {
		return nil, err
	}
	return workouts, nil
}

func (s *workoutService) GetWorkoutDetails(ctx context.Context, viewer *model.User, workoutID uint64) (*model.Workout, []model.WorkoutIntervalRecordWithRank, []model.WorkoutIntervalRecordWithRank, error) {
	workout, err := s.GetReadableWorkout(viewer, workoutID, true)
	if err != nil {
		return nil, nil, nil, err
	}

	if len(workout.RouteSegmentMatches) > 0 {
		visibleMatches := make([]*model.RouteSegmentMatch, 0, len(workout.RouteSegmentMatches))
		for _, m := range workout.RouteSegmentMatches {
			if m != nil && m.RouteSegment != nil {
				canRead, _ := model.CanReadRouteSegment(s.db, viewer, m.RouteSegment)
				if canRead {
					visibleMatches = append(visibleMatches, m)
				}
			}
		}
		workout.RouteSegmentMatches = visibleMatches
	}

	records, _ := model.GetWorkoutIntervalRecordsWithRank(s.db, workout.ProfileID, workout.Type, workout.ID)
	powerRecords, _ := model.GetWorkoutPowerIntervalRecordsWithRank(s.db, workout.ProfileID, workout.Type, workout.ID)

	return workout, records, powerRecords, nil
}

func (s *workoutService) CreateWorkoutsFromFile(
	ctx context.Context,
	viewer *model.User,
	files []UploadedWorkoutFile,
	notes string,
	workoutType model.WorkoutType,
	visibility *model.WorkoutVisibility,
	names []string,
) ([]*model.Workout, []error) {
	if workoutType == "" {
		workoutType = model.WorkoutTypeAutoDetect
	}

	createdWorkouts := make([]*model.Workout, 0, len(files))
	errList := make([]error, 0)

	for i, file := range files {
		viewer.Profile.User = viewer
		ws, addErr := viewer.Profile.AddWorkout(s.db, workoutType, notes, file.Filename, file.Content)
		if len(addErr) > 0 {
			errList = append(errList, addErr...)
			continue
		}

		var customName string
		if i < len(names) {
			customName = strings.TrimSpace(names[i])
		}
		if customName == "" && len(names) == 1 && len(files) == 1 {
			customName = strings.TrimSpace(names[0])
		}

		for _, w := range ws {
			needsSave := false
			if customName != "" {
				w.Name = customName
				needsSave = true
			}
			if visibility != nil && visibility.IsValid() {
				w.Visibility = *visibility
				needsSave = true
			}
			if needsSave {
				_ = w.Save(s.db)
			}
			createdWorkouts = append(createdWorkouts, w)

			if err := s.enqueueWorkoutUpdate(ctx, w.ID); err != nil {
				s.logger.Error("Failed to enqueue workout update", "workout_id", w.ID, "error", err)
			}
		}
	}

	return createdWorkouts, errList
}

func (s *workoutService) CreateWorkoutManual(ctx context.Context, viewer *model.User, req *dto.ManualWorkout) (*model.Workout, error) {
	workout := &model.Workout{}
	if err := req.Update(workout); err != nil {
		return nil, err
	}
	if req.Visibility == nil {
		workout.Visibility = viewer.EffectiveDefaultWorkoutVisibility()
	}

	workout.Profile = &viewer.Profile
	workout.ProfileID = viewer.Profile.ID
	workout.Creator = "web-interface"

	equipment, err := s.equipmentRepo.GetByUserIDs(viewer.Profile.ID, req.EquipmentIDs)
	if err != nil {
		return nil, err
	}

	if err := workout.Save(s.db); err != nil {
		return nil, err
	}

	if err := s.db.Model(&workout).Association("Equipment").Replace(equipment); err != nil {
		return nil, err
	}

	if err := model.PreloadWorkoutDetails(s.db).Preload("Equipment").First(&workout, workout.ID).Error; err != nil {
		return nil, err
	}

	if err := s.enqueueWorkoutUpdate(ctx, workout.ID); err != nil {
		s.logger.Error("Failed to enqueue workout update", "workout_id", workout.ID, "error", err)
	}

	return workout, nil
}

func (s *workoutService) UpdateWorkout(ctx context.Context, viewer *model.User, workout *model.Workout, req *dto.ManualWorkout) (*model.Workout, error) {
	if err := req.Update(workout); err != nil {
		return nil, err
	}

	if req.EquipmentIDs != nil {
		equipment, err := s.equipmentRepo.GetByUserIDs(viewer.ID, req.EquipmentIDs)
		if err != nil {
			return nil, err
		}
		if err := s.db.Model(&workout).Association("Equipment").Replace(equipment); err != nil {
			return nil, err
		}
	}

	if err := workout.Save(s.db); err != nil {
		return nil, err
	}

	if err := model.PreloadWorkoutDetails(s.db).Preload("Equipment").First(&workout, workout.ID).Error; err != nil {
		return nil, err
	}

	if err := s.enqueueWorkoutUpdate(ctx, workout.ID); err != nil {
		s.logger.Error("Failed to enqueue workout update", "workout_id", workout.ID, "error", err)
	}

	return workout, nil
}

func (s *workoutService) DeleteWorkout(_ context.Context, viewer *model.User, workout *model.Workout) error {
	if err := s.apOutboxRepo.DeleteEntryForWorkout(viewer.ID, workout.ID); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return workout.Delete(s.db)
}

func (s *workoutService) ToggleWorkoutLock(_ context.Context, viewer *model.User, workout *model.Workout) error {
	if workout.ProfileID != viewer.Profile.ID {
		return errors.New("not authorized")
	}
	workout.Locked = !workout.Locked
	return workout.Save(s.db)
}

func (s *workoutService) RefreshWorkout(ctx context.Context, workout *model.Workout) error {
	workout.Dirty = true
	if err := workout.Save(s.db); err != nil {
		return err
	}
	return s.enqueueWorkoutUpdate(ctx, workout.ID)
}

func (s *workoutService) GetWorkoutBreakdown(_ context.Context, workout *model.Workout, preferredUnits string, params dto.WorkoutBreakdownRequest) (dto.WorkoutBreakdownResponse, error) {
	preferLaps := (params.Mode == "" || params.Mode == "auto" || params.Mode == "laps") && len(workout.Laps) > 1

	if preferLaps {
		return dto.WorkoutBreakdownResponse{
			Mode:  "laps",
			Items: dto.NewWorkoutBreakdownItemsFromLaps(workout.Laps, workout.Records),
		}, nil
	}

	if len(workout.Records) == 0 {
		return dto.WorkoutBreakdownResponse{}, errors.New("workout has no records")
	}

	breakdown, err := workout.StatisticsPer(params.Count, preferredUnits)
	if err != nil {
		return dto.WorkoutBreakdownResponse{}, err
	}

	return dto.WorkoutBreakdownResponse{
		Mode:  "unit",
		Items: dto.NewWorkoutBreakdownItemsFromUnit(breakdown.Items, params.Count),
	}, nil
}

func (s *workoutService) GetWorkoutRangeStats(workout *model.Workout, startIndex, endIndex *int) (*model.MapDataRangeStats, int, int, error) {
	if len(workout.Records) == 0 {
		return nil, 0, 0, errors.New("workout has no records")
	}

	points := workout.Records
	startIdx := 0
	endIdx := len(points) - 1

	if startIndex != nil {
		startIdx = *startIndex
	}
	if endIndex != nil {
		endIdx = *endIndex
	}

	if startIdx < 0 || endIdx >= len(points) || startIdx > endIdx {
		return nil, 0, 0, errors.New("invalid range")
	}

	stats, ok := model.StatsForRange(workout.Records, startIdx, endIdx)
	if !ok {
		return nil, 0, 0, errors.New("invalid range")
	}

	return &stats, startIdx, endIdx, nil
}

func (s *workoutService) GetWorkoutCalendar(_ context.Context, targetUser, viewer *model.User, start, end, timezone *string) ([]dto.CalendarEventResponse, error) {
	tz := time.UTC
	if timezone != nil {
		location, err := time.LoadLocation(*timezone)
		if err == nil {
			tz = location
		}
	}

	db := model.ScopeVisibleWorkouts(
		model.PreloadWorkoutData(s.db),
		targetUser.Profile.ID,
		viewer.Profile.ID,
	)

	const calTS = "2006-01-02T15:04:05"
	if start != nil {
		if startDate, err := time.ParseInLocation(calTS, *start, tz); err == nil {
			db = db.Where("workouts.date >= ?", startDate)
		}
	}
	if end != nil {
		if endDate, err := time.ParseInLocation(calTS, *end, tz); err == nil {
			db = db.Where("workouts.date <= ?", endDate)
		}
	}

	var workouts []*model.Workout
	if err := db.Find(&workouts).Error; err != nil {
		return nil, err
	}

	events := make([]dto.CalendarEventResponse, len(workouts))
	for i, w := range workouts {
		title := w.Name
		if title == "" {
			title = string(w.Type)
		}
		if w.TotalDistance > 0 {
			title += " - " + formatDistance(w.TotalDistance)
		}
		if w.TotalDuration.Seconds() > 0 {
			title += " " + formatDuration(int64(w.TotalDuration.Seconds()))
		}
		events[i] = dto.CalendarEventResponse{
			Title: title,
			Start: w.GetDate().In(tz),
			End:   w.GetEnd().In(tz),
			URL:   "/workouts/" + strconv.FormatUint(w.ID, 10),
		}
	}
	return events, nil
}

func (s *workoutService) ExportWorkoutsZip(_ context.Context, profileID uint64, ids []uint64) ([]byte, error) {
	var workouts []model.Workout
	if err := s.db.Preload("File").Where("profile_id = ?", profileID).Find(&workouts, ids).Error; err != nil {
		return nil, err
	}
	if len(workouts) == 0 {
		return nil, errors.New("no workouts found")
	}

	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	for _, w := range workouts {
		if !w.HasFile() {
			continue
		}
		filename := w.File.Filename
		if filename == "" {
			filename = "workout_" + strconv.FormatUint(w.ID, 10) + ".gpx"
		}
		f, err := zipWriter.Create(filename)
		if err != nil {
			return nil, err
		}
		if _, err := f.Write(w.File.Content); err != nil {
			return nil, err
		}
	}
	if err := zipWriter.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *workoutService) AddEquipmentToWorkouts(_ context.Context, profileID uint64, workoutIDs, equipmentIDs []uint64) error {
	equipment, err := s.equipmentRepo.GetByUserIDs(profileID, equipmentIDs)
	if err != nil {
		return err
	}
	if len(equipment) == 0 {
		return errors.New("no valid equipment found")
	}

	var workouts []model.Workout
	if err := s.db.Where("profile_id = ?", profileID).Find(&workouts, workoutIDs).Error; err != nil {
		return err
	}

	tx := s.db.Begin()
	for _, w := range workouts {
		if err := tx.Model(&w).Association("Equipment").Append(equipment); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

func (s *workoutService) GetWorkoutFilterOptions(_ context.Context, profileID uint64) ([]string, map[string][]string, error) {
	var types []string
	if err := s.db.Model(&model.Workout{}).
		Where("profile_id = ?", profileID).
		Where("type IS NOT NULL AND type != ''").
		Order("type").
		Distinct("type").
		Pluck("type", &types).Error; err != nil {
		return nil, nil, err
	}

	type typeSubPair struct {
		Type    string `gorm:"column:type"`
		SubType string `gorm:"column:sub_type"`
	}
	var pairs []typeSubPair
	if err := s.db.Model(&model.Workout{}).
		Select("type, sub_type").
		Where("profile_id = ?", profileID).
		Where("type IS NOT NULL AND type != ''").
		Where("sub_type IS NOT NULL AND sub_type != ''").
		Group("type, sub_type").
		Order("type, sub_type").
		Find(&pairs).Error; err != nil {
		return nil, nil, err
	}

	subTypesByType := make(map[string][]string)
	for _, p := range pairs {
		subTypesByType[p.Type] = append(subTypesByType[p.Type], p.SubType)
	}

	return types, subTypesByType, nil
}

func (s *workoutService) GetWorkoutLikes(ctx context.Context, workoutID uint64) ([]dto.WorkoutLikeResponse, error) {
	likes, err := s.workoutLikeRepo.ListByWorkoutID(workoutID)
	if err != nil {
		return nil, err
	}

	results := make([]dto.WorkoutLikeResponse, 0, len(likes))
	for i := range likes {
		likeResponse := dto.NewWorkoutLikeResponse(&likes[i])
		if likes[i].Profile != nil && likes[i].Profile.UserID == nil {
			if actorIRI := strings.TrimSpace(likes[i].Profile.ActorURL()); actorIRI != "" {
				profile, err := s.apProfileService.GetByActorIRI(ctx, actorIRI)
				if err == nil && profile != nil {
					if name := strings.TrimSpace(profile.DisplayName); name != "" {
						likeResponse.ActorName = &name
					}
					if profile.AvatarRemoteURL != nil && strings.TrimSpace(*profile.AvatarRemoteURL) != "" {
						avatarURL := strings.TrimSpace(*profile.AvatarRemoteURL)
						likeResponse.AvatarURL = &avatarURL
					}
				}
			}
		}
		results = append(results, likeResponse)
	}
	return results, nil
}

func (s *workoutService) GetWorkoutReplies(ctx context.Context, workoutID uint64, page, perPage int) ([]dto.WorkoutReplyResponse, int64, error) {
	offset := (page - 1) * perPage
	totalCount, err := s.workoutReplyRepo.CountByWorkoutID(workoutID)
	if err != nil {
		return nil, 0, err
	}

	replies, err := s.workoutReplyRepo.ListByWorkoutID(workoutID, perPage, offset)
	if err != nil {
		return nil, 0, err
	}

	results := make([]dto.WorkoutReplyResponse, 0, len(replies))
	for i := range replies {
		replyResponse := dto.NewWorkoutReplyResponse(&replies[i])
		if replies[i].Profile != nil && replies[i].Profile.UserID == nil {
			if actorIRI := strings.TrimSpace(replies[i].Profile.ActorURL()); actorIRI != "" {
				profile, err := s.apProfileService.GetByActorIRI(ctx, actorIRI)
				if err == nil && profile != nil {
					if replyResponse.ActorName == nil {
						if name := strings.TrimSpace(profile.DisplayName); name != "" {
							replyResponse.ActorName = &name
						}
					}
					if profile.AvatarRemoteURL != nil && strings.TrimSpace(*profile.AvatarRemoteURL) != "" {
						avatarURL := strings.TrimSpace(*profile.AvatarRemoteURL)
						replyResponse.AvatarURL = &avatarURL
					}
				}
			}
		}
		results = append(results, replyResponse)
	}
	return results, totalCount, nil
}

func (s *workoutService) LikeWorkout(ctx context.Context, viewer *model.User, workout *model.Workout) (map[string]any, error) {
	if workout.ProfileID == viewer.Profile.ID {
		return nil, errors.New("cannot like your own workout")
	}

	if err := s.workoutLikeRepo.LikeByProfile(workout.ID, viewer.Profile.ID); err != nil {
		return nil, err
	}

	ownerID := workoutOwnerUserID(workout)
	if ownerID != 0 && ownerID != viewer.ID {
		var owner model.User
		if err := s.db.First(&owner, ownerID).Error; err == nil {
			_ = s.notify.Send(ctx, &owner, notification.NewWorkoutLike(viewer.Profile.DisplayName, workout.ID))
		}
	}

	counts, err := s.workoutLikeRepo.CountMapByWorkoutIDs([]uint64{workout.ID})
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"workout_id":  workout.ID,
		"likes_count": counts[workout.ID],
		"liked":       true,
	}, nil
}

func (s *workoutService) LikeWorkoutByObject(ctx context.Context, viewer *model.User, objectID string, localActorIRI string) (map[string]any, int, error) {
	localWorkoutID, localErr := s.apOutboxRepo.ResolveWorkoutIDByObjectOrActivityID(0, objectID)
	if localErr == nil {
		workout, err := s.GetReadableWorkout(viewer, localWorkoutID, false)
		if err != nil {
			return nil, http.StatusNotFound, err
		}
		res, err := s.LikeWorkout(ctx, viewer, workout)
		if err != nil {
			return nil, http.StatusBadRequest, err
		}
		return res, http.StatusOK, nil
	}

	if !errors.Is(localErr, gorm.ErrRecordNotFound) {
		return nil, http.StatusInternalServerError, localErr
	}

	if !viewer.ActivityPubEnabled() {
		return nil, http.StatusBadRequest, errors.New("activitypub must be enabled to like remote workouts")
	}

	actorIRI, inbox, err := aputil.ResolveObjectActorAndInbox(ctx, objectID)
	if err != nil {
		return nil, http.StatusNotFound, err
	}
	if inbox == "" {
		return nil, http.StatusBadRequest, errors.New("remote actor inbox not found")
	}
	if actorIRI == localActorIRI {
		return nil, http.StatusBadRequest, errors.New("cannot like your own workout")
	}

	if err := s.actorService.SendLike(ctx, &viewer.Profile, inbox, objectID); err != nil {
		return nil, http.StatusBadGateway, err
	}

	return map[string]any{
		"object_id": objectID,
		"liked":     true,
	}, http.StatusOK, nil
}

func (s *workoutService) likeLocalWorkout(ctx context.Context, viewer *model.User, localWorkoutID uint64) (map[string]any, int, error) {
	workout, err := s.workoutRepo.GetByIDForRead(localWorkoutID, false)
	if err != nil {
		return nil, http.StatusNotFound, err
	}

	allowed, err := s.CanReadWorkout(viewer, workout)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if !allowed {
		return nil, http.StatusNotFound, gorm.ErrRecordNotFound
	}

	if workoutOwnerUserID(workout) == viewer.ID {
		return nil, http.StatusBadRequest, errors.New("cannot like your own workout")
	}

	if err := s.workoutLikeRepo.LikeByProfile(localWorkoutID, viewer.Profile.ID); err != nil {
		return nil, http.StatusInternalServerError, err
	}

	if ownerID := workoutOwnerUserID(workout); ownerID != 0 && ownerID != viewer.ID {
		var owner model.User
		if err := s.db.First(&owner, ownerID).Error; err == nil {
			_ = s.notify.Send(ctx, &owner, notification.NewWorkoutLike(viewer.Profile.DisplayName, workout.ID))
		}
	}

	counts, err := s.workoutLikeRepo.CountMapByWorkoutIDs([]uint64{localWorkoutID})
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	return map[string]any{
		"workout_id":  localWorkoutID,
		"likes_count": counts[localWorkoutID],
		"liked":       true,
	}, http.StatusOK, nil
}

func (s *workoutService) CreateReply(ctx context.Context, viewer *model.User, workout *model.Workout, content string) (*dto.WorkoutReplyResponse, error) {
	reply, err := s.workoutReplyRepo.CreateLocalReply(workout.ID, viewer.Profile.ID, content)
	if err != nil {
		return nil, err
	}

	if err := s.db.Preload("Profile").Preload("Profile.User").First(reply, reply.ID).Error; err != nil {
		return nil, err
	}

	if ownerID := workoutOwnerUserID(workout); ownerID != 0 && ownerID != viewer.ID {
		var owner model.User
		if err := s.db.First(&owner, ownerID).Error; err == nil {
			_ = s.notify.Send(ctx, &owner, notification.NewWorkoutReply(viewer.Profile.DisplayName, workout.ID))
		}
	}

	res := dto.NewWorkoutReplyResponse(reply)
	return &res, nil
}

func (s *workoutService) EnrichWorkoutsMetadata(_ context.Context, viewer *model.User, workouts []*model.Workout) ([]dto.WorkoutResponse, error) {
	ids := workoutIDs(workouts)
	results := dto.NewWorkoutsResponse(workouts)

	if len(ids) == 0 {
		return results, nil
	}

	published, _ := s.apOutboxRepo.PublishedMap(viewer.ID, ids)
	applyPublishedFlags(results, published)

	counts, err := s.workoutLikeRepo.CountMapByWorkoutIDs(ids)
	if err == nil {
		liked, likedErr := s.workoutLikeRepo.LikedMapByProfile(ids, viewer.Profile.ID)
		recentLikes, _ := s.workoutLikeRepo.RecentLikesMapByWorkoutIDs(ids, 3)
		if likedErr == nil {
			applyLikeMetadata(results, counts, liked, recentLikes)
		}
	}

	replyCounts, err := s.workoutReplyRepo.CountMapByWorkoutIDs(ids)
	if err == nil {
		applyReplyMetadata(results, replyCounts)
	}

	return results, nil
}

func workoutIDs(ws []*model.Workout) []uint64 {
	ids := make([]uint64, 0, len(ws))
	for _, w := range ws {
		if w != nil {
			ids = append(ids, w.ID)
		}
	}
	return ids
}

func applyPublishedFlags(results []dto.WorkoutResponse, published map[uint64]bool) {
	for i := range results {
		results[i].ActivityPubPublished = published[results[i].ID]
	}
}

func applyLikeMetadata(results []dto.WorkoutResponse, counts map[uint64]int64, liked map[uint64]bool, recentLikes map[uint64][]model.APStatusLike) {
	for i := range results {
		results[i].LikesCount = counts[results[i].ID]
		results[i].LikedByMe = liked[results[i].ID]
		if likes, ok := recentLikes[results[i].ID]; ok && len(likes) > 0 {
			items := make([]dto.WorkoutLikeResponse, 0, len(likes))
			for _, l := range likes {
				items = append(items, dto.NewWorkoutLikeResponse(&l))
			}
			results[i].RecentLikes = items
		}
	}
}

func applyReplyMetadata(results []dto.WorkoutResponse, counts map[uint64]int64) {
	for i := range results {
		results[i].RepliesCount = counts[results[i].ID]
	}
}

func workoutOwnerUserID(workout *model.Workout) uint64 {
	if workout == nil || workout.Profile == nil || workout.Profile.UserID == nil {
		return 0
	}
	return *workout.Profile.UserID
}

func formatDistance(meters float64) string {
	km := meters / 1000
	if km < 10 {
		return strconv.FormatFloat(km, 'f', 2, 64) + " km"
	}
	return strconv.FormatFloat(km, 'f', 1, 64) + " km"
}

func formatDuration(seconds int64) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	if hours > 0 {
		return strconv.FormatInt(hours, 10) + "h " + strconv.FormatInt(minutes, 10) + "m"
	}
	return strconv.FormatInt(minutes, 10) + "m"
}

func (s *workoutService) enqueueWorkoutUpdate(ctx context.Context, workoutID uint64) error {
	if s.client == nil {
		return nil
	}
	raw, err := json.Marshal(struct {
		ID uint64 `json:"id"`
	}{ID: workoutID})
	if err != nil {
		return err
	}
	return s.client.Enqueue(ctx, &gue.Job{
		Type: "update_workout",
		Args: raw,
	})
}
