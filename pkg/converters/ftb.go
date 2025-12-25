package converters

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"path"
	"strings"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
)

func ParseFTB(content []byte) ([]*database.Workout, error) {
	zipReader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, err
	}

	result := []*database.Workout{}

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

func readFtbXMLFile(zf *zip.File) ([]*database.Workout, error) {
	c, err := readFileFromZip(zf)
	if err != nil {
		return nil, err
	}

	data := &FitoTrackBackup{}
	if err := xml.Unmarshal(c, &data); err != nil {
		return nil, err
	}

	result := []*database.Workout{}

	for _, is := range data.IndoorWorkouts.IndoorWorkouts {
		result = append(result, convertToWorkout(is))
	}

	return result, nil
}

func convertToWorkout(iw indoorWorkout) *database.Workout {
	wd := database.WorkoutData{
		Name:             iw.ExportFileName,
		Type:             iw.WorkoutType,
		Start:            iw.StartTime(),
		Stop:             iw.EndTime(),
		TotalDuration:    time.Duration(iw.Duration * int64(time.Millisecond)),
		TotalRepetitions: iw.Repetitions,
	}

	name := strings.TrimSuffix(path.Base(wd.Name), path.Ext(wd.Name))
	w := &database.Workout{
		Data: &database.MapData{WorkoutData: wd},
		Date: wd.Start,
		Name: name,
	}

	w.UpdateAverages()
	w.UpdateExtraMetrics()

	return w
}
