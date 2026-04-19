package model

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/tkrajina/gpxgo/gpx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrInvalidData          = errors.New("could not convert data to a GPX structure")
	ErrWorkoutAlreadyExists = errors.New("user already has workout with exact start time")
	ErrWorkoutParserMissing = errors.New("workout parser is not configured")
)

type WorkoutVisibility string

const (
	WorkoutVisibilityPrivate   WorkoutVisibility = ""
	WorkoutVisibilityFollowers WorkoutVisibility = "followers"
	WorkoutVisibilityPublic    WorkoutVisibility = "public"
)

func (v WorkoutVisibility) IsValid() bool {
	switch v {
	case WorkoutVisibilityPrivate, WorkoutVisibilityFollowers, WorkoutVisibilityPublic:
		return true
	default:
		return false
	}
}

func ScopeVisibleWorkouts(query *gorm.DB, ownerID uint64, viewerID uint64, viewerActorIRI string) *gorm.DB {
	if viewerID != 0 && viewerID == ownerID {
		return query.Where("workouts.profile_id = ?", ownerID)
	}

	if viewerActorIRI == "" {
		return query.Where("workouts.profile_id = ? AND workouts.visibility = ?", ownerID, WorkoutVisibilityPublic)
	}

	return query.Where(
		"workouts.profile_id = ? AND (workouts.visibility = ? OR (workouts.visibility = ? AND EXISTS (SELECT 1 FROM followers WHERE followers.user_id = ? AND followers.actor_iri = ? AND followers.approved = ?)))",
		ownerID,
		WorkoutVisibilityPublic,
		WorkoutVisibilityFollowers,
		ownerID,
		viewerActorIRI,
		true,
	)
}

type Workout struct {
	Model

	ProfileID uint64   `gorm:"not null;index;uniqueIndex:idx_start_user" json:"profile_id"` // The ID of the user who owns the workout
	Profile   *Profile `gorm:"foreignKey:ProfileID" json:"profile"`                         // The user who owns the workout

	Date       time.Time         `gorm:"not null;uniqueIndex:idx_start_user" json:"date"` // The timestamp the workout was recorded
	DateEnd    time.Time         `json:"date_end"`                                        // The stop time of the workout
	Visibility WorkoutVisibility `json:"visibility"`                                      // The visibility of the workout (private, followers, public)

	Data                *WorkoutGeoMeta      `gorm:"constraint:OnDelete:CASCADE" json:"data,omitempty"` // The map data associated with the workout
	File                *WorkoutFile         `gorm:"constraint:OnDelete:CASCADE" json:"file,omitempty"` // The file data associated with the workout
	Name                string               `gorm:"not null" json:"name"`                              // The name of the workout
	Creator             string               `json:"creator"`                                           // The device/app that created the workout
	Notes               string               `json:"notes"`                                             // The notes associated with the workout, in markdown
	Type                WorkoutType          `json:"type"`                                              // The type of the workout
	SubType             string               `json:"subType"`                                           // The subtype of the workout
	CustomType          string               `json:"custom_type"`                                       // The type of the workout, custom
	StatsID             *uint64              `json:"-"`
	Stats               *WorkoutStats        `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:StatsID;references:ID" json:"stats,omitempty"`
	TotalDistance       float64              `json:"totalDistance"`                                                                      // The total distance of the workout
	TotalDistance2D     float64              `json:"totalDistance2D"`                                                                    // The total 2D distance of the workout
	TotalDuration       time.Duration        `json:"totalDuration"`                                                                      // The total duration of the workout
	PauseDuration       time.Duration        `json:"pauseDuration"`                                                                      // The total pause duration of the workout
	TotalRepetitions    int                  `json:"totalRepetitions"`                                                                   // The number of repetitions of the workout
	TotalWeight         float64              `json:"totalWeight"`                                                                        // The weight of the workout
	ExtraMetrics        []string             `gorm:"serializer:json" json:"extraMetrics"`                                                // Extra metrics available
	Equipment           []Equipment          `gorm:"constraint:OnDelete:CASCADE;many2many:workout_equipment" json:"equipment,omitempty"` // Which equipment is used for this workout
	RouteSegmentMatches []*RouteSegmentMatch `gorm:"constraint:OnDelete:CASCADE" json:"routeSegmentMatches,omitempty"`                   // Which route segments match
	Attachments         []WorkoutAttachment  `gorm:"constraint:OnDelete:CASCADE" json:"attachments,omitempty"`
	Records             []WorkoutRecord      `gorm:"constraint:OnDelete:CASCADE" json:"records"`          // The GPS points of the workout
	Events              []WorkoutEvent       `gorm:"constraint:OnDelete:CASCADE" json:"events,omitempty"` // Parsed workout events (e.g. FIT timer start/stop)
	Laps                []WorkoutLap         `gorm:"constraint:OnDelete:CASCADE" json:"laps"`             // The laps of the workout
	Climbs              []WorkoutClimb       `gorm:"constraint:OnDelete:CASCADE" json:"climbs"`           // Auto-detected climbs
	Locked              bool                 `json:"locked"`                                              // Whether the workout's main attributes should be auto-updated
	Dirty               bool                 `json:"dirty"`                                               // Whether the workout has been modified and the details should be re-rendered
}

