package model

import (
	"crypto/sha256"
	"errors"
	"path/filepath"
	"strings"

	"github.com/codingsince1985/geo-golang"
	"github.com/restayway/gogis"
	"github.com/tkrajina/gpxgo/gpx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RouteSegmentDifficulty string

const (
	RouteSegmentDifficultyEasy      RouteSegmentDifficulty = "easy"
	RouteSegmentDifficultyModerate  RouteSegmentDifficulty = "moderate"
	RouteSegmentDifficultyDifficult RouteSegmentDifficulty = "difficult"
)

func (d RouteSegmentDifficulty) IsValid() bool {
	switch d {
	case "", RouteSegmentDifficultyEasy, RouteSegmentDifficultyModerate, RouteSegmentDifficultyDifficult:
		return true
	}
	return false
}

type RoutSegmentCreationParams struct {
	Name        string                 `form:"name" json:"name"`
	Start       int                    `form:"start" json:"start"`
	End         int                    `form:"end" json:"end"`
	Category    string                 `form:"category" json:"category"`
	Visibility  WorkoutVisibility      `form:"visibility" json:"visibility"`
	Description string                 `form:"description" json:"description"`
	Difficulty  RouteSegmentDifficulty `form:"difficulty" json:"difficulty"`
}

func (rscp *RoutSegmentCreationParams) Filename() string {
	if rscp.Name == "" {
		return "noname.gpx"
	}

	if strings.HasSuffix(rscp.Name, ".gpx") {
		return rscp.Name
	}

	return rscp.Name + ".gpx"
}

type RouteSegment struct {
	Model
	ProfileID   uint64                 `gorm:"not null;index" json:"profile_id"` // Owner of the route segment
	Profile     *Profile               `json:"profile,omitempty"`
	Category    string                 `json:"category"`                                    // Workout type this is intended for
	Visibility  WorkoutVisibility      `gorm:"not null;default:'public'" json:"visibility"` // Public, followers, private
	Description string                 `json:"description"`                                 // Description of the route segment
	Difficulty  RouteSegmentDifficulty `json:"difficulty"`                                  // easy, moderate, difficult

	GeoAddress    *geo.Address `gorm:"serializer:json" json:"geoAddress"` // The address of the workout
	Name          string       `gorm:"not null" json:"name"`              // The name of the workout
	Notes         string       `json:"notes"`                             // The notes associated with the workout, in markdown
	AddressString string       `json:"addressString"`                     // The generic location of the workout
	Filename      string       `json:"filename"`                          // The filename of the file

	Points gogis.LineString `gorm:"type:geometry(LineString,4326)" json:"points"` // The GPS points of the route segment

	Content             []byte               `gorm:"type:bytes" json:"content"`            // The file content
	Checksum            []byte               `gorm:"not null;uniqueIndex" json:"checksum"` // The checksum of the content
	RouteSegmentMatches []*RouteSegmentMatch `json:"routeSegmentMatches"`                  // The matches of the route segment
	Center              MapCenter            `gorm:"serializer:json" json:"center"`        // The center of the workout (in coordinates)

	TotalDistance float64 `json:"totalDistance"` // The total distance of the workout
	MinElevation  float64 `json:"minElevation"`  // The minimum elevation of the workout
	MaxElevation  float64 `json:"maxElevation"`  // The maximum elevation of the workout
	TotalUp       float64 `json:"totalUp"`       // The total distance up of the workout
	TotalDown     float64 `json:"totalDown"`     // The total distance down of the workout
	Bidirectional bool    `json:"bidirectional"` // Whether the route segment is bidirectional
	Circular      bool    `json:"circular"`      // Whether the route segment is circular

	Dirty bool `json:"dirty"` // Whether the route segment should be recalculated
}

func (rs *RouteSegment) HasFile() bool {
	return rs.Filename != "" && rs.Content != nil
}

func NewRouteSegment(notes string, filename string, content []byte) (*RouteSegment, error) {
	filename = filepath.Base(filename)
	name := strings.TrimSuffix(filename, ".gpx")

	h := sha256.New()
	h.Write(content)

	rs := &RouteSegment{
		Name:       name,
		Notes:      notes,
		Visibility: WorkoutVisibilityPublic,
		Dirty:      true,

		Content:  content,
		Checksum: h.Sum(nil),
		Filename: filename,
	}

	if err := rs.UpdateFromContent(); err != nil {
		return nil, err
	}

	return rs, nil
}

func RouteSegmentFromPoints(workout *Workout, params *RoutSegmentCreationParams) ([]byte, error) {
	points := workout.Records[params.Start-1 : params.End-1]

	s := gpx.GPXTrackSegment{}

	for _, p := range points {
		gpxPoint := gpx.Point{
			Latitude:  p.Lat(),
			Longitude: p.Lng(),
			Elevation: *gpx.NewNullableFloat64(p.ExtraMetrics.Get("elevation")),
		}

		pt := gpx.GPXPoint{Point: gpxPoint}
		s.AppendPoint(&pt)
	}

	newFile := &gpx.GPX{
		Creator: "Workout Tracker",
		Tracks:  []gpx.GPXTrack{{Segments: []gpx.GPXTrackSegment{s}}},
	}

	content, err := newFile.ToXml(gpx.ToXmlParams{Version: "1.1", Indent: true})
	if err != nil {
		return nil, err
	}

	return content, nil
}

func (rs *RouteSegment) UpdateFromContent() error {
	if WorkoutParser == nil {
		return ErrWorkoutParserMissing
	}

	parsed, err := WorkoutParser(rs.Filename, rs.Content)
	if err != nil {
		return err
	}

	if len(parsed) == 0 || parsed[0] == nil || parsed[0].Data == nil {
		return errors.New("route segment parse returned no data")
	}

	stats := parsed[0].Stats
	if stats == nil {
		stats = &WorkoutStats{}
	}

	rs.TotalDistance = parsed[0].TotalDistance
	rs.MinElevation = stats.MinElevation
	rs.MaxElevation = stats.MaxElevation
	rs.TotalUp = stats.TotalUp
	rs.TotalDown = stats.TotalDown

	records := parsed[0].Records
	pts := make([]gogis.Point, 0, len(records))
	for _, r := range records {
		if r.Point != nil && (r.Point.Lat != 0 || r.Point.Lng != 0) {
			pts = append(pts, *r.Point)
		}
	}
	rs.Points = gogis.LineString{Points: pts}

	// Detect whether the route is circular so matching can wrap around the end of the track.
	if len(rs.Points.Points) > 1 {
		first := rs.Points.Points[0]
		last := rs.Points.Points[len(rs.Points.Points)-1]
		rs.Circular = PointDistance(first, last) <= MaxDeltaMeter
	}

	return nil
}

func (rs *RouteSegment) Delete(db *gorm.DB) error {
	return db.Select(clause.Associations).Delete(rs).Error
}

func (rs *RouteSegment) Create(db *gorm.DB) error {
	if rs.Content == nil {
		return ErrInvalidData
	}

	if rs.ProfileID == 0 {
		var firstProfile Profile
		if err := db.Order("id ASC").First(&firstProfile).Error; err == nil {
			rs.ProfileID = firstProfile.ID
		} else {
			defaultProfile := Profile{Username: "default", DisplayName: "Default"}
			if err := db.Create(&defaultProfile).Error; err == nil {
				rs.ProfileID = defaultProfile.ID
			}
		}
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("RouteSegmentMatches").Create(rs).Error; err != nil {
			return err
		}

		if rs.RouteSegmentMatches != nil {
			if err := replaceRouteSegmentMatches(tx, rs.ID, rs.RouteSegmentMatches); err != nil {
				return err
			}
		}

		return nil
	})
}

