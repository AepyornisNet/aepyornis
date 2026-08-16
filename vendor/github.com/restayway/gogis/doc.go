// Package gogis provides PostGIS geometry types for GORM.
//
// This package implements PostGIS geometric types that can be used with GORM
// for storing and querying spatial data in PostgreSQL databases with the PostGIS extension.
//
// # Supported Geometry Types
//
// The package supports all major PostGIS geometry types:
//   - Point: Represents a single location with longitude and latitude
//   - LineString: Represents a path as an ordered series of points
//   - Polygon: Represents an area with outer boundary and optional holes
//   - GeometryCollection: Represents a collection of heterogeneous geometries
//
// # Database Setup
//
// Before using this package, ensure your PostgreSQL database has the PostGIS extension enabled:
//
//	CREATE EXTENSION IF NOT EXISTS postgis;
//
// # GORM Integration
//
// All geometry types implement the sql.Scanner and driver.Valuer interfaces,
// making them compatible with GORM out of the box:
//
//	type Location struct {
//		ID       uint
//		Name     string
//		Point    gogis.Point    `gorm:"type:geometry(Point,4326)"`
//		Area     gogis.Polygon  `gorm:"type:geometry(Polygon,4326)"`
//		Path     gogis.LineString `gorm:"type:geometry(LineString,4326)"`
//	}
//
// # Coordinate System
//
// All geometries use SRID 4326 (WGS 84) coordinate system by default.
// Coordinates are stored as longitude (X) and latitude (Y) in decimal degrees.
//
// # Indexing
//
// For optimal performance with spatial queries, create spatial indexes:
//
//	CREATE INDEX idx_locations_point ON locations USING GIST (point);
//	CREATE INDEX idx_locations_area ON locations USING GIST (area);
//
// # Example Usage
//
//	// Create a location
//	location := Location{
//		Name: "Central Park",
//		Point: gogis.Point{
//			Lng: -73.965355, // Longitude (X)
//			Lat: 40.782865,  // Latitude (Y)
//		},
//	}
//
//	// Save to database
//	db.Create(&location)
//
//	// Query within distance
//	var nearby []Location
//	db.Where("ST_DWithin(point, ST_Point(?, ?), ?)", -73.965355, 40.782865, 0.01).Find(&nearby)
//
// # Well-Known Binary (WKB) Support
//
// All geometry types can parse and generate Well-Known Binary (WKB) format
// as used by PostGIS, supporting both little-endian and big-endian byte orders.
//
// # Well-Known Text (WKT) Support
//
// All geometry types provide String() methods that generate Well-Known Text (WKT)
// format with SRID specification for use in SQL queries.
package gogis