func omitWorkoutAssociations(tx *gorm.DB) *gorm.DB {
	return tx.Omit(clause.Associations).Omit("Stats", "Data", "File", "Equipment", "RouteSegmentMatches", "Records", "Events", "Laps", "Climbs", "Attachments")
}

func (w *Workout) HasCustomType() bool {
	return w.Type == WorkoutTypeOther
}

func (w *Workout) AfterFind(tx *gorm.DB) error {
	if w.Profile != nil && w.Profile.User != nil {
		w.Profile.User.db = tx
	}

	return nil
}

func (w *Workout) GetDate() time.Time {
	return w.Date
}

func (w *Workout) Filename() string {
	if !w.HasFile() {
		return w.Name + ".txt"
	}

	return w.File.Filename
}

func (w *Workout) HasElevationData() bool {
	// If both min & max elevation are 0, we don't have elevation information
	return w.MinElevation() != 0 || w.MaxElevation() != 0
}

func (w *Workout) HasPause() bool {
	return w.PauseDuration == 0
}

func (w *Workout) HasFile() bool {
	if w.File == nil {
		return false
	}

	return w.File.Filename != "" && w.File.Content != nil
}

func (w *Workout) HasTracks() bool {
	if w.Data == nil {
		return false
	}

	if w.Data.Center.IsZero() {
		return false
	}

	if len(w.Records) == 0 {
		return false
	}

	return w.Type.IsLocation()
}

func (w *Workout) Weight() float64 {
	return w.TotalWeight
}

func (w *Workout) AverageSpeed() float64 {
	if w.Stats == nil {
		return 0
	}

	return w.Stats.AverageSpeed
}

func (w *Workout) GetEnd() time.Time {
	if w.TotalDuration <= 0 {
		return w.GetDate().Add(minEventDuration)
	}

	return w.GetDate().Add(w.Duration())
}

func (w *Workout) Repetitions() int {
	return w.TotalRepetitions
}

func (w *Workout) Duration() time.Duration {
	return w.TotalDuration
}

func (w *Workout) FullAddress() string {
	if w.Data == nil {
		return ""
	}

	if w.Data.Address != nil {
		return w.Data.Address.FormattedAddress
	}

	return w.Data.AddressString
}

func (w *Workout) Center() *MapCenter {
	if w.Data == nil {
		return nil
	}

	return &w.Data.Center
}

func (w *Workout) TotalDown() float64 {
	if w.Stats == nil {
		return 0
	}

	return w.Stats.TotalDown
}

func (w *Workout) TotalUp() float64 {
	if w.Stats == nil {
		return 0
	}

	return w.Stats.TotalUp
}

func (w *Workout) MaxElevation() float64 {
	if w.Stats == nil {
		return 0
	}

	return w.Stats.MaxElevation
}

func (w *Workout) MinElevation() float64 {
	if w.Stats == nil {
		return 0
	}

	return w.Stats.MinElevation
}

func (w *Workout) MaxSpeed() float64 {
	if w.Stats == nil {
		return 0
	}

	return w.Stats.MaxSpeed
}

func (w *Workout) MaxCadence() float64 {
	if w.Stats == nil {
		return 0
	}

	return w.Stats.MaxCadence
}

func (w *Workout) AverageSpeedNoPause() float64 {
	if w.Stats == nil {
		return 0
	}

	return w.Stats.AverageSpeedNoPause
}

func (w *Workout) AverageCadence() float64 {
	if w.Stats == nil {
		return 0
	}

	return w.Stats.AverageCadence
}

func (w *Workout) City() string {
	if w.Data == nil || w.Data.Address == nil {
		return ""
	}

	return w.Data.Address.City
}

