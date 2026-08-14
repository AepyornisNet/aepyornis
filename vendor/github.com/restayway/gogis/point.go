package gogis

import (
	"bytes"
	"database/sql/driver"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// Point represents a geometric point in 2D space with longitude and latitude coordinates.
//
// Point implements the PostGIS Point geometry type and can be used in GORM models
// to store geographic locations. It uses the WGS 84 coordinate system (SRID 4326)
// with longitude (X) and latitude (Y) values in decimal degrees.
//
// Database Column Type:
//
//	geometry(Point,4326)
//
// Example:
//
//	type Location struct {
//	    ID    uint
//	    Name  string
//	    Point gogis.Point `gorm:"type:geometry(Point,4326)"`
//	}
//
//	location := Location{
//	    Name: "Statue of Liberty",
//	    Point: gogis.Point{
//	        Lng: -74.0445,  // Longitude (X coordinate)
//	        Lat: 40.6892,   // Latitude (Y coordinate)
//	    },
//	}
//
// Spatial Queries:
//
//	// Find points within 1km
//	db.Where("ST_DWithin(point, ST_Point(?, ?), ?)", lng, lat, 0.009).Find(&locations)
//
//	// Find nearest points
//	db.Order("ST_Distance(point, ST_Point(?, ?))").Limit(10).Find(&locations)
//
//	// Check if point is within polygon
//	db.Where("ST_Within(point, ?)", polygon.String()).Find(&locations)
type Point struct {
	Lng float64 `json:"lng"` // Longitude (X coordinate) in decimal degrees
	Lat float64 `json:"lat"` // Latitude (Y coordinate) in decimal degrees
}

// Ensure Point implements Geometry interface
var _ Geometry = (*Point)(nil)

// String returns the Well-Known Text (WKT) representation of the Point.
//
// The returned string includes the SRID (Spatial Reference System Identifier)
// and follows the format: "SRID=4326;POINT(longitude latitude)"
//
// Example output: "SRID=4326;POINT(-74.0445 40.6892)"
func (p *Point) String() string {
	return fmt.Sprintf("SRID=4326;POINT(%v %v)", p.Lng, p.Lat)
}

// Scan implements the sql.Scanner interface for reading Point data from the database.
//
// This method is called automatically by GORM when reading Point values from
// PostGIS geometry columns. It parses Well-Known Binary (WKB) format data
// returned by PostGIS and populates the Point's Lng and Lat fields.
//
// The method supports both little-endian and big-endian WKB formats and
// handles the complete WKB structure including byte order, geometry type,
// and coordinate data.
//
// Parameters:
//
//	val: The raw value from the database, typically a hex-encoded WKB string
//	     or []uint8 containing the hex-encoded WKB data
//
// Returns:
//
//	error: Any error encountered during parsing, or nil on success
func (p *Point) Scan(val any) error {
	if val == nil {
		return nil
	}
	var decode string
	uint8Val, ok := val.([]uint8)
	if ok {
		decode = string(uint8Val)
	} else {
		decode = val.(string)
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

	var wkbGeometryType uint64
	if err := binary.Read(r, byteOrder, &wkbGeometryType); err != nil {
		return err
	}

	if err := binary.Read(r, byteOrder, p); err != nil {
		return err
	}

	return nil
}

// Value implements the driver.Valuer interface for writing Point data to the database.
//
// This method is called automatically by GORM when saving Point values to
// PostGIS geometry columns. It returns the Well-Known Text (WKT) representation
// of the Point, which PostGIS can directly parse and store.
//
// Returns:
//
//	driver.Value: The WKT string representation of the Point
//	error: Always nil for Point (no validation errors possible)
//
// Example output: "SRID=4326;POINT(-74.0445 40.6892)"
func (p Point) Value() (driver.Value, error) {
	return p.String(), nil
}
