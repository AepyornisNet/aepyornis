package converters

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"path"
	"strings"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
)

func ParseFTB(content []byte) ([]*model.Workout, error) {
	zipReader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, err
	}

	result := []*model.Workout{}

	// Read all the files from zip archive
	for _, zipFile := range zipReader.File {
		if zipFile.Name != "data.xml" {
			continue
		}

		gpx, err := readFtbXMLFile(zipFile)
		if err != nil {
			return nil, err
		}

		result = append(result, gpx...)
	}

	return result, nil
}

func readFtbXMLFile(zf *zip.File) ([]*model.Workout, error) {
	c, err := readFileFromZip(zf)
	if err != nil {
		return nil, err
	}

	data := &FitoTrackBackup{}
	if err := xml.Unmarshal(c, &data); err != nil {
		return nil, err
	}

	result := []*model.Workout{}

	for _, is := range data.IndoorWorkouts.IndoorWorkouts {
		result = append(result, convertToWorkout(is))
	}

	return result, nil
}

func convertToWorkout(iw indoorWorkout) *model.Workout {
	start := iw.StartTime()
	end := iw.EndTime()
	name := strings.TrimSuffix(path.Base(iw.ExportFileName), path.Ext(iw.ExportFileName))
	workoutType, found := model.WorkoutTypeFromData(iw.WorkoutType)
	customType := ""
	if !found {
		customType = iw.WorkoutType
	}
	w := &model.Workout{
		Data:             &model.WorkoutGeoMeta{},
		Date:             start,
		DateEnd:          end,
		Name:             name,
		Type:             workoutType,
		CustomType:       customType,
		TotalDuration:    time.Duration(iw.Duration * int64(time.Millisecond)),
		TotalRepetitions: iw.Repetitions,
	}

	w.UpdateAverages()
	w.UpdateExtraMetrics()

	return w
}