func (w *Workout) Timezone() string {
	if w.Data == nil {
		return ""
	}

	return w.Data.Center.TZ
}

func (w *Workout) Address() string {
	if w.Data == nil {
		return UnknownLocation
	}

	if w.Data.AddressString != "" {
		return w.Data.AddressString
	}

	return w.Data.addressString()
}

func (w *Workout) Distance() float64 {
	return w.TotalDistance
}

func NewWorkout(p *Profile, workoutType WorkoutType, notes string, filename string, content []byte) ([]*Workout, error) {
	if p == nil {
		return nil, ErrNoUser
	}

	filename = filepath.Base(filename)

	if WorkoutParser == nil {
		return nil, ErrWorkoutParserMissing
	}

	parsed, err := WorkoutParser(filename, content)
	if err != nil {
		return nil, fmt.Errorf("could not parse workout data: %w", err)
	}

	if len(parsed) == 0 {
		return nil, nil
	}

	workouts := make([]*Workout, 0, len(parsed))

	for _, parsedWorkout := range parsed {
		if parsedWorkout == nil {
			continue
		}

		w := parsedWorkout
		w.Profile = p
		w.ProfileID = p.ID
		w.Dirty = true
		w.Notes = notes

		if w.Data == nil {
			w.Data = &WorkoutGeoMeta{}
		}

		if workoutType == WorkoutTypeAutoDetect {
			w.Type = autoDetectWorkoutType(w.Stats, w.Creator, string(w.Type), w.Name)
		} else {
			w.Type = workoutType
		}

		// If multiple files are extracted (e.g., from a zip), prefer the per-file filename.
		if w.File != nil && w.File.Filename == "" {
			w.File.Filename = filename
		}

		workouts = append(workouts, w)
	}

	return workouts, nil
}

// SetContent stores the raw workout file along with its checksum.
func (w *Workout) SetContent(filename string, content []byte) {
	if content == nil {
		return
	}

	h := sha256.New()
	h.Write(content)

	w.File = &WorkoutFile{
		Content:  content,
		Checksum: h.Sum(nil),
		Filename: filename,
	}
}

func WorkoutTypeFromData(gpxType string) (WorkoutType, bool) {
	switch strings.ToLower(gpxType) {
	case "running", "run":
		return WorkoutTypeRunning, true
	case "walking", "walk":
		return WorkoutTypeWalking, true
	case "cycling", "cycle":
		return WorkoutTypeCycling, true
	case "snowboarding":
		return WorkoutTypeSnowboarding, true
	case "horse-riding", "horseback-riding":
		return WorkoutTypeHorseRiding, true
	case "inline-skating", "skating", "skate":
		return WorkoutTypeInlineSkating, true
	case "skiing":
		return WorkoutTypeSkiing, true
	case "swimming":
		return WorkoutTypeSwimming, true
	case "kayaking":
		return WorkoutTypeKayaking, true
	case "golfing":
		return WorkoutTypeGolfing, true
	case "hiking":
		return WorkoutTypeHiking, true
	case "push-ups":
		return WorkoutTypePushups, true
	case "rowing":
		return WorkoutTypeRowing, true
	default:
		return WorkoutTypeAutoDetect, false
	}
}

func autoDetectWorkoutType(stats *WorkoutStats, creator string, dataType string, dataName string) WorkoutType {
	if workoutType, ok := WorkoutTypeFromData(dataType); ok {
		return workoutType
	}

	if workoutType, ok := WorkoutTypeFromData(creator); ok {
		return workoutType
	}

	if len(dataName) > 0 {
		nameField := strings.Fields(dataName)
		if len(nameField) > 0 {
			if workoutType, ok := WorkoutTypeFromData(nameField[0]); ok {
				return workoutType
			}
		}
	}

	if stats != nil {
		if 3.6*stats.AverageSpeedNoPause > 15.0 {
			return WorkoutTypeCycling
		}

		if 3.6*stats.AverageSpeedNoPause > 7.0 {
			return WorkoutTypeRunning
		}
	}

	return WorkoutTypeWalking
}

func GetRecentWorkoutsWithOffset(db *gorm.DB, count int, offset int) ([]*Workout, error) {
	var w []*Workout

	if err := PreloadWorkoutData(db).Preload("Profile").Preload("Profile.User").Order("date DESC").Limit(count).Offset(offset).Find(&w).Error; err != nil {
		return nil, err
	}

	return w, nil
}

