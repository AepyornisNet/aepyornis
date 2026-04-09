package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cast"
	"github.com/tkrajina/gpxgo/gpx"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type ExtraMetrics map[string]float64

func (ExtraMetrics) GormDataType() string {
	return "json"
}

// Scan scan value into Jsonb, implements sql.Scanner interface
func (em *ExtraMetrics) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}

	result := ExtraMetrics{}
	err := json.Unmarshal(bytes, &result)
	*em = result
	return err
}

// Value return json value, implement driver.Valuer interface
func (em ExtraMetrics) Value() (driver.Value, error) {
	if len(em) == 0 {
		return nil, nil
	}
	return json.Marshal(em)
}

func (em ExtraMetrics) Set(key string, value float64) {
	em[key] = value
}

func (em ExtraMetrics) Get(key string) float64 {
	return em[key]
}

func (em ExtraMetrics) ParseGPXExtensions(extension gpx.Extension) {
	for _, n := range extension.Nodes {
		if key, value := getGPXExtensionKeyValue(&n); key != "" {
			em.Set(key, value)
		}

		for _, subN := range n.Nodes {
			if key, value := getGPXExtensionKeyValue(&subN); key != "" {
				em.Set(key, value)
			}
		}
	}
}

func (ExtraMetrics) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "mysql", "sqlite":
		return "JSON"
	case "postgres":
		return "JSONB"
	}
	return ""
}

func getGPXExtensionKeyValue(n *gpx.ExtensionNode) (string, float64) {
	name := standardExtensionName(n.XMLName.Local)

	if data, err := cast.ToFloat64E(n.Data); err == nil {
		return name, data
	}

	return "", 0
}

func standardExtensionName(name string) string {
	name = strings.ToLower(name)
	name = strings.TrimPrefix(name, "gpxdata:")
	name = strings.TrimPrefix(name, "ns3:")

	switch name {
	case "course":
		return "heading"
	case "hacc": // horizontal accuracy estimate [mm]
		return "horizontal-accuracy"
	case "vacc": // vertical accuracy estimate [mm]
		return "vertical-accuracy"
	case "hr", "heartrate":
		return "heart-rate"
	case "cad":
		return "cadence"
	case "atemp", "temp":
		return "temperature"
	default:
		return name
	}
}
