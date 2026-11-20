package scene

import (
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiabin2wang20230918/xengine/utils/log"
)

func TestAddSceneOverwriteStopsPreviousCron(t *testing.T) {
	log.InitInnerLog(log.GetLog())

	var fires int64
	setActionNotify(func(_ string, _ uint16) {
		atomic.AddInt64(&fires, 1)
	})
	t.Cleanup(func() {
		setActionNotify(nil)
		RemoveScene(42)
	})

	// Every second (7-field cloud cron; year dropped by parseTriggerTime).
	triggers, err := json.Marshal([]map[string]interface{}{
		{
			"time": "* * * * * * *",
			"type": "timer",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	params := &SceneParams{
		Id:                     42,
		Triggers:               triggers,
		Conditions:             json.RawMessage(`[]`),
		ActionNotifyModuleName: "test-module",
	}
	if err := AddScene(params); err != nil {
		t.Fatalf("first AddScene: %v", err)
	}
	if err := AddScene(params); err != nil {
		t.Fatalf("overwrite AddScene: %v", err)
	}

	// Allow about 3 seconds of ticking. With the leak bug, both crons fire
	// (~6 events). A single cron should produce roughly 3.
	time.Sleep(3200 * time.Millisecond)
	got := atomic.LoadInt64(&fires)
	if got > 4 {
		t.Fatalf("scene fired %d times after overwrite; expected a single cron (~3), leak would be ~6", got)
	}
	if got < 2 {
		t.Fatalf("scene fired only %d times; timer may not be running", got)
	}
}

func TestAddSceneRemoveSceneConcurrentWithPropertyCheck(t *testing.T) {
	log.InitInnerLog(log.GetLog())
	setActionNotify(func(string, uint16) {})
	t.Cleanup(func() {
		setActionNotify(nil)
		RemoveScene(7)
		RemoveScene(8)
	})

	triggers, err := json.Marshal([]map[string]interface{}{
		{
			"deviceId":   "d-1",
			"identifier": "onOff",
			"type":       "property",
			"value":      1,
			"condition":  "==",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			_ = AddScene(&SceneParams{
				Id:                     uint16(7 + i%2),
				Triggers:               triggers,
				Conditions:             json.RawMessage(`[]`),
				ActionNotifyModuleName: "test-module",
			})
			PropertyTriggerCheck("d-1", "onOff", 1)
			RemoveScene(uint16(7 + i%2))
		}
	}()

	for i := 0; i < 200; i++ {
		PropertyTriggerCheck("d-1", "onOff", 1)
		EventTriggerCheck("d-1", "warn")
	}
	<-done
	// Allow in-flight action goroutines from checkAndAction to finish.
	time.Sleep(50 * time.Millisecond)
}