func GetWorkouts(db *gorm.DB) ([]*Workout, error) {
	var w []*Workout

	if err := PreloadWorkoutDetails(db).Order("date DESC").Find(&w).Error; err != nil {
		return nil, err
	}

	return w, nil
}

func GetWorkoutDetails(db *gorm.DB, id uint64) (*Workout, error) {
	return GetWorkout(PreloadWorkoutDetails(db).Preload("File"), id)
}

func GetMapData(db *gorm.DB, id uint64) (*WorkoutGeoMeta, error) {
	var md WorkoutGeoMeta

	if err := db.First(&md, id).Error; err != nil {
		return nil, err
	}

	return &md, nil
}

func GetWorkout(db *gorm.DB, id uint64) (*Workout, error) {
	var w Workout

	if err := db.
		Preload("RouteSegmentMatches.RouteSegment").
		Preload("Stats").
		Preload("Data").
		Preload("Laps", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("sort_order ASC")
		}).
		Preload("Laps.Stats").
		Preload("Climbs", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("sort_order ASC")
		}).
		Preload("Records", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("sort_order ASC")
		}).
		Preload("Events", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("sort_order ASC")
		}).
		Preload("Profile").
		Preload("Profile.User").
		Preload("Equipment").
		First(&w, id).
		Error; err != nil {
		return nil, err
	}

	sort.Slice(w.RouteSegmentMatches, func(i, j int) bool {
		return w.RouteSegmentMatches[i].Distance > w.RouteSegmentMatches[j].Distance
	})

	return &w, nil
}

func (w *Workout) Delete(db *gorm.DB) error {
	return db.Select(clause.Associations).Delete(w).Error
}

func (w *Workout) Create(db *gorm.DB) error {
	err := w.create(db)
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrWorkoutAlreadyExists
	}

	return err
}

func (w *Workout) create(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := omitWorkoutAssociations(tx).Create(w).Error; err != nil {
			return fmt.Errorf("create workout row: %w", err)
		}

		return persistWorkoutRelations(tx, w)
	})
}

func (w *Workout) Save(db *gorm.DB) error {
	err := w.save(db)
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrWorkoutAlreadyExists
	}

	return err
}

func (w *Workout) save(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if w.ID == 0 && w.ProfileID != 0 && !w.Date.IsZero() {
			var existing Workout
			if err := tx.Select("id").Where("profile_id = ? AND date = ?", w.ProfileID, w.Date).First(&existing).Error; err == nil {
				w.ID = existing.ID
			}
		}

		if w.ID == 0 {
			if err := omitWorkoutAssociations(tx).Create(w).Error; err != nil {
				return err
			}
		} else {
			if err := omitWorkoutAssociations(tx).Save(w).Error; err != nil {
				return err
			}
		}

		return persistWorkoutRelations(tx, w)
	})
}

func persistWorkoutRelations(tx *gorm.DB, w *Workout) error {
	if err := saveWorkoutStats(tx, w); err != nil {
		return err
	}

	if err := saveWorkoutGeoMeta(tx, w); err != nil {
		return err
	}

	if err := persistWorkoutRecords(tx, w); err != nil {
		return err
	}

	if err := persistWorkoutEvents(tx, w); err != nil {
		return err
	}

	if err := persistWorkoutLaps(tx, w); err != nil {
		return err
	}

	if err := persistWorkoutClimbs(tx, w); err != nil {
		return err
	}

	if err := persistWorkoutFile(tx, w); err != nil {
		return err
	}

	if err := persistWorkoutRouteSegmentMatches(tx, w); err != nil {
		return err
	}

	return nil
}

func persistWorkoutRecords(tx *gorm.DB, w *Workout) error {
	if w.Records == nil {
		return nil
	}

	for i := range w.Records {
		w.Records[i].WorkoutID = w.ID
		w.Records[i].SortOrder = i
	}

	if err := tx.Where("workout_id = ?", w.ID).Delete(&WorkoutRecord{}).Error; err != nil {
		return err
	}

	if len(w.Records) == 0 {
		return nil
	}

	return tx.CreateInBatches(&w.Records, mapDataPointsInsertBatchSize).Error
}

