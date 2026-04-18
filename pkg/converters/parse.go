package converters

import (
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/tkrajina/gpxgo/gpx"
)

var (
	ErrUnsupportedFile = errors.New("unsupported file")
	SupportedFileTypes = []string{".fit", ".ftb", ".gpx", ".tcx", ".zip"}
)

const (
	// Ignore tiny stop/go noise; only material pauses become synthetic timer events.
	minSyntheticPauseDuration = 5 * time.Second
	maxSyntheticPauseSpeed    = 0.3 // m/s
)

type (
	parserFunc func(content []byte) (*gpx.GPX, error)
)

func init() {
	model.WorkoutParser = ParseCollection
}

func Parse(filename string, content []byte) (*model.Workout, error) {
	c, err := ParseCollection(filename, content)
	if err != nil {
		return nil, err
	}

	if len(c) == 0 {
		return nil, nil
	}

	return c[0], nil
}

func ParseCollection(filename string, content []byte) ([]*model.Workout, error) {
	if filename == "" {
		// Assume GPX when filename is empty
		return parseSingle(ParseGPX, "gpx", "", content)
	}

	basename := path.Base(filename)

	c, err := parseContent(basename, content)
	if err != nil {
		return nil, err
	}

	for _, w := range c {
		ensureWorkoutName(w, basename)
	}

	return c, nil
}

func parseContent(filename string, content []byte) ([]*model.Workout, error) {
	suffix := strings.ToLower(path.Ext(filename))

	switch suffix {
	case ".gpx":
		return parseSingle(ParseGPX, "gpx", filename, content)
	case ".fit":
		return ParseFit(content, filename)
	case ".tcx":
		return parseSingle(ParseTCX, "tcx", filename, content)
	case ".zip":
		return ParseZip(content)
	case ".ftb":
		return ParseFTB(content)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFile, filename)
	}
}

func parseSingle(f parserFunc, fileType string, filename string, content []byte) ([]*model.Workout, error) {
	g, err := f(content)
	if err != nil {
		return nil, err
	}

	if g == nil {
		return nil, nil
	}

	return []*model.Workout{workoutFromGPX(g, filename, fileType, content)}, nil
}

func workoutFromGPX(g *gpx.GPX, filename string, fileType string, content []byte) *model.Workout {
	data, records := model.MapDataAndRecordsFromGPX(g)
	if data == nil {
		data = &model.WorkoutGeoMeta{}
	}
	totalDistance, totalDistance2D, totalDuration := model.WorkoutTotalsFromRecords(records)
	statsValues := model.WorkoutStatsFromRecords(records)
	pauseDuration := model.WorkoutPauseDurationFromAverages(totalDistance, totalDuration, statsValues.AverageSpeedNoPause)
	gpxType := model.GPXType(g)
	workoutType, found := model.WorkoutTypeFromData(gpxType)
	customType := ""
	if !found {
		customType = gpxType
	}
	stats := &statsValues

	w := &model.Workout{
		Data:            data,
		Stats:           stats,
		Records:         append([]model.WorkoutRecord(nil), records...),
		Events:          synthesizeTimerEventsFromRecords(records),
		Name:            model.GPXName(g),
		Creator:         g.Creator,
		Type:            workoutType,
		CustomType:      customType,
		DateEnd:         model.WorkoutEndFromRecords(records),
		TotalDistance:   totalDistance,
		TotalDistance2D: totalDistance2D,
		TotalDuration:   totalDuration,
		PauseDuration:   pauseDuration,
	}

	if date := model.GPXDate(g); date != nil {
		w.Date = *date
	}

	setContentAndName(w, filename, fileType, content)
	w.UpdateAverages()
	w.UpdateExtraMetrics()

	return w
}

func ensureWorkoutName(w *model.Workout, basename string) {
	if w == nil || w.Name != "" {
		return
	}

	if basename == "" {
		basename = "workout"
	}

	w.Name = strings.TrimSuffix(basename, path.Ext(basename))
}

func setContentAndName(w *model.Workout, filename string, fileType string, content []byte) {
	ext := strings.TrimPrefix(path.Ext(filename), ".")
	name := strings.TrimSuffix(path.Base(filename), path.Ext(filename))

	if name == "" {
		name = w.Name
	}

	if name == "" {
		name = "workout"
	}

	if ext == "" {
		ext = strings.TrimPrefix(fileType, ".")
	}

	finalName := name
	if ext != "" {
		finalName += "." + ext
	}

	if w.Name == "" {
		w.Name = name
	}

	w.SetContent(finalName, content)
}

func synthesizeTimerEventsFromRecords(records []model.WorkoutRecord) []model.WorkoutEvent {
	if len(records) < 2 {
		return nil
	}

	events := make([]model.WorkoutEvent, 0)
	paused := false
	pausedAt := time.Time{}
	pausedFor := time.Duration(0)

	for i := 1; i < len(records); i++ {
		prev := records[i-1]
		cur := records[i]

		dt := recordIntervalDuration(prev, cur)
		pauseInterval := isPauseInterval(cur, dt)

		if pauseInterval {
			if !paused {
				paused = true
				pausedAt = prev.Time
				pausedFor = 0
			}

			pausedFor += dt
			continue
		}

		if paused {
			if pausedFor >= minSyntheticPauseDuration {
				events = appendSyntheticPauseEvents(events, pausedAt, cur.Time)
			}

			paused = false
			pausedAt = time.Time{}
			pausedFor = 0
		}
	}

	if paused && pausedFor >= minSyntheticPauseDuration {
		lastTime := records[len(records)-1].Time
		events = appendSyntheticPauseEvents(events, pausedAt, lastTime)
	}

	if len(events) == 0 {
		return nil
	}

	return events
}

func recordIntervalDuration(prev, cur model.WorkoutRecord) time.Duration {
	if cur.Duration > 0 {
		return cur.Duration
	}

	if !prev.Time.IsZero() && !cur.Time.IsZero() {
		return cur.Time.Sub(prev.Time)
	}

	return 0
}

func isPauseInterval(cur model.WorkoutRecord, dt time.Duration) bool {
	if dt <= 0 {
		return false
	}

	speed := cur.Distance / dt.Seconds()
	return speed <= maxSyntheticPauseSpeed
}

func appendSyntheticPauseEvents(events []model.WorkoutEvent, stopAt, startAt time.Time) []model.WorkoutEvent {
	if stopAt.IsZero() || startAt.IsZero() || !startAt.After(stopAt) {
		return events
	}

	return append(events,
		model.WorkoutEvent{Timestamp: stopAt, Event: "timer", EventType: "stop"},
		model.WorkoutEvent{Timestamp: startAt, Event: "timer", EventType: "start"},
	)
}
