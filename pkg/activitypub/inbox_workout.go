package activitypub

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/jovandeginste/workout-tracker/v2/pkg/converters"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
)

// InboxWorkoutRepository handles persistence of workouts received via ActivityPub.
type InboxWorkoutRepository interface {
	CreateExternalWorkout(workout *model.Workout) error
	ExternalWorkoutExists(objectIRI string) (bool, error)
}

// isCreateWorkoutActivity returns true when the Create activity wraps a
// WorkoutNote (identified by the presence of a workoutSport field) that is
// *not* a reply (no inReplyTo).
func isCreateWorkoutActivity(rawPayload []byte) bool {
	if len(rawPayload) == 0 {
		return false
	}

	note, err := parseWorkoutNoteFromRawActivity(rawPayload)
	if err != nil || note == nil {
		return false
	}

	// Must have a sport to be considered a workout note
	if note.WorkoutSport == "" {
		return false
	}

	// Replies are handled separately
	if note.InReplyTo != "" {
		return false
	}

	return true
}

func handleCreateWorkoutActivity(ctx InboxHandlerContext) error {
	if ctx.RequestingActor == nil {
		return errors.New("requesting actor invalid")
	}

	note, err := parseWorkoutNoteFromRawActivity(ctx.RawPayload)
	if err != nil {
		return err
	}

	if note == nil || note.WorkoutSport == "" {
		return nil
	}

	objectIRI := note.ID.String()
	if objectIRI == "" {
		objectIRI = itemIRIString(note.URL)
	}

	if objectIRI == "" {
		return nil
	}

	// Deduplicate: skip if we already have this external workout
	exists, err := ctx.WorkoutRepo.ExternalWorkoutExists(objectIRI)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	actorIRI := ctx.RequestingActor.ID.String()

	// Cache actor profile
	actorName := ""
	if ctx.RequestingActor.Name != nil {
		actorName = ctx.RequestingActor.Name.String()
	}

	avatarURL := ActorIconIRI(ctx.RequestingActor)
	CacheActorProfile(actorIRI, actorName, avatarURL)

	// Build the workout from the WorkoutNote fields
	workout := workoutFromNote(note, actorIRI, objectIRI)

	// Try to download and parse the FIT file for detailed track data
	if note.WorkoutFitFile != "" {
		if fitWorkout, fitErr := downloadAndParseFIT(context.Background(), string(note.WorkoutFitFile)); fitErr == nil && fitWorkout != nil {
			mergeTrackDataFromFIT(workout, fitWorkout)
		}
	}

	// Extract image attachments from the note
	extractImageAttachments(workout, note)

	return ctx.WorkoutRepo.CreateExternalWorkout(workout)
}

// workoutFromNote creates a Workout from the fields in a WorkoutNote.
func workoutFromNote(note *WorkoutNote, actorIRI, objectIRI string) *model.Workout {
	workoutType := resolveWorkoutType(note.WorkoutSport)
	customType := ""

	if workoutType == model.WorkoutTypeOther {
		customType = note.WorkoutSport
	}

	published := time.Now().UTC()
	if !note.Published.IsZero() {
		published = note.Published
	}

	name := ""
	if note.Content != nil {
		name = note.Content.String()
		// Only use first line as name
		if idx := strings.IndexAny(name, "\n\r"); idx >= 0 {
			name = name[:idx]
		}
	}

	if name == "" {
		name = note.WorkoutSport
	}

	w := &model.Workout{
		Date:              published,
		Name:              name,
		Type:              workoutType,
		CustomType:        customType,
		Visibility:        model.WorkoutVisibilityFollowers,
		ActorIRI:          &actorIRI,
		ExternalObjectIRI: &objectIRI,
		Data: &model.MapData{
			AddressString: note.WorkoutLocation,
			Creator:       "activitypub",
			WorkoutData: model.WorkoutData{
				TotalDistance:   note.WorkoutDistance,
				TotalDistance2D: note.WorkoutDistance2D,
				TotalDuration:  time.Duration(note.WorkoutDuration) * time.Second,
				PauseDuration:  time.Duration(note.WorkoutPauseDuration) * time.Second,
				TotalRepetitions: note.WorkoutRepetitions,
				TotalWeight:    note.WorkoutWeight,
				WorkoutStats: model.WorkoutStats{
					TotalUp:             note.WorkoutElevationGain,
					TotalDown:           note.WorkoutElevationLoss,
					AverageSpeed:        note.WorkoutAverageSpeed,
					AverageSpeedNoPause: note.WorkoutAverageSpeedMove,
					MaxSpeed:            note.WorkoutMaxSpeed,
					AverageCadence:      note.WorkoutAverageCadence,
					MaxCadence:          note.WorkoutMaxCadence,
					AverageHeartRate:    note.WorkoutAverageHeartRate,
					MaxHeartRate:        note.WorkoutMaxHeartRate,
					AveragePower:        note.WorkoutAveragePower,
					MaxPower:            note.WorkoutMaxPower,
				},
			},
		},
	}

	return w
}