func persistWorkoutLaps(tx *gorm.DB, w *Workout) error {
	if w.Laps == nil {
		return nil
	}

	for i := range w.Laps {
		w.Laps[i].WorkoutID = w.ID
		w.Laps[i].SortOrder = i
		if err := saveLapStats(tx, &w.Laps[i]); err != nil {
			return err
		}
	}

	if err := tx.Where("workout_id = ?", w.ID).Delete(&WorkoutLap{}).Error; err != nil {
		return err
	}

	if len(w.Laps) == 0 {
		return nil
	}

	return tx.CreateInBatches(&w.Laps, mapDataClimbsInsertBatchSize).Error
}

func persistWorkoutEvents(tx *gorm.DB, w *Workout) error {
	if w.Events == nil {
		return nil
	}

	for i := range w.Events {
		w.Events[i].WorkoutID = w.ID
		w.Events[i].SortOrder = i
	}

	if err := tx.Where("workout_id = ?", w.ID).Delete(&WorkoutEvent{}).Error; err != nil {
		return err
	}

	if len(w.Events) == 0 {
		return nil
	}

	return tx.CreateInBatches(&w.Events, mapDataClimbsInsertBatchSize).Error
}

func persistWorkoutClimbs(tx *gorm.DB, w *Workout) error {
	if w.Climbs == nil {
		return nil
	}

	for i := range w.Climbs {
		w.Climbs[i].WorkoutID = w.ID
		w.Climbs[i].SortOrder = i
	}

	if err := tx.Where("workout_id = ?", w.ID).Delete(&WorkoutClimb{}).Error; err != nil {
		return err
	}

	if len(w.Climbs) == 0 {
		return nil
	}

	return tx.CreateInBatches(&w.Climbs, mapDataClimbsInsertBatchSize).Error
}

func persistWorkoutFile(tx *gorm.DB, w *Workout) error {
	if w.File == nil {
		return nil
	}

	w.File.WorkoutID = w.ID

	if w.File.ID == 0 {
		return tx.Create(w.File).Error
	}

	return tx.Save(w.File).Error
}

func persistWorkoutRouteSegmentMatches(tx *gorm.DB, w *Workout) error {
	if w.RouteSegmentMatches == nil {
		return nil
	}

	return replaceWorkoutRouteSegmentMatches(tx, w.ID, w.RouteSegmentMatches)
}

func (w *Workout) ReparseFile() (*Workout, error) {
	if !w.HasFile() {
		return nil, errors.New("workout has no GPX")
	}

	if WorkoutParser == nil {
		return nil, ErrWorkoutParserMissing
	}

	workouts, err := WorkoutParser(w.File.Filename, w.File.Content)
	if err != nil {
		return nil, err
	}

	if len(workouts) == 0 {
		return nil, nil
	}

	return workouts[0], nil
}

func (w *Workout) setData(updated *Workout) {
	if updated == nil || updated.Data == nil {
		return
	}

	data := updated.Data
	records := updated.Records
	events := updated.Events
	laps := updated.Laps

	if !w.Locked {
		w.DateEnd = updated.DateEnd
		w.SubType = updated.SubType
		w.Stats = updated.Stats
		w.TotalDistance = updated.TotalDistance
		w.TotalDistance2D = updated.TotalDistance2D
		w.TotalDuration = updated.TotalDuration
		w.PauseDuration = updated.PauseDuration
		w.TotalRepetitions = updated.TotalRepetitions
		w.TotalWeight = updated.TotalWeight
		w.ExtraMetrics = append([]string(nil), updated.ExtraMetrics...)
	} else if w.Data != nil {
		data.Address = w.Data.Address
	}

	if w.Data == nil {
		w.Data = data
		w.Data.WorkoutID = w.ID
		w.Records = append([]WorkoutRecord(nil), records...)
		w.Events = append([]WorkoutEvent(nil), events...)
		w.Laps = append([]WorkoutLap(nil), laps...)

		return
	}

	data.ID = w.Data.ID
	data.CreatedAt = w.Data.CreatedAt
	data.WorkoutID = w.ID

	w.Records = append([]WorkoutRecord(nil), records...)
	w.Events = append([]WorkoutEvent(nil), events...)
	w.Laps = append([]WorkoutLap(nil), laps...)
	w.Data = data
}

func (w *Workout) UpdateAverages() {
	if w.Data == nil {
		return
	}

	if w.Stats == nil {
		w.Stats = &WorkoutStats{}
	}

	if stats, ok := w.aggregateDetailsStats(); ok {
		w.applyRangeStats(stats)
		return
	}

	w.calculateAverageSpeeds()
}

