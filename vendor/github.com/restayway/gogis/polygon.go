package gogis

import (
	"bytes"
	"database/sql/driver"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
)

// Polygon represents a spatial area with an outer boundary and optional holes
type Polygon struct {
	Rings [][]Point `json:"rings"` // First ring is outer boundary, others are holes
}

// Ensure Polygon implements Geometry interface
var _ Geometry = (*Polygon)(nil)

// String returns the WKT (Well Known Text) representation
func (p *Polygon) String() string {
	if len(p.Rings) == 0 {
		return "SRID=4326;POLYGON EMPTY"
	}

	rings := make([]string, len(p.Rings))
	for i, ring := range p.Rings {
		points := make([]string, len(ring))
		for j, pt := range ring {
			points[j] = fmt.Sprintf("%v %v", pt.Lng, pt.Lat)
		}
		rings[i] = fmt.Sprintf("(%s)", strings.Join(points, ","))
	}

	return fmt.Sprintf("SRID=4326;POLYGON(%s)", strings.Join(rings, ","))
}

// Scan implements the sql.Scanner interface
func (p *Polygon) Scan(val any) error {
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
		return fmt.Errorf("cannot scan type %T into Polygon", val)
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

	if wkbGeometryType != 3 {
		return fmt.Errorf("invalid geometry type for Polygon: %d", wkbGeometryType)
	}

	var numRings uint32
	if err := binary.Read(r, byteOrder, &numRings); err != nil {
		return err
	}

	p.Rings = make([][]Point, numRings)
	for i := uint32(0); i < numRings; i++ {
		var numPoints uint32
		if err := binary.Read(r, byteOrder, &numPoints); err != nil {
			return err
		}

		p.Rings[i] = make([]Point, numPoints)
		for j := uint32(0); j < numPoints; j++ {
			if err := binary.Read(r, byteOrder, &p.Rings[i][j].Lng); err != nil {
				return err
			}
			if err := binary.Read(r, byteOrder, &p.Rings[i][j].Lat); err != nil {
				return err
			}
		}
	}

	return nil
}

// Value implements the driver.Valuer interface
func (p Polygon) Value() (driver.Value, error) {
	return p.String(), nil
}
