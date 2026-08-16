# GoGIS

[![Go Reference](https://pkg.go.dev/badge/github.com/restayway/gogis.svg)](https://pkg.go.dev/github.com/restayway/gogis)
[![Go Report Card](https://goreportcard.com/badge/github.com/restayway/gogis)](https://goreportcard.com/report/github.com/restayway/gogis)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**GoGIS** is a comprehensive Go library that provides PostGIS geometry types for GORM, enabling seamless integration of spatial data with your Go applications.

## Features

✨ **Complete PostGIS Integration** - Full support for Point, LineString, Polygon, and GeometryCollection types  
🚀 **GORM Compatible** - Implements `sql.Scanner` and `driver.Valuer` for automatic ORM integration  
🗺️ **WGS 84 Support** - Uses SRID 4326 coordinate system for global geographic data  
📐 **Well-Known Formats** - Supports both WKB (Well-Known Binary) and WKT (Well-Known Text)  
⚡ **High Performance** - Optimized for spatial queries with proper indexing support  
🧪 **Thoroughly Tested** - Comprehensive test suite with >95% coverage  
📚 **Well Documented** - Complete API documentation and usage examples  

## Installation

```bash
go get github.com/restayway/gogis
```

## Prerequisites

- **Go 1.19+**
- **PostgreSQL 12+** with **PostGIS 3.0+**
- **GORM v2**

### Database Setup

Enable PostGIS extension in your PostgreSQL database:

```sql
CREATE EXTENSION IF NOT EXISTS postgis;
```

## Quick Start

```go
package main

import (
    "github.com/restayway/gogis"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

type Location struct {
    ID    uint        `gorm:"primaryKey"`
    Name  string      `gorm:"not null"`
    Point gogis.Point `gorm:"type:geometry(Point,4326)"`
}

func main() {
    // Connect to database
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        panic(err)
    }

    // Auto-migrate
    db.AutoMigrate(&Location{})

    // Create location
    location := Location{
        Name: "Statue of Liberty",
        Point: gogis.Point{
            Lng: -74.0445, // Longitude
            Lat: 40.6892,  // Latitude
        },
    }
    db.Create(&location)

    // Spatial query - find locations within 1km
    var nearby []Location
    db.Where("ST_DWithin(point, ST_Point(?, ?), ?)", 
        -74.0445, 40.6892, 0.009).Find(&nearby)
}
```

## Supported Geometry Types

### Point
Represents a single location with longitude and latitude coordinates.

```go
type Location struct {
    ID    uint        `gorm:"primaryKey"`
    Name  string
    Point gogis.Point `gorm:"type:geometry(Point,4326)"`
}

location := Location{
    Name: "Central Park",
    Point: gogis.Point{
        Lng: -73.9665, // Longitude (X)
        Lat: 40.7812,  // Latitude (Y)
    },
}
```

### LineString
Represents paths, routes, or any sequence of connected points.

```go
type Route struct {
    ID   uint             `gorm:"primaryKey"`
    Name string
    Path gogis.LineString `gorm:"type:geometry(LineString,4326)"`
}

route := Route{
    Name: "Broadway Walk",
    Path: gogis.LineString{
        Points: []gogis.Point{
            {Lng: -73.9857, Lat: 40.7580}, // Times Square
            {Lng: -73.9857, Lat: 40.7614}, // North on Broadway
            {Lng: -73.9855, Lat: 40.7682}, // Further north
        },
    },
}
```

### Polygon
Represents areas with outer boundaries and optional holes.

```go
type Region struct {
    ID   uint          `gorm:"primaryKey"`
    Name string
    Area gogis.Polygon `gorm:"type:geometry(Polygon,4326)"`
}

region := Region{
    Name: "Central Park",
    Area: gogis.Polygon{
        Rings: [][]gogis.Point{
            { // Outer boundary
                {Lng: -73.9812, Lat: 40.7644},
                {Lng: -73.9734, Lat: 40.7644},
                {Lng: -73.9734, Lat: 40.7947},
                {Lng: -73.9812, Lat: 40.7947},
                {Lng: -73.9812, Lat: 40.7644}, // Close the ring
            },
            // Optional: holes can be added as additional rings
        },
    },
}
```

### GeometryCollection
Represents collections of heterogeneous geometries.

```go
type Place struct {
    ID         uint                      `gorm:"primaryKey"`
    Name       string
    Geometries gogis.GeometryCollection `gorm:"type:geometry(GeometryCollection,4326)"`
}

place := Place{
    Name: "Campus",
    Geometries: gogis.GeometryCollection{
        Geometries: []gogis.Geometry{
            &gogis.Point{Lng: -73.9550, Lat: 40.8050},      // Building
            &gogis.LineString{Points: walkwayPoints},        // Walkway
            &gogis.Polygon{Rings: campusAreaRings},          // Campus area
        },
    },
}
```

## Common Spatial Queries

### Distance-Based Queries

```go
// Find locations within 1km of a point
var locations []Location
db.Where("ST_DWithin(point, ST_Point(?, ?), ?)", 
    lng, lat, 0.009).Find(&locations)

// Find nearest 10 locations
db.Order("ST_Distance(point, ST_Point(?, ?))").
    Limit(10).Find(&locations, lng, lat)
```

### Containment Queries

```go
// Find points within a polygon
var pointsInside []Location
db.Where("ST_Contains(?, point)", polygon.String()).Find(&pointsInside)

// Find polygons containing a point
var regions []Region
db.Where("ST_Contains(area, ST_Point(?, ?))", lng, lat).Find(&regions)
```

### Intersection Queries

```go
// Find routes that intersect with an area
var routes []Route
db.Where("ST_Intersects(path, ?)", polygon.String()).Find(&routes)

// Find overlapping regions
var overlapping []Region
db.Where("ST_Overlaps(area, ?)", otherPolygon.String()).Find(&overlapping)
```

### Area and Length Calculations

```go
// Calculate polygon areas in square meters
db.Select("*, ST_Area(ST_Transform(area, 3857)) as area_meters").Find(&regions)

// Calculate route lengths in meters
db.Select("*, ST_Length(ST_Transform(path, 3857)) as length_meters").Find(&routes)
```

## Performance Optimization

### Spatial Indexes

Create spatial indexes for optimal query performance:

```sql
-- For points
CREATE INDEX idx_locations_point ON locations USING GIST (point);

-- For linestrings
CREATE INDEX idx_routes_path ON routes USING GIST (path);

-- For polygons  
CREATE INDEX idx_regions_area ON regions USING GIST (area);

-- For geometry collections
CREATE INDEX idx_places_geometries ON places USING GIST (geometries);
```

### Query Optimization Tips

1. **Use ST_DWithin instead of ST_Distance with WHERE clauses**
2. **Create compound indexes for frequently filtered queries**
3. **Use ST_Transform for accurate distance/area calculations**
4. **Consider partial indexes for large datasets**

See [INDEXING.md](INDEXING.md) for detailed performance optimization guide.

## Examples

Comprehensive examples are available in the [`examples/`](examples/) directory:

- **[basic_usage/](examples/basic_usage/)** - Point operations and spatial queries
- **[linestring_example/](examples/linestring_example/)** - Routes and path analysis
- **[polygon_example/](examples/polygon_example/)** - Area operations and containment
- **[geometry_collection_example/](examples/geometry_collection_example/)** - Complex geometries

Run examples:
```bash
cd examples/basic_usage && go run main.go
cd examples/linestring_example && go run main.go
cd examples/polygon_example && go run main.go
cd examples/geometry_collection_example && go run main.go
```

## Testing

### Run Unit Tests
```bash
go test -v
```

### Run Integration Tests (requires PostGIS)
```bash
# Set up test database
createdb gogis_test
psql gogis_test -c "CREATE EXTENSION postgis;"

# Run integration tests
go test -tags=integration -v
```

### Run Benchmarks
```bash
go test -bench=. -benchmem
```

## API Documentation

Full API documentation is available at [pkg.go.dev](https://pkg.go.dev/github.com/restayway/gogis).

## Migration Guide

### From Other Libraries

If migrating from other PostGIS Go libraries:

1. **Update imports**: Change to `github.com/restayway/gogis`
2. **Update struct tags**: Use `gorm:"type:geometry(Type,4326)"`
3. **Update coordinate order**: GoGIS uses `{Lng, Lat}` (longitude, latitude)
4. **Update method calls**: Check method names in documentation

### Breaking Changes

See [CHANGELOG.md](CHANGELOG.md) for detailed migration information between versions.

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Quick Contribution Guide

1. Fork the repository
2. Create a feature branch
3. Write tests for your changes
4. Ensure all tests pass
5. Submit a pull request

## Roadmap

- [ ] **3D Geometry Support** - PointZ, LineStringZ, PolygonZ
- [ ] **Additional Geometry Types** - MultiPoint, MultiLineString, MultiPolygon
- [ ] **Custom SRID Support** - Beyond SRID 4326
- [ ] **Spatial Functions** - Built-in Go implementations of common operations
- [ ] **GeoJSON Integration** - Direct GeoJSON marshaling/unmarshaling
- [ ] **Performance Optimizations** - Further query performance improvements

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- **PostGIS Team** - For the excellent spatial database extension
- **GORM Team** - For the fantastic ORM library
- **Go Community** - For the great language and ecosystem

## Support

- 📖 **Documentation**: [pkg.go.dev](https://pkg.go.dev/github.com/restayway/gogis)
- 🐛 **Issues**: [GitHub Issues](https://github.com/restayway/gogis/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/restayway/gogis/discussions)
- 📧 **Email**: For security issues

---

**GoGIS** - Bringing PostGIS spatial capabilities to the Go ecosystem with GORM integration. Build location-aware applications with confidence! 🌍