func (w *Workout) aggregateDetailsStats() (MapDataRangeStats, bool) {
	if w.Data == nil {
		return MapDataRangeStats{}, false
	}

	if len(w.Records) < 2 {
		return MapDataRangeStats{}, false
	}

	return StatsForRange(w.Records, 0, len(w.Records)-1)
}

func (w *Workout) applyRangeStats(stats MapDataRangeStats) {
	w.Stats.MinElevation = stats.MinElevation
	w.Stats.MaxElevation = stats.MaxElevation
	w.Stats.TotalUp = stats.TotalUp
	w.Stats.TotalDown = stats.TotalDown
	w.Stats.AverageSlope = stats.AverageSlope
	w.Stats.MinSlope = stats.MinSlope
	w.Stats.MaxSlope = stats.MaxSlope
	w.Stats.AverageSpeed = stats.AverageSpeed
	w.Stats.AverageSpeedNoPause = stats.AverageSpeedNoPause
	w.Stats.MinSpeed = stats.MinSpeed
	w.Stats.MaxSpeed = stats.MaxSpeed
	w.Stats.AverageCadence = stats.AverageCadence
	w.Stats.MinCadence = stats.MinCadence
	w.Stats.MaxCadence = stats.MaxCadence
	w.Stats.AverageHeartRate = stats.AverageHeartRate
	w.Stats.MinHeartRate = stats.MinHeartRate
	w.Stats.MaxHeartRate = stats.MaxHeartRate
	w.Stats.AverageRespirationRate = stats.AverageRespirationRate
	w.Stats.MinRespirationRate = stats.MinRespirationRate
	w.Stats.MaxRespirationRate = stats.MaxRespirationRate
	w.Stats.AveragePower = stats.AveragePower
	w.Stats.MinPower = stats.MinPower
	w.Stats.MaxPower = stats.MaxPower
	w.Stats.AverageTemperature = stats.AverageTemperature
	w.Stats.MinTemperature = stats.MinTemperature
	w.Stats.MaxTemperature = stats.MaxTemperature
}

func (w *Workout) calculateAverageSpeeds() {
	if w.Stats == nil {
		w.Stats = &WorkoutStats{}
	}

	w.Stats.AverageSpeed = 0
	w.Stats.AverageSpeedNoPause = 0

	if w.TotalDuration == 0 {
		return
	}

	w.Stats.AverageSpeed = w.TotalDistance / w.TotalDuration.Seconds()

	if w.TotalDuration == w.PauseDuration {
		w.Stats.AverageSpeedNoPause = w.Stats.AverageSpeed
		return
	}

	w.Stats.AverageSpeedNoPause = w.TotalDistance / (w.TotalDuration - w.PauseDuration).Seconds()
}

func (w *Workout) UpdateData(db *gorm.DB) error {
	if !w.HasFile() {
		// We only update data from (stored) GPX data
		w.Dirty = false

		return w.Save(db)
	}

	updatedWorkout, err := w.ReparseFile()
	if err != nil {
		return err
	}

	if updatedWorkout == nil || updatedWorkout.Data == nil {
		return errors.New("parsed workout has no map data")
	}

	w.setData(updatedWorkout)

	if err := w.Data.Save(db); err != nil {
		return err
	}

	if err := w.UpdateRouteSegmentMatches(db); err != nil {
		return err
	}

	w.UpdateAverages()
	w.UpdateExtraMetrics()
	if err := w.UpdateRecords(db); err != nil {
		return err
	}
	w.Data.UpdateAddress()
	w.CalculateSlopes()

	w.Dirty = false

	return w.Save(db)
}

func (w *Workout) UpdateRouteSegmentMatches(db *gorm.DB) error {
	var routeSegments []*RouteSegment
	if err := db.Preload("RouteSegmentMatches.Workout").Order("created_at DESC").Find(&routeSegments).Error; err != nil {
		return err
	}

	w.RouteSegmentMatches = w.FindMatches(routeSegments)

	return nil
}

func (w *Workout) RepetitionFrequencyPerMinute() float64 {
	if w.TotalRepetitions == 0 || w.Duration() <= 0 {
		return 0
	}

	return float64(w.TotalRepetitions) / w.Duration().Minutes()
}

func (w *Workout) HasCalories() bool {
	return w.Type.IsDuration()
}

func (w *Workout) CaloriesBurned() float64 {
	if !w.Type.IsDuration() {
		return 0
	}

	if w.Profile == nil || w.Profile.User == nil {
		return 0
	}

	weight := w.Profile.User.WeightAt(w.Date)
	// Calories burned = weight * time * intensity (MET)
	cb := weight * w.Duration().Hours() * w.MET()

	return cb
}

