package gogis

import (
	"bytes"
	"database/sql/driver"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
)

// GeometryType represents the type of a geometry
type GeometryType uint64

const (
	GeometryTypePoint              GeometryType = 1
	GeometryTypeLineString         GeometryType = 2
	GeometryTypePolygon            GeometryType = 3
	GeometryTypeGeometryCollection GeometryType = 7
)

// Geometry is an interface for all geometry types
type Geometry interface {
	String() string
}

// GeometryCollection represents a collection of heterogeneous geometries
type GeometryCollection struct {
	Geometries []Geometry `json:"geometries"`
}

// String returns the WKT (Well Known Text) representation
func (gc *GeometryCollection) String() string {
	if len(gc.Geometries) == 0 {
		return "SRID=4326;GEOMETRYCOLLECTION EMPTY"
	}

	geoms := make([]string, len(gc.Geometries))
	for i, g := range gc.Geometries {
		// Remove SRID prefix from individual geometries
		str := g.String()
		if idx := strings.Index(str, ";"); idx != -1 {
			str = str[idx+1:]
		}
		geoms[i] = str
	}

	return fmt.Sprintf("SRID=4326;GEOMETRYCOLLECTION(%s)", strings.Join(geoms, ","))
}

// Scan implements the sql.Scanner interface
func (gc *GeometryCollection) Scan(val any) error {
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
		return fmt.Errorf("cannot scan type %T into GeometryCollection", val)
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

	if wkbGeometryType != uint64(GeometryTypeGeometryCollection) {
		return fmt.Errorf("invalid geometry type for GeometryCollection: %d", wkbGeometryType)
	}

	var numGeometries uint32
	if err := binary.Read(r, byteOrder, &numGeometries); err != nil {
		return err
	}

	gc.Geometries = make([]Geometry, 0, numGeometries)

	for i := uint32(0); i < numGeometries; i++ {
		// Read the byte order for this geometry
		var geomByteOrder uint8
		if err := binary.Read(r, binary.LittleEndian, &geomByteOrder); err != nil {
			return err
		}

		var geomOrder binary.ByteOrder
		switch geomByteOrder {
		case 0:
			geomOrder = binary.BigEndian
		case 1:
			geomOrder = binary.LittleEndian
		default:
			return fmt.Errorf("invalid byte order %d for geometry %d", geomByteOrder, i)
		}

		var geomType uint64
		if err := binary.Read(r, geomOrder, &geomType); err != nil {
			return err
		}

		switch GeometryType(geomType) {
		case GeometryTypePoint:
			var p Point
			if err := binary.Read(r, geomOrder, &p.Lng); err != nil {
				return err
			}
			if err := binary.Read(r, geomOrder, &p.Lat); err != nil {
				return err
			}
			gc.Geometries = append(gc.Geometries, &p)

		case GeometryTypeLineString:
			var numPoints uint32
			if err := binary.Read(r, geomOrder, &numPoints); err != nil {
				return err
			}

			ls := &LineString{Points: make([]Point, numPoints)}
			for j := uint32(0); j < numPoints; j++ {
				if err := binary.Read(r, geomOrder, &ls.Points[j].Lng); err != nil {
					return err
				}
				if err := binary.Read(r, geomOrder, &ls.Points[j].Lat); err != nil {
					return err
				}
			}
			gc.Geometries = append(gc.Geometries, ls)

		case GeometryTypePolygon:
			var numRings uint32
			if err := binary.Read(r, geomOrder, &numRings); err != nil {
				return err
			}

			poly := &Polygon{Rings: make([][]Point, numRings)}
			for j := uint32(0); j < numRings; j++ {
				var numPoints uint32
				if err := binary.Read(r, geomOrder, &numPoints); err != nil {
					return err
				}

				poly.Rings[j] = make([]Point, numPoints)
				for k := uint32(0); k < numPoints; k++ {
					if err := binary.Read(r, geomOrder, &poly.Rings[j][k].Lng); err != nil {
						return err
					}
					if err := binary.Read(r, geomOrder, &poly.Rings[j][k].Lat); err != nil {
						return err
					}
				}
			}
			gc.Geometries = append(gc.Geometries, poly)

		default:
			return fmt.Errorf("unsupported geometry type in collection: %d", geomType)
		}
	}

	return nil
}

// Value implements the driver.Valuer interface
func (gc GeometryCollection) Value() (driver.Value, error) {
	return gc.String(), nil
}
