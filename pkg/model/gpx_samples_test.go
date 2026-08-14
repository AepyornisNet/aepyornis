package model

import (
	"testing/fstest"
)

var gpxFS fstest.MapFS

func init() { //nolint:gochecknoinits
	populateGPXFS()
}

func populateGPXFS() {
	gpxFS = fstest.MapFS{}

	gpxFS["sample1.gpx"] = &fstest.MapFile{Data: []byte(GpxSample1)}
}
