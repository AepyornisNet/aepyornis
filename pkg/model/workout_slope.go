package model

import (
	"cmp"
	"math"
	"slices"
	"time"
)

type (
	// SlopeState represents the type of slope detected.
	SlopeState string
	// SlopeKind represents the type of slope detected (climb or descent).
	SlopeKind string
	// SlopeState represents the category of the climb.
	Category string
)

const (
	// State machine states for the detector logic
	StateSearching       SlopeState = "SEARCHING"
	StateInSegment       SlopeState = "IN_SEGMENT"
	StateEvaluatingPause SlopeState = "EVALUATING_PAUSE"
	StateStartClimb      SlopeState = "START_CLIMB"
	StateEndClimb        SlopeState = "END_CLIMB"
	StateStartDescent    SlopeState = "START_DESCENT"
	StateEndDescent      SlopeState = "END_DESCENT"

	// Kindes of slope
	SlopeKindClimb   SlopeKind = "climb"
	SlopeKindDescent SlopeKind = "descent"

	// Climb categories
	CategoryHorsCategorie Category = "Hors Catégorie"
	Category1             Category = "Category 1"
	Category2             Category = "Category 2"
	Category3             Category = "Category 3"
	Category4             Category = "Category 4"
	Category5             Category = "Category 5"
	Category6             Category = "Category 6"
	CategoryUncategorized Category = "Uncategorized"

	// Thresholds from Python code
	StartClimbThreshold   float64 = 0.02
	EndClimbThreshold     float64 = 0.01
	MaxPauseLengthMeters  float64 = 200.0
	MaxPauseDescentMeters float64 = 10.0
	MinGain               float64 = 20.0
	MinLength             float64 = 300.0
)

// Detector holds the state for the segment detection process.
type Detector struct {
	segments  []WorkoutClimb
	kind      SlopeKind
	slopeSign float64
	state     SlopeState

	currentSegmentPoints []*WorkoutRecord

	startIdx      int
	pauseStartIdx int
	pauseLength   float64
	pauseDescent  float64
}

// CalculateSlopes processes workout records and stores detected climbs on the workout.
func (w *Workout) CalculateSlopes() {
	points := w.Records
	climbs := DetectSignificantSegments(points, SlopeKindClimb)
	descents := DetectSignificantSegments(points, SlopeKindDescent)

	climbs = append(climbs, descents...)
	slices.SortFunc(climbs, func(a, b WorkoutClimb) int {
		return cmp.Compare(a.Start.TotalDistance, b.Start.TotalDistance)
	})

	w.Climbs = climbs
}

// NewDetector initializes a new Detector for a given kind.
func NewDetector(kind SlopeKind) *Detector {
	slopeSign := 1.0
	if kind != SlopeKindClimb {
		slopeSign = -1.0
	}

	return &Detector{
		kind:      kind,
		slopeSign: slopeSign,
		state:     StateSearching,
	}
}

// SmoothSlopeGrades computes a weighted average slope at each point.
func SmoothSlopeGrades(points []WorkoutRecord, windowMeters float64) {
	for i := range points {
		centerDist := points[i].TotalDistance
		var weightedSlopeSum, totalWeight float64

		for j := range points {
			distDiff := points[j].TotalDistance - centerDist
			distFromCenter := math.Abs(distDiff)
			if distFromCenter > windowMeters/2 || distFromCenter < MaxDeltaMeter/2 {
				continue
			}

			elevDiff := points[j].Elevation - points[i].Elevation
			slope := elevDiff / distDiff

			weight := 1.0 / distFromCenter
			weightedSlopeSum += slope * weight
			totalWeight += weight
		}

		if totalWeight > 0 {
			points[i].SlopeGrade = weightedSlopeSum / totalWeight
		} else {
			points[i].SlopeGrade = 0
		}
	}
}

