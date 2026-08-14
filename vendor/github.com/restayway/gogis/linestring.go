package gogis

import (
	"bytes"
	"database/sql/driver"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
)

// LineString represents a path between locations as an ordered series of points.
//
// LineString implements the PostGIS LineString geometry type and can be used in GORM models
// to store paths, routes, boundaries, or any sequence of connected points. It uses the
// WGS 84 coordinate system (SRID 4326) and requires at least 2 points to form a valid line.
//
// Database Column Type:
//
//	geometry(LineString,4326)
//
// Example:
//
//	type Route struct {
//	    ID   uint
//	    Name string
//	    Path gogis.LineString `gorm:"type:geometry(LineString,4326)"`
//	}
//
//	route := Route{
//	    Name: "Broadway",
//	    Path: gogis.LineString{
//	        Points: []gogis.Point{
//	            {Lng: -73.989, Lat: 40.756}, // Times Square
//	            {Lng: -73.985, Lat: 40.758}, // North on Broadway
//	            {Lng: -73.981, Lat: 40.761}, // Further north
//	        },
//	    },
//	}
//
// Spatial Queries:
//
//	// Find routes that intersect with a polygon
//	db.Where("ST_Intersects(path, ?)", polygon.String()).Find(&routes)
//
//	// Find routes within distance of a point
//	db.Where("ST_DWithin(path, ST_Point(?, ?), ?)", lng, lat, 0.01).Find(&routes)
//
//	// Get length of routes in meters
//	db.Select("*, ST_Length(ST_Transform(path, 3857)) as length_meters").Find(&routes)
//
//	// Find routes that cross a specific line
//	db.Where("ST_Crosses(path, ?)", otherLine.String()).Find(&routes)
type LineString struct {
	Points []Point `json:"points"` // Ordered sequence of points forming the line
}

// Ensure LineString implements Geometry interface
var _ Geometry = (*LineString)(nil)

// String returns the Well-Known Text (WKT) representation of the LineString.
//
// The returned string includes the SRID and follows the format:
// "SRID=4326;LINESTRING(lng1 lat1,lng2 lat2,...)" for non-empty LineStrings
// or "SRID=4326;LINESTRING EMPTY" for empty LineStrings.
//
// Example output: "SRID=4326;LINESTRING(-73.989 40.756,-73.985 40.758,-73.981 40.761)"
func (ls *LineString) String() string {
	if len(ls.Points) == 0 {
		return "SRID=4326;LINESTRING EMPTY"
	}

	points := make([]string, len(ls.Points))
	for i, p := range ls.Points {
		points[i] = fmt.Sprintf("%v %v", p.Lng, p.Lat)
	}

	return fmt.Sprintf("SRID=4326;LINESTRING(%s)", strings.Join(points, ","))
}

// Scan implements the sql.Scanner interface for reading LineString data from the database.
//
// This method is called automatically by GORM when reading LineString values from
// PostGIS geometry columns. It parses Well-Known Binary (WKB) format data
// returned by PostGIS and populates the LineString's Points slice.
//
// The method supports both little-endian and big-endian WKB formats and
// validates that the geometry type is correct for LineString (type 2).
//
// Parameters:
//
//	val: The raw value from the database, typically a hex-encoded WKB string
//	     or []uint8 containing the hex-encoded WKB data
//
// Returns:
//
//	error: Any error encountered during parsing, or nil on success
func (ls *LineString) Scan(val any) error {
	if val == nil {
		return nil
	}

	var decode string
	switch v := val.(type) {
	case []uint8:
		decode = string(v)
	case string:
		decode = v
	default:
		return fmt.Errorf("cannot scan type %T into LineString", val)
	}

	b, err := hex.DecodeString(decode)
	if err != nil {
		return err
	}

	r := bytes.NewReader(b)

	var wkbByteOrder uint8
	if err := binary.Read(r, binary.LittleEndian, &wkbByteOrder); err != nil {
		return err
	}

	var byteOrder binary.ByteOrder
	switch wkbByteOrder {
	case 0:
		byteOrder = binary.BigEndian
	case 1:
		byteOrder = binary.LittleEndian
	default:
		return fmt.Errorf("invalid byte order %d", wkbByteOrder)
	}

	var wkbGeometryType uint32
	if err := binary.Read(r, byteOrder, &wkbGeometryType); err != nil {
		return err
	}

	// Handle EWKB format which includes SRID in the geometry type
	// LineString type can be 2 (WKB) or 0x20000002 (EWKB with SRID)
	geometryType := wkbGeometryType & 0x1FFFFFFF // Mask out SRID flag
	if geometryType != 2 {
		return fmt.Errorf("invalid geometry type for LineString: %d", wkbGeometryType)
	}

	// If EWKB format, skip the SRID
	if wkbGeometryType&0x20000000 != 0 {
		var srid uint32
		if err := binary.Read(r, byteOrder, &srid); err != nil {
			return err
		}
	}

	var numPoints uint32
	if err := binary.Read(r, byteOrder, &numPoints); err != nil {
		return err
	}

	ls.Points = make([]Point, numPoints)
	for i := uint32(0); i < numPoints; i++ {
		if err := binary.Read(r, byteOrder, &ls.Points[i].Lng); err != nil {
			return err
		}
		if err := binary.Read(r, byteOrder, &ls.Points[i].Lat); err != nil {
			return err
		}
	}

	return nil
}

// Value implements the driver.Valuer interface for writing LineString data to the database.
//
// This method is called automatically by GORM when saving LineString values to
// PostGIS geometry columns. It returns the Well-Known Text (WKT) representation
// of the LineString, which PostGIS can directly parse and store.
//
// Returns:
//
//	driver.Value: The WKT string representation of the LineString
//	error: Always nil for LineString (no validation errors possible)
//
// Example output: "SRID=4326;LINESTRING(-73.989 40.756,-73.985 40.758)"
func (ls LineString) Value() (driver.Value, error) {
	return ls.String(), nil
}
