package scene

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiabin2wang20230918/xengine/shadow"
)

func setTestActionHook(t *testing.T, fn func(moduleName string, sceneID uint16)) {
	t.Helper()
	sendActionMu.Lock()
	prev := sendAction
	sendAction = fn
	sendActionMu.Unlock()
	t.Cleanup(func() {
		sendActionMu.Lock()
		sendAction = prev
		sendActionMu.Unlock()
	})
}

func TestAddSceneAndRemoveScene(t *testing.T) {
	s := `{
		"id": 123,
		"triggers": [
			{"deviceId": "dev-1", "identifier": "powerSwitch", "type": "property", "value": 1, "condition": "=="},
			{"deviceId": "dev-1", "identifier": "powerSwitch", "type": "property", "value": 3, "condition": "=="},
			{"deviceId": "dev-1", "identifier": "openNotify", "type": "event"},
			{"deviceId": "dev-1", "identifier": "openNotify1", "type": "event"},
			{"time": "52 34 14 ? * 1,3,4,5,6,7 *", "type": "timer"},
			{"time": "0 30 10-13 ? * WED,FRI *", "type": "timer"}
		],
		"conditions": [
			{"deviceId": "dev-1", "identifier": "powerSwitch", "type": "property", "value": 1, "condition": "=="},
			{"time": "52 34 14 ? * 3,4,5,6,7 *", "startTime": "13:22", "endTime": "17:33", "type": "timer"}
		]
	}`
	p := &SceneParams{}
	if err := json.Unmarshal([]byte(s), p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := AddScene(p); err != nil {
		t.Fatalf("AddScene: %v", err)
	}
	defer RemoveScene(123)

	cacheMu.RLock()
	sc, ok := defaultCacheScene[123]
	cacheMu.RUnlock()
	if !ok {
		t.Fatal("scene 123 not cached")
	}
	if len(sc.timeConditions) != 1 || len(sc.propertyConditions) != 1 ||
		len(sc.propertyTriggers["dev-1"]) != 2 || len(sc.eventTriggers["dev-1"]) != 2 ||
		len(sc.tickerTriggers) != 2 {
		t.Fatalf("unexpected scene structure: %+v", sc)
	}

	RemoveScene(123)
	cacheMu.RLock()
	_, ok = defaultCacheScene[123]
	cacheMu.RUnlock()
	if ok {
		t.Fatal("scene 123 still cached after RemoveScene")
	}
}

func TestAddSceneValidationErrors(t *testing.T) {
	if err := AddScene(nil); err == nil {
		t.Fatal("expected error for nil params")
	}

	badCron := &SceneParams{
		Id:       901,
		Triggers: json.RawMessage(`[{"type":"timer","time":"bad cron"}]`),
	}
	if err := AddScene(badCron); err == nil {
		t.Fatal("expected error for invalid timer cron")
	}
	cacheMu.RLock()
	_, exists := defaultCacheScene[901]
	cacheMu.RUnlock()
	if exists {
		t.Fatal("invalid scene should not be cached")
	}
}

func TestAddSceneOverwriteStopsOldCron(t *testing.T) {
	var hits atomic.Int32
	setTestActionHook(t, func(moduleName string, sceneID uint16) {
		if sceneID == 77 {
			hits.Add(1)
		}
	})

	p := &SceneParams{
		Id:                     77,
		Triggers:               json.RawMessage(`[{"type":"timer","time":"*/1 * * * * * *"}]`),
		ActionNotifyModuleName: "mod",
	}
	if err := AddScene(p); err != nil {
		t.Fatalf("first AddScene: %v", err)
	}
	if err := AddScene(p); err != nil {
		t.Fatalf("second AddScene: %v", err)
	}
	defer RemoveScene(77)

	time.Sleep(1500 * time.Millisecond)
	RemoveScene(77)

	count := hits.Load()
	if count < 1 || count > 2 {
		t.Fatalf("expected 1..2 ticks in 1.5s without leaked cron, got %d", count)
	}

	// Ensure no further ticks occur after RemoveScene.
	time.Sleep(1200 * time.Millisecond)
	if after := hits.Load(); after != count {
		t.Fatalf("cron still firing after RemoveScene: before=%d after=%d", count, after)
	}
}

func TestPropertyAndEventTriggerDispatch(t *testing.T) {
	ch := make(chan uint16, 4)
	setTestActionHook(t, func(moduleName string, sceneID uint16) {
		if moduleName == "targetMod" {
			ch <- sceneID
		}
	})

	shadow.UpdatePropertyShadow("cond-dev", "light", float64(80))

	p := &SceneParams{
		Id: 42,
		Triggers: json.RawMessage(`[
			{"deviceId":"trig-dev","identifier":"onOff","type":"property","value":1,"condition":"=="},
			{"deviceId":"trig-dev","identifier":"alarm","type":"event"}
		]`),
		Conditions: json.RawMessage(`[
			{"deviceId":"cond-dev","identifier":"light","type":"property","value":50,"condition":">="}
		]`),
		ActionNotifyModuleName: "targetMod",
	}
	if err := AddScene(p); err != nil {
		t.Fatalf("AddScene: %v", err)
	}
	defer RemoveScene(42)

	PropertyTriggerCheck("trig-dev", "onOff", float64(1))
	select {
	case id := <-ch:
		if id != 42 {
			t.Fatalf("got scene id %d, want 42", id)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for property trigger action")
	}

	EventTriggerCheck("trig-dev", "alarm")
	select {
	case id := <-ch:
		if id != 42 {
			t.Fatalf("got scene id %d, want 42", id)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event trigger action")
	}

	// Failing condition should suppress action.
	shadow.UpdatePropertyShadow("cond-dev", "light", float64(10))
	PropertyTriggerCheck("trig-dev", "onOff", float64(1))
	select {
	case id := <-ch:
		t.Fatalf("unexpected action for scene %d when condition unmet", id)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestParseTriggerTime(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"0 0/1 * * * * *", "0 0/1 * * * *"},
		{"52 34 14 ? * 1,3,4,5,6,7 *", "52 34 14 ? * 0,2,3,4,5,6"},
		{"0 30 10-13 ? * WED,FRI *", "0 30 10-13 ? * WED,FRI"},
		{"0 0 8 ? * */2 *", "0 0 8 ? * */2"},
		{"0 0 8 ? * 1-7/2 *", "0 0 8 ? * 0-6/2"},
	}
	for _, tc := range tests {
		got, err := parseTriggerTime(tc.in)
		if err != nil {
			t.Fatalf("parseTriggerTime(%q) error: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("parseTriggerTime(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSceneConcurrentAddRemoveAndTrigger(t *testing.T) {
	setTestActionHook(t, func(moduleName string, sceneID uint16) {})

	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			id := uint16(500 + worker)
			p := &SceneParams{
				Id:                     id,
				Triggers:               json.RawMessage(fmt.Sprintf(`[{"deviceId":"d-%d","identifier":"sw","type":"property","value":1,"condition":"=="}]`, worker)),
				ActionNotifyModuleName: "noop",
			}
			for i := 0; i < 50; i++ {
				_ = AddScene(p)
				PropertyTriggerCheck(fmt.Sprintf("d-%d", worker), "sw", 1)
				EventTriggerCheck(fmt.Sprintf("d-%d", worker), "ev")
				RemoveScene(id)
			}
		}(g)
	}
	wg.Wait()
}
