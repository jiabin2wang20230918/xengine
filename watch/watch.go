// Package watch inspects device-to-cloud MQTT messages, updates device property shadows,
// and triggers scene evaluation.
package watch

import (
	"encoding/json"
	"strings"

	"github.com/jiabin2wang20230918/xengine/scene"
	"github.com/jiabin2wang20230918/xengine/shadow"
)

// Watch parses an incoming device MQTT topic and payload:
//   - /sys/{pk}/{deviceId}/thing/event/property/post updates shadow and evaluates property triggers;
//   - /sys/{pk}/{deviceId}/thing/event/{eventName}/post evaluates event triggers.
func Watch(topic string, payload []byte) error {
	split := strings.Split(topic, "/")
	if len(split) < 4 {
		return nil
	}
	deviceId := split[3]
	if len(split) == 8 && split[7] == "post" {
		if split[6] == "property" {
			p := &propertyPost{}
			if err := json.Unmarshal(payload, p); err != nil {
				return err
			}

			for pName, value := range p.Params {
				shadow.UpdatePropertyShadow(deviceId, pName, value.Value)
			}
			for pName, value := range p.Params {
				scene.PropertyTriggerCheck(deviceId, pName, value.Value)
			}
		} else {
			eventName := split[6]
			scene.EventTriggerCheck(deviceId, eventName)
		}
	}
	return nil
}

type propertyPost struct {
	Params map[string]propertyPostValue `json:"params"`
}

type propertyPostValue struct {
	Value interface{} `json:"value"`
	Time  int64       `json:"time"`
}