func (w *Workout) HasElevation() bool {
	return w.HasExtraMetric("elevation")
}

func (w *Workout) HasEnhancedSpeed() bool {
	return w.HasExtraMetric("speed")
}

func (w *Workout) HasTemperature() bool {
	return w.HasExtraMetric("temperature")
}

func (w *Workout) HasCadence() bool {
	return w.HasExtraMetric("cadence")
}

func (w *Workout) HasHeartRate() bool {
	return w.HasExtraMetric("heart-rate")
}

func (w *Workout) HasHeading() bool {
	return w.HasExtraMetric("heading")
}

func (w *Workout) HasAccuracy() bool {
	return w.HasExtraMetric("accuracy")
}

func (w *Workout) UpdateExtraMetrics() {
	if w.Data == nil {
		return
	}

	w.ExtraMetrics = w.Data.UpdateExtraMetrics(w.Records)
}

// UpdateRecords recalculates and persists best distance intervals for this workout.
func (w *Workout) UpdateRecords(db *gorm.DB) error {
	if db == nil {
		return errors.New("nil db")
	}

	targets := distanceRecordTargetsFor(w.Type)

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("workout_id = ?", w.ID).Delete(&WorkoutIntervalBest{}).Error; err != nil {
			return err
		}

		if len(targets) == 0 || w.Data == nil || len(w.Records) < 2 {
			return nil
		}

		records := fastestDistancesForWorkout(w, targets)
		if len(records) == 0 {
			return nil
		}

		rows := make([]*WorkoutIntervalBest, 0, len(records))
		for _, r := range records {
			rows = append(rows, &WorkoutIntervalBest{
				WorkoutID:       w.ID,
				Label:           r.Label,
				TargetDistance:  r.TargetDistance,
				Distance:        r.Distance,
				DurationSeconds: r.Duration.Seconds(),
				Average:         r.AverageSpeed,
				Type:            WorkoutIntervalBestTypeSpeed,
				StartIndex:      r.StartIndex,
				EndIndex:        r.EndIndex,
			})
		}

		return tx.Create(&rows).Error
	})
}

func (w *Workout) HasExtraMetrics() bool {
	return len(w.ExtraMetrics) > 0
}

func (w *Workout) HasExtraMetric(name string) bool {
	return slices.Contains(w.ExtraMetrics, name)
}

func (w *Workout) EquipmentIDs() []uint64 {
	ids := make([]uint64, 0, len(w.Equipment))

	for _, e := range w.Equipment {
		ids = append(ids, e.ID)
	}

	return ids
}

func (w *Workout) Uses(e Equipment) bool {
	return slices.Contains(w.EquipmentIDs(), e.ID)
}

