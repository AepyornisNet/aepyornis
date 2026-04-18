package converters

import (
	"encoding/json"
	"testing"

	"github.com/muktihari/fit/profile/mesgdef"
	"github.com/muktihari/fit/profile/typedef"
	"github.com/stretchr/testify/assert"
)

func TestBuildFitEventPayload_TimerUsesTriggerOnly(t *testing.T) {
	e := mesgdef.NewEvent(nil)
	e.Event = typedef.EventTimer
	e.EventType = typedef.EventTypeStart
	e.Data = uint32(typedef.TimerTriggerAuto)

	payload := buildFitEventPayload(e)
	if assert.NotNil(t, payload) {
		var decoded struct {
			Trigger string `json:"trigger"`
		}
		assert.NoError(t, json.Unmarshal(payload, &decoded))
		assert.Equal(t, "auto", decoded.Trigger)
	}
}

func TestBuildFitEventPayload_GearChangeStructuredFields(t *testing.T) {
	front := mesgdef.NewEvent(nil)
	front.Event = typedef.EventFrontGearChange
	front.FrontGearNum = 2
	front.FrontGear = 52

	frontPayload := buildFitEventPayload(front)
	if assert.NotNil(t, frontPayload) {
		var decoded struct {
			FrontGearNum uint8 `json:"front_gear_num"`
			FrontGear    uint8 `json:"front_gear"`
		}
		assert.NoError(t, json.Unmarshal(frontPayload, &decoded))
		assert.EqualValues(t, 2, decoded.FrontGearNum)
		assert.EqualValues(t, 52, decoded.FrontGear)
	}

	rear := mesgdef.NewEvent(nil)
	rear.Event = typedef.EventRearGearChange
	rear.RearGearNum = 9
	rear.RearGear = 16

	rearPayload := buildFitEventPayload(rear)
	if assert.NotNil(t, rearPayload) {
		var decoded struct {
			RearGearNum uint8 `json:"rear_gear_num"`
			RearGear    uint8 `json:"rear_gear"`
		}
		assert.NoError(t, json.Unmarshal(rearPayload, &decoded))
		assert.EqualValues(t, 9, decoded.RearGearNum)
		assert.EqualValues(t, 16, decoded.RearGear)
	}
}