// DetectSignificantSegments processes a slice of points to find climbs or descents.
func DetectSignificantSegments(points []WorkoutRecord, kind SlopeKind) []WorkoutClimb {
	detector := NewDetector(kind)

	if len(points) < 2 {
		return nil
	}

	SmoothSlopeGrades(points, 300.0)

	// Start with the first point.
	detector.currentSegmentPoints = append(detector.currentSegmentPoints, &points[0])

	for i := 1; i < len(points); i++ {
		prevPoint := &points[i-1]
		currentPoint := &points[i]

		distDiff := currentPoint.TotalDistance - prevPoint.TotalDistance
		elevDiff := (currentPoint.Elevation - prevPoint.Elevation)

		slope := currentPoint.SlopeGrade

		// Adjust slope and elevation diff based on "kind" (climb or descent).
		effectiveSlope := slope * detector.slopeSign
		effectiveElevDiff := elevDiff * detector.slopeSign

		// State machine logic
		switch detector.state {
		case StateSearching:
			if effectiveSlope >= StartClimbThreshold {
				detector.state = StateInSegment
				detector.startIdx = i - 1
				// Add previous and current point to start the segment.
				detector.currentSegmentPoints = []*WorkoutRecord{prevPoint, currentPoint}
			}

		case StateInSegment:
			if effectiveSlope >= EndClimbThreshold {
				detector.currentSegmentPoints = append(detector.currentSegmentPoints, currentPoint)
			} else {
				detector.state = StateEvaluatingPause
				detector.pauseStartIdx = i - 1
				detector.pauseLength = 0
				detector.pauseDescent = 0
				detector.currentSegmentPoints = append(detector.currentSegmentPoints, currentPoint)
			}

		case StateEvaluatingPause:
			detector.currentSegmentPoints = append(detector.currentSegmentPoints, currentPoint)
			detector.pauseLength += distDiff
			if effectiveElevDiff < 0 {
				detector.pauseDescent += math.Abs(effectiveElevDiff)
			}

			if effectiveSlope >= EndClimbThreshold {
				detector.state = StateInSegment
			} else if detector.pauseLength > MaxPauseLengthMeters || detector.pauseDescent > MaxPauseDescentMeters {
				// The pause is too long or a significant descent occurred.
				// Finalize the segment before the pause.
				finalSegmentPoints := detector.currentSegmentPoints[:len(detector.currentSegmentPoints)-(i-detector.pauseStartIdx)]

				detector.validateAndAppendSegment(finalSegmentPoints)

				// Reset state to search for a new segment.
				detector.state = StateSearching
				detector.currentSegmentPoints = []*WorkoutRecord{}
			}
		}
	}

	// Final check for any segment in progress at the end of the data.
	if detector.state == StateInSegment || detector.state == StateEvaluatingPause {
		detector.validateAndAppendSegment(detector.currentSegmentPoints)
	}

	return detector.segments
}

// validateAndAppendSegment is a private method that validates and appends a segment to the detector's slice.
func (d *Detector) validateAndAppendSegment(segmentPoints []*WorkoutRecord) {
	if len(segmentPoints) < 2 {
		return
	}

	start := segmentPoints[0]
	end := segmentPoints[len(segmentPoints)-1]

	var length, gain float64
	var duration time.Duration

	for i := 1; i < len(segmentPoints); i++ {
		length += segmentPoints[i].Distance
		duration += segmentPoints[i].Duration

		elevDiff := segmentPoints[i].Elevation - segmentPoints[i-1].Elevation
		if (d.kind == SlopeKindClimb && elevDiff > 0) || (d.kind == SlopeKindDescent && elevDiff < 0) {
			gain += math.Abs(elevDiff)
		}
	}

	if length <= MinLength || gain <= MinGain {
		return
	}

	avgSlope := gain / length

	cat := CategoryUncategorized
	if d.kind == SlopeKindClimb {
		cat = ClassifyClimbCategory(length, avgSlope)
	}

	segment := WorkoutClimb{
		Index:    len(d.segments) + 1,
		StartIdx: d.startIdx,
		EndIdx:   d.startIdx + len(segmentPoints) - 1,
		Category: cat,
		Type:     d.kind,
		Length:   length,
		AvgSlope: avgSlope,
		Gain:     gain,
		Start:    *start,
		End:      *end,
		Duration: duration,
	}

	d.segments = append(d.segments, segment)
}

func ClassifyClimbCategory(length, slope float64) Category {
	switch {
	case length >= 10000 && slope >= 0.06:
		return CategoryHorsCategorie
	case length >= 8000 && slope >= 0.05:
		return Category1
	case length >= 5000 && slope >= 0.04:
		return Category2
	case length >= 3000 && slope >= 0.03:
		return Category3
	case length >= 2000 && slope >= 0.03:
		return Category4
	case length >= 1000 && slope >= 0.02:
		return Category5
	case length >= 500 && slope >= 0.01:
		return Category6
	default:
		return CategoryUncategorized
	}
}
