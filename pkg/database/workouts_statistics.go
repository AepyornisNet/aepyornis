package database

import (
	"errors"
	"fmt"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/templatehelpers"
)

type BreakdownItem struct {
	FirstPoint    *MapPoint     `json:"firstPoint"`    // First GPS point in this item
	LastPoint     *MapPoint     `json:"lastPoint"`     // Last GPS point in this item
	StartIndex    int           `json:"startIndex"`    // Start index in the points slice
	EndIndex      int           `json:"endIndex"`      // End index in the points slice
	UnitName      string        `json:"unitName"`      // Unit name
	UnitCount     float64       `json:"unitCount"`     // Count of the unit per item
	Counter       int           `json:"counter"`       // Counter of this item in the list of items
	Distance      float64       `json:"distance"`      // Distance in this item
	TotalDistance float64       `json:"totalDistance"` // Total distance in all items up to and including this item
	Duration      time.Duration `json:"duration"`      // Duration in this item (moving time)
	TotalDuration time.Duration `json:"totalDuration"` // Total duration in all items up to and including this item (moving time)
	Speed         float64       `json:"speed"`         // Speed in this item
	PauseDuration time.Duration `json:"pauseDuration"` // Paused duration in this item

	MinElevation float64 `json:"minElevation"`
	MaxElevation float64 `json:"maxElevation"`
	TotalUp      float64 `json:"totalUp"`
	TotalDown    float64 `json:"totalDown"`

	AverageSpeedNoPause float64 `json:"averageSpeedNoPause"`
	MaxSpeed            float64 `json:"maxSpeed"`

	AverageCadence float64 `json:"averageCadence"`
	MaxCadence     float64 `json:"maxCadence"`

	AverageHeartRate float64 `json:"averageHeartRate"`
	MaxHeartRate     float64 `json:"maxHeartRate"`

	AveragePower float64 `json:"averagePower"`
	MaxPower     float64 `json:"maxPower"`
	IsBest       bool    `json:"isBest"`  // Whether this item is the best of the list
	IsWorst      bool    `json:"isWorst"` // Whether this item is the worst of the list
}

func (bi *BreakdownItem) createNext(fp *MapPoint) BreakdownItem {
	return BreakdownItem{
		UnitCount:     bi.UnitCount,
		UnitName:      bi.UnitName,
		Counter:       bi.Counter + 1,
		TotalDistance: bi.TotalDistance,
		TotalDuration: bi.TotalDuration,
		FirstPoint:    fp,
		StartIndex:    bi.EndIndex,
	}
}

func (bi *BreakdownItem) canHave(count float64, unit string, fp *MapPoint) bool {
	switch unit {
	case "distance":
		return bi.canHaveDistance(fp.Distance, float64(bi.Counter)*count)
	case "duration":
		return bi.canHaveDuration(fp.Duration, time.Duration(float64(bi.Counter)*count))
	}

	return true
}

func (bi *BreakdownItem) canHaveDistance(distance, next float64) bool {
	return bi.TotalDistance+distance < next
}

func (bi *BreakdownItem) canHaveDuration(duration, next time.Duration) bool {
	return bi.TotalDuration+duration < next
}

func (bi *BreakdownItem) CalcultateSpeed() {
	if bi.Duration.Seconds() == 0 {
		bi.Speed = 0
		return
	}

	bi.Speed = bi.Distance / bi.Duration.Seconds()
}

func (bi *BreakdownItem) applyRangeStats(stats MapDataRangeStats) {
	bi.MinElevation = stats.MinElevation
	bi.MaxElevation = stats.MaxElevation
	bi.TotalUp = stats.TotalUp
	bi.TotalDown = stats.TotalDown

	bi.AverageSpeedNoPause = stats.AverageSpeedNoPause
	bi.Speed = stats.AverageSpeedNoPause
	bi.MaxSpeed = stats.MaxSpeed

	bi.AverageCadence = stats.AverageCadence
	bi.MaxCadence = stats.MaxCadence

	bi.AverageHeartRate = stats.AverageHeartRate
	bi.MaxHeartRate = stats.MaxHeartRate

	bi.AveragePower = stats.AveragePower
	bi.MaxPower = stats.MaxPower
}

//gocyclo:ignore
func calculateBestAndWorst(items []BreakdownItem) {
	if len(items) == 0 {
		return
	}

	worst := 0
	best := 0

	for i := range items {
		if items[i].Speed < items[worst].Speed {
			worst = i
		}

		if items[i].Speed > items[best].Speed {
			best = i
		}
	}

	items[worst].IsWorst = true
	items[best].IsBest = true
}

func (w *Workout) statisticsWithUnit(count float64, unit string) []BreakdownItem {
	if w.Data.Details == nil ||
		len(w.Data.Details.Points) == 0 {
		return nil
	}

	var items []BreakdownItem

	points := w.Data.Details.Points

	nextItem := BreakdownItem{
		UnitCount:  count,
		UnitName:   unit,
		Counter:    1,
		FirstPoint: &points[0],
		StartIndex: 0,
	}

	for i := range points {
		p := points[i]

		if !nextItem.canHave(count, unit, &points[i]) {
			nextItem.EndIndex = i
			nextItem.LastPoint = &points[i]
			nextItem.CalcultateSpeed()
			if stats, ok := w.Data.Details.StatsForRange(nextItem.StartIndex, nextItem.EndIndex); ok {
				nextItem.applyRangeStats(stats)
			}
			items = append(items, nextItem)
			nextItem = nextItem.createNext(&points[i])
			nextItem.StartIndex = i
		}

		nextItem.Distance += p.Distance
		nextItem.TotalDistance += p.Distance

		// m/s -> km/h, cut-off is speed less than 1 km/h
		if p.AverageSpeed()*3.6 >= 1.0 {
			nextItem.Duration += p.Duration
			nextItem.TotalDuration += p.Duration
		} else {
			nextItem.PauseDuration += p.Duration
		}
	}

	nextItem.EndIndex = len(points) - 1
	nextItem.LastPoint = &points[len(points)-1]

	if nextItem.FirstPoint != nil {
		nextItem.CalcultateSpeed()
		if stats, ok := w.Data.Details.StatsForRange(nextItem.StartIndex, nextItem.EndIndex); ok {
			nextItem.applyRangeStats(stats)
		}
		items = append(items, nextItem)
	}

	calculateBestAndWorst(items)

	return items
}

type WorkoutBreakdown struct {
	Unit  string          `json:"unit"`
	Items []BreakdownItem `json:"items"`
}

func (w *Workout) StatisticsPer(count float64, unit string) (WorkoutBreakdown, error) {
	wb := WorkoutBreakdown{Unit: unit}

	switch unit {
	case "m":
		wb.Items = w.statisticsWithUnit(count, "distance")
	case "km":
		wb.Items = w.statisticsWithUnit(count*templatehelpers.MeterPerKM, "distance")
	case "mi":
		wb.Items = w.statisticsWithUnit(count*templatehelpers.MeterPerMile, "distance")
	case "sec":
		wb.Items = w.statisticsWithUnit(count*float64(time.Second), "duration")
	case "min":
		wb.Items = w.statisticsWithUnit(count*float64(time.Minute), "duration")
	case "hour":
		wb.Items = w.statisticsWithUnit(count*float64(time.Hour), "duration")
	default:
		return wb, fmt.Errorf("unknown unit: %s", unit)
	}

	if len(wb.Items) == 0 {
		return wb, errors.New("no data")
	}

	return wb, nil
}
