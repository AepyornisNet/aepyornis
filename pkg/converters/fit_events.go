package converters

import (
	"encoding/json"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/muktihari/fit/profile/filedef"
	"github.com/muktihari/fit/profile/mesgdef"
	"github.com/muktihari/fit/profile/typedef"
	"gorm.io/datatypes"
)

func parseWorkoutEvents(act *filedef.Activity, lengths []fitSwimLength) []model.WorkoutEvent {
	var events []model.WorkoutEvent

	if act != nil && len(act.Events) > 0 {
		events = make([]model.WorkoutEvent, 0, len(act.Events)+len(lengths)*2)
		for _, e := range act.Events {
			if e == nil {
				continue
			}

			ts := e.Timestamp.Local()
			if !fitTimeIsValid(ts) {
				continue
			}

			events = append(events, model.WorkoutEvent{
				Timestamp:      ts,
				StartTimestamp: e.StartTimestamp.Local(),
				Event:          e.Event.String(),
				EventType:      e.EventType.String(),
				EventGroup:     e.EventGroup,
				Payload:        buildFitEventPayload(e),
			})
		}
	}

	for _, l := range lengths {
		if !l.isActive && l.elapsed > 0 {
			events = append(events,
				model.WorkoutEvent{
					Timestamp: l.start,
					Event:     "timer",
					EventType: "stop_all",
				},
				model.WorkoutEvent{
					Timestamp: l.end,
					Event:     "timer",
					EventType: "start",
				},
			)
		}
	}

	return events
}

func buildFitEventPayload(e *mesgdef.Event) datatypes.JSON {
	if e == nil {
		return nil
	}

	event := e.Event.String()
	switch event {
	case "timer":
		triggerType := typedef.TimerTrigger(e.Data)
		if triggerType == typedef.TimerTriggerInvalid {
			return nil
		}

		return mustJSONPayload(struct {
			Trigger string `json:"trigger"`
		}{
			Trigger: triggerType.String(),
		})
	case "front_gear_change":
		return mustJSONPayload(struct {
			FrontGearNum uint8 `json:"front_gear_num"`
			FrontGear    uint8 `json:"front_gear"`
		}{
			FrontGearNum: e.FrontGearNum,
			FrontGear:    e.FrontGear,
		})
	case "rear_gear_change":
		return mustJSONPayload(struct {
			RearGearNum uint8 `json:"rear_gear_num"`
			RearGear    uint8 `json:"rear_gear"`
		}{
			RearGearNum: e.RearGearNum,
			RearGear:    e.RearGear,
		})
	default:
		return nil
	}
}

func mustJSONPayload(v any) datatypes.JSON {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}

	return datatypes.JSON(b)
}
