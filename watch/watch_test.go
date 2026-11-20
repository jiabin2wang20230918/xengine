package watch

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/kubeedge/beehive/pkg/common"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"

	"github.com/jiabin2wang20230918/xengine/scene"
	"github.com/jiabin2wang20230918/xengine/shadow"
	"github.com/jiabin2wang20230918/xengine/utils"
)

func TestWatchPropertyAndEventTriggers(t *testing.T) {
	const modName = "watchTestReceiver"
	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})
	beehiveContext.AddModule(&common.ModuleInfo{
		ModuleName: modName,
		ModuleType: common.MsgCtxTypeChannel,
	})
	defer beehiveContext.Cleanup(modName)

	received := make(chan model.Message, 8)
	go func() {
		for {
			msg, err := beehiveContext.Receive(modName)
			if err != nil {
				return
			}
			received <- msg
		}
	}()

	rawScene := `{
		"id": 123,
		"triggers": [
			{"deviceId": "d-1", "identifier": "onOff", "type": "property", "value": 1, "condition": "=="},
			{"deviceId": "d-3", "identifier": "warn", "type": "event"}
		],
		"conditions": [
			{"deviceId": "d-5", "identifier": "lightness", "type": "property", "value": 50, "condition": ">="}
		]
	}`
	p := &scene.SceneParams{}
	if err := json.Unmarshal([]byte(rawScene), p); err != nil {
		t.Fatalf("unmarshal scene: %v", err)
	}
	p.ActionNotifyModuleName = modName
	if err := scene.AddScene(p); err != nil {
		t.Fatalf("AddScene: %v", err)
	}
	defer scene.RemoveScene(123)

	// Condition unmet (lightness=10 < 50): property trigger should update shadow but not recall.
	if err := Watch("/sys/pk/d-5/thing/event/property/post", []byte(`{"params":{"lightness":{"value":10,"time":1}}}`)); err != nil {
		t.Fatalf("watch d-5: %v", err)
	}
	if v, ok := shadow.GetPropertyShadow("d-5", "lightness"); !ok || v != float64(10) {
		t.Fatalf("shadow d-5 lightness = (%v, %v), want (10, true)", v, ok)
	}
	if err := Watch("/sys/pk/d-1/thing/event/property/post", []byte(`{"params":{"onOff":{"value":1,"time":2}}}`)); err != nil {
		t.Fatalf("watch d-1: %v", err)
	}
	select {
	case msg := <-received:
		t.Fatalf("unexpected recall when condition unmet: %+v", msg)
	case <-time.After(100 * time.Millisecond):
	}

	// Condition met (lightness=80 >= 50): property trigger should dispatch recall.
	if err := Watch("/sys/pk/d-5/thing/event/property/post", []byte(`{"params":{"lightness":{"value":80,"time":3}}}`)); err != nil {
		t.Fatalf("watch d-5 high: %v", err)
	}
	if err := Watch("/sys/pk/d-1/thing/event/property/post", []byte(`{"params":{"onOff":{"value":1,"time":4}}}`)); err != nil {
		t.Fatalf("watch d-1 trigger: %v", err)
	}
	select {
	case msg := <-received:
		if msg.GetOperation() != utils.SceneActionRecallOperation || msg.GetContent() != uint16(123) {
			t.Fatalf("unexpected recall message: op=%q content=%v", msg.GetOperation(), msg.GetContent())
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for property-triggered scene recall")
	}

	// Event trigger should also dispatch recall.
	if err := Watch("/sys/pk/d-3/thing/event/warn/post", []byte(`{}`)); err != nil {
		t.Fatalf("watch d-3 event: %v", err)
	}
	select {
	case msg := <-received:
		if msg.GetOperation() != utils.SceneActionRecallOperation || msg.GetContent() != uint16(123) {
			t.Fatalf("unexpected event recall message: op=%q content=%v", msg.GetOperation(), msg.GetContent())
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event-triggered scene recall")
	}
}

func TestWatchEdgeCasesAndConcurrency(t *testing.T) {
	if err := Watch("/short", []byte(`{}`)); err != nil {
		t.Fatalf("short topic returned error: %v", err)
	}
	if err := Watch("/sys/pk/d-1/thing/event/property/post", []byte(`{invalid`)); err == nil {
		t.Fatal("expected error for malformed property payload")
	}

	var wg sync.WaitGroup
	for g := 0; g < 6; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			topic := fmt.Sprintf("/sys/pk/dev-%d/thing/event/property/post", id)
			payload := []byte(`{"params":{"temp":{"value":25,"time":1}}}`)
			for i := 0; i < 50; i++ {
				if err := Watch(topic, payload); err != nil {
					t.Errorf("Watch error: %v", err)
				}
			}
		}(g)
	}
	wg.Wait()
}