// mergeTrackDataFromFIT enriches the workout with detailed track data parsed
// from the FIT file, but keeps the summary stats from the note as authoritative.
func mergeTrackDataFromFIT(target *model.Workout, fitWorkout *model.Workout) {
	if target == nil || fitWorkout == nil {
		return
	}

	if !fitWorkout.Date.IsZero() {
		target.Date = fitWorkout.Date
	}

	if fitWorkout.Data == nil {
		return
	}

	if target.Data == nil {
		target.Data = &model.MapData{}
	}

	// Preserve existing summary stats from the WorkoutNote, merge in details
	target.Data.MergeNonZero(fitWorkout.Data.WorkoutData)
	target.Data.Center = fitWorkout.Data.Center
	target.Data.Details = fitWorkout.Data.Details
	target.Data.Climbs = fitWorkout.Data.Climbs
	target.Data.Laps = fitWorkout.Data.Laps

	if fitWorkout.GPX != nil {
		target.GPX = fitWorkout.GPX
	}
}

// extractImageAttachments stores image attachment references from the note
// as external URL attachments on the workout.
func extractImageAttachments(workout *model.Workout, note *WorkoutNote) {
	if workout == nil || note == nil || vocab.IsNil(note.Attachment) {
		return
	}

	order := 0

	_ = vocab.OnItemCollection(note.Attachment, func(items *vocab.ItemCollection) error {
		for _, item := range *items {
			_ = vocab.OnObject(item, func(obj *vocab.Object) error {
				if !vocab.ImageType.Match(obj.Type) {
					return nil
				}

				imageURL := itemIRIString(obj.URL)
				if imageURL == "" {
					imageURL = obj.ID.String()
				}

				if imageURL == "" {
					return nil
				}

				filename := ""
				if obj.Name != nil {
					filename = obj.Name.String()
				}

				if filename == "" {
					filename = "image"
				}

				contentType := string(obj.MediaType)
				if contentType == "" {
					contentType = "image/png"
				}

				kind := model.WorkoutAttachmentKindRouteImage
				if order > 0 {
					kind = "image"
				}

				workout.Attachments = append(workout.Attachments, model.WorkoutAttachment{
					Kind:        kind,
					Filename:    filename,
					ContentType: contentType,
					ExternalURL: &imageURL,
					SortOrder:   order,
				})
				order++

				return nil
			})
		}

		return nil
	})
}

// downloadAndParseFIT fetches a FIT file from the given URL and parses it.
func downloadAndParseFIT(ctx context.Context, fitURL string) (*model.Workout, error) {
	if fitURL == "" {
		return nil, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fitURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", FitMIMEType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to download FIT file: " + resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, nil
	}

	workouts, err := converters.ParseFit(data, "workout.fit")
	if err != nil {
		return nil, err
	}

	if len(workouts) == 0 {
		return nil, nil
	}

	return workouts[0], nil
}

// resolveWorkoutType maps a sport string from ActivityPub to a WorkoutType.
func resolveWorkoutType(sport string) model.WorkoutType {
	sport = strings.TrimSpace(strings.ToLower(sport))
	if sport == "" {
		return model.WorkoutTypeOther
	}

	typeMap := map[string]model.WorkoutType{
		"running":         model.WorkoutTypeRunning,
		"run":             model.WorkoutTypeRunning,
		"walking":         model.WorkoutTypeWalking,
		"walk":            model.WorkoutTypeWalking,
		"cycling":         model.WorkoutTypeCycling,
		"cycle":           model.WorkoutTypeCycling,
		"swimming":        model.WorkoutTypeSwimming,
		"hiking":          model.WorkoutTypeHiking,
		"skiing":          model.WorkoutTypeSkiing,
		"snowboarding":    model.WorkoutTypeSnowboarding,
		"kayaking":        model.WorkoutTypeKayaking,
		"rowing":          model.WorkoutTypeRowing,
		"golfing":         model.WorkoutTypeGolfing,
		"push-ups":        model.WorkoutTypePushups,
		"horse-riding":    model.WorkoutTypeHorseRiding,
		"inline-skating":  model.WorkoutTypeInlineSkating,
	}

	if wt, ok := typeMap[sport]; ok {
		return wt
	}

	return model.WorkoutTypeOther
}

// parseWorkoutNoteFromRawActivity extracts the "object" from a raw activity
// JSON and unmarshals it as a WorkoutNote.
func parseWorkoutNoteFromRawActivity(rawPayload []byte) (*WorkoutNote, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rawPayload, &raw); err != nil {
		return nil, err
	}

	objBytes, ok := raw["object"]
	if !ok {
		return nil, nil
	}

	note := &WorkoutNote{}
	if err := note.UnmarshalJSON(objBytes); err != nil {
		return nil, err
	}

	return note, nil
}