func (w *Workout) Export() ([]byte, error) {
	var buf bytes.Buffer

	if err := json.NewEncoder(&buf).Encode(w); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func replaceWorkoutRouteSegmentMatches(tx *gorm.DB, workoutID uint64, matches []*RouteSegmentMatch) error {
	if tx == nil {
		return errors.New("nil db")
	}

	if err := tx.Where("workout_id = ?", workoutID).Delete(&RouteSegmentMatch{}).Error; err != nil {
		return err
	}

	if len(matches) == 0 {
		return nil
	}

	rows := make([]*RouteSegmentMatch, 0, len(matches))
	for _, m := range matches {
		if m == nil || m.RouteSegmentID == 0 {
			continue
		}

		rows = append(rows, &RouteSegmentMatch{
			RouteSegmentID: m.RouteSegmentID,
			WorkoutID:      workoutID,
			FirstID:        m.FirstID,
			LastID:         m.LastID,
			Distance:       m.Distance,
			Duration:       m.Duration,
		})
	}

	if len(rows) == 0 {
		return nil
	}

	return tx.Omit(clause.Associations).CreateInBatches(&rows, mapDataPointsInsertBatchSize).Error
}

func init() {
	WorkoutParser = defaultWorkoutParser
}

func defaultWorkoutParser(filename string, content []byte) ([]*Workout, error) {
	gpxContent, err := gpx.ParseBytes(content)
	if err != nil {
		return nil, err
	}

	data, records := MapDataAndRecordsFromGPX(gpxContent)
	totalDistance, totalDistance2D, totalDuration := WorkoutTotalsFromRecords(records)
	statsValues := WorkoutStatsFromRecords(records)
	pauseDuration := WorkoutPauseDurationFromAverages(totalDistance, totalDuration, statsValues.AverageSpeedNoPause)
	workoutType, _ := WorkoutTypeFromData(GPXType(gpxContent))
	dateEnd := WorkoutEndFromRecords(records)
	stats := &WorkoutStats{
		MinElevation:           statsValues.MinElevation,
		MaxElevation:           statsValues.MaxElevation,
		TotalUp:                statsValues.TotalUp,
		TotalDown:              statsValues.TotalDown,
		AverageSlope:           statsValues.AverageSlope,
		MinSlope:               statsValues.MinSlope,
		MaxSlope:               statsValues.MaxSlope,
		AverageSpeed:           statsValues.AverageSpeed,
		AverageSpeedNoPause:    statsValues.AverageSpeedNoPause,
		MinSpeed:               statsValues.MinSpeed,
		MaxSpeed:               statsValues.MaxSpeed,
		AverageCadence:         statsValues.AverageCadence,
		MinCadence:             statsValues.MinCadence,
		MaxCadence:             statsValues.MaxCadence,
		AverageHeartRate:       statsValues.AverageHeartRate,
		MinHeartRate:           statsValues.MinHeartRate,
		MaxHeartRate:           statsValues.MaxHeartRate,
		AverageRespirationRate: statsValues.AverageRespirationRate,
		MinRespirationRate:     statsValues.MinRespirationRate,
		MaxRespirationRate:     statsValues.MaxRespirationRate,
		AveragePower:           statsValues.AveragePower,
		MinPower:               statsValues.MinPower,
		MaxPower:               statsValues.MaxPower,
		AverageTemperature:     statsValues.AverageTemperature,
		MinTemperature:         statsValues.MinTemperature,
		MaxTemperature:         statsValues.MaxTemperature,
	}
	w := &Workout{
		Data:            data,
		Stats:           stats,
		Records:         append([]WorkoutRecord(nil), records...),
		Name:            GPXName(gpxContent),
		Creator:         gpxContent.Creator,
		Type:            workoutType,
		DateEnd:         dateEnd,
		TotalDistance:   totalDistance,
		TotalDistance2D: totalDistance2D,
		TotalDuration:   totalDuration,
		PauseDuration:   pauseDuration,
	}

	if date := GPXDate(gpxContent); date != nil {
		w.Date = *date
	}

	if filename == "" {
		filename = w.Name
	}

	w.SetContent(filename, content)
	w.UpdateAverages()
	w.UpdateExtraMetrics()

	return []*Workout{w}, nil
}

func saveWorkoutStats(tx *gorm.DB, w *Workout) error {
	if w == nil {
		return nil
	}

	if w.Stats == nil {
		w.StatsID = nil

		return tx.Model(w).Update("stats_id", nil).Error
	}

	if w.Stats.ID == 0 {
		if err := tx.Create(w.Stats).Error; err != nil {
			return err
		}
	} else {
		if err := tx.Save(w.Stats).Error; err != nil {
			return err
		}
	}

	statsID := w.Stats.ID
	w.StatsID = &statsID

	return tx.Model(&Workout{}).Where("id = ?", w.ID).Update("stats_id", w.StatsID).Error
}

func saveWorkoutGeoMeta(tx *gorm.DB, w *Workout) error {
	if w == nil {
		return nil
	}

	if w.Data == nil {
		if w.ID == 0 {
			return nil
		}

		return tx.Where("workout_id = ?", w.ID).Delete(&WorkoutGeoMeta{}).Error
	}

	w.Data.WorkoutID = w.ID

	return w.Data.Save(tx)
}

func saveLapStats(tx *gorm.DB, lap *WorkoutLap) error {
	if lap == nil {
		return nil
	}

	if lap.Stats == nil {
		lap.StatsID = nil

		return nil
	}

	if lap.Stats.ID == 0 {
		if err := tx.Create(lap.Stats).Error; err != nil {
			return err
		}
	} else {
		if err := tx.Save(lap.Stats).Error; err != nil {
			return err
		}
	}

	statsID := lap.Stats.ID
	lap.StatsID = &statsID

	return nil
}

const minEventDuration = 1 * time.Second

// WorkoutParser is configured by the converters package to parse file content into Workout models.
// It is left nil in tests that do not require parsing.
var WorkoutParser func(filename string, content []byte) ([]*Workout, error)
