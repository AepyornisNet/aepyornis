package converters

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
	"github.com/tkrajina/gpxgo/gpx"
)

var (
	ErrUnsupportedFile = errors.New("unsupported file")
	SupportedFileTypes = []string{".fit", ".ftb", ".gpx", ".tcx", ".zip"}
)

type (
	parserFunc func(content []byte) (*gpx.GPX, error)
)

func init() {
	database.WorkoutParser = ParseCollection
}

func Parse(filename string, content []byte) (*database.Workout, error) {
	c, err := ParseCollection(filename, content)
	if err != nil {
		return nil, err
	}

	if len(c) == 0 {
		return nil, nil
	}

	return c[0], nil
}

func ParseCollection(filename string, content []byte) ([]*database.Workout, error) {
	if filename == "" {
		// Assume GPX when filename is empty
		return parseSingle(ParseGPX, "gpx", "", content)
	}

	basename := path.Base(filename)

	c, err := parseContent(basename, content)
	if err != nil {
		return nil, err
	}

	for _, w := range c {
		ensureWorkoutName(w, basename)
	}

	return c, nil
}

func parseContent(filename string, content []byte) ([]*database.Workout, error) {
	suffix := strings.ToLower(path.Ext(filename))

	switch suffix {
	case ".gpx":
		return parseSingle(ParseGPX, "gpx", filename, content)
	case ".fit":
		return ParseFit(content, filename)
	case ".tcx":
		return parseSingle(ParseTCX, "tcx", filename, content)
	case ".zip":
		return ParseZip(content)
	case ".ftb":
		return ParseFTB(content)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFile, filename)
	}
}

func parseSingle(f parserFunc, fileType string, filename string, content []byte) ([]*database.Workout, error) {
	g, err := f(content)
	if err != nil {
		return nil, err
	}

	if g == nil {
		return nil, nil
	}

	return []*database.Workout{workoutFromGPX(g, filename, fileType, content)}, nil
}

func workoutFromGPX(g *gpx.GPX, filename string, fileType string, content []byte) *database.Workout {
	data := database.MapDataFromGPX(g)
	if data == nil {
		data = &database.MapData{}
	}

	w := &database.Workout{
		Data: data,
		Name: data.WorkoutData.Name,
	}

	if date := database.GPXDate(g); date != nil {
		w.Date = *date
	}

	setContentAndName(w, filename, fileType, content)
	w.UpdateAverages()
	w.UpdateExtraMetrics()

	return w
}

func ensureWorkoutName(w *database.Workout, basename string) {
	if w == nil || w.Name != "" {
		return
	}

	if basename == "" {
		basename = "workout"
	}

	w.Name = strings.TrimSuffix(basename, path.Ext(basename))
}

func setContentAndName(w *database.Workout, filename string, fileType string, content []byte) {
	ext := strings.TrimPrefix(path.Ext(filename), ".")
	name := strings.TrimSuffix(path.Base(filename), path.Ext(filename))

	if name == "" {
		name = w.Name
	}

	if name == "" {
		name = "workout"
	}

	if ext == "" {
		ext = strings.TrimPrefix(fileType, ".")
	}

	finalName := name
	if ext != "" {
		finalName += "." + ext
	}

	if w.Name == "" {
		w.Name = name
	}

	w.SetContent(finalName, content)
}