func (rs *RouteSegment) Save(db *gorm.DB) error {
	if rs.Content == nil {
		return ErrInvalidData
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if rs.RouteSegmentMatches != nil {
			if err := replaceRouteSegmentMatches(tx, rs.ID, rs.RouteSegmentMatches); err != nil {
				return err
			}
		}

		return tx.Omit("RouteSegmentMatches").Save(rs).Error
	})
}

func replaceRouteSegmentMatches(tx *gorm.DB, routeSegmentID uint64, matches []*RouteSegmentMatch) error {
	if err := tx.Where("route_segment_id = ?", routeSegmentID).Delete(&RouteSegmentMatch{}).Error; err != nil {
		return err
	}

	if len(matches) == 0 {
		return nil
	}

	rows := make([]*RouteSegmentMatch, 0, len(matches))
	for _, m := range matches {
		if m == nil || m.WorkoutID == 0 {
			continue
		}

		rows = append(rows, &RouteSegmentMatch{
			RouteSegmentID: routeSegmentID,
			WorkoutID:      m.WorkoutID,
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

func (rs *RouteSegment) Address() string {
	if rs.AddressString != "" && rs.AddressString != UnknownLocation {
		return rs.AddressString
	}

	if rs.GeoAddress != nil {
		return rs.GeoAddress.FormattedAddress
	}

	return ""
}
