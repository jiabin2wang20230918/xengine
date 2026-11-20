package watch

import (
	"encoding/json"
	"testing"

	"github.com/jiabin2wang20230918/xengine/scene"
	"github.com/jiabin2wang20230918/xengine/shadow"
	"github.com/jiabin2wang20230918/xengine/utils/log"
)

func TestWatchPropertyAndEvent(t *testing.T) {
	log.InitInnerLog(log.GetLog())

	triggers, err := json.Marshal([]map[string]interface{}{
		{
			"deviceId":   "d-1",
			"identifier": "onOff",
			"type":       "property",
			"value":      1,
			"condition":  "==",
		},
		{
			"deviceId":   "d-3",
			"identifier": "warn",
			"type":       "event",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	conditions, err := json.Marshal([]map[string]interface{}{
		{
			"deviceId":   "d-5",
			"identifier": "lightness",
			"type":       "property",
			"value":      50,
			"condition":  ">=",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := scene.AddScene(&scene.SceneParams{
		Id:                     123,
		Triggers:               triggers,
		Conditions:             conditions,
		ActionNotifyModuleName: "test-module",
	}); err != nil {
		t.Fatalf("AddScene: %v", err)
	}
	t.Cleanup(func() { scene.RemoveScene(123) })

	propTopic := "/sys/pk/d-1/thing/event/property/post"
	propPayload := []byte(`{
		"Id":"123",
		"version":"1.0",
		"timezone":"Asia/Shanghai",
		"params":{
			"onOff":{"value":1,"time":1524448722000}
		}
	}`)
	if err := Watch(propTopic, propPayload); err != nil {
		t.Fatalf("Watch property: %v", err)
	}
	if v, ok := shadow.GetPropertyShadow("d-1", "onOff"); !ok || v != float64(1) {
		t.Fatalf("shadow onOff want 1, got %v ok=%v", v, ok)
	}

	condTopic := "/sys/pk/d-5/thing/event/property/post"
	condPayload := []byte(`{
		"Id":"123",
		"version":"1.0",
		"timezone":"Asia/Shanghai",
		"params":{
			"lightness":{"value":100,"time":1524448722000}
		}
	}`)
	if err := Watch(condTopic, condPayload); err != nil {
		t.Fatalf("Watch condition property: %v", err)
	}

	eventTopic := "/sys/pk/d-3/thing/event/warn/post"
	if err := Watch(eventTopic, []byte(`{}`)); err != nil {
		t.Fatalf("Watch event: %v", err)
	}
}

// TestWatchManualLoop was a long-running manual integration harness that
// blocked forever on select{}. Kept skipped so `go test ./...` stays usable.
func TestWatchManualLoop(t *testing.T) {
	t.Skip("manual integration harness; previously hung on select{}")
}
