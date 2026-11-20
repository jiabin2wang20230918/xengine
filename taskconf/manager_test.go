package taskconf

import (
	"errors"
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/jiabin2wang20230918/xengine/linkage"
	"github.com/jiabin2wang20230918/xengine/utils"
)

func setupTaskManager(t *testing.T, south linkage.WriteToSouth, north WriteToNorth) *TaskManager {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "taskconf_test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := linkage.InitRuleDb(db); err != nil {
		t.Fatalf("InitRuleDb: %v", err)
	}
	t.Cleanup(func() {
		_ = linkage.InitRuleDb(nil)
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return &TaskManager{
		enable:         true,
		publishToSouth: south,
		publishToNorth: north,
	}
}

func TestParseTaskPayloadLifecycleAndDuplicateAdd(t *testing.T) {
	var southTopics []string
	tm := setupTaskManager(t, func(topic string, payload []byte) error {
		southTopics = append(southTopics, topic)
		return nil
	}, nil)

	topic := "/sys/pk1/dev1/thing/edge/notify"
	addPayload := []byte(`{
		"id": "msg-1",
		"version": "1.0",
		"params": {
			"notifyType": "scene",
			"operateType": "add",
			"id": "rule-1",
			"data": {
				"triggers": [{"deviceId":"d1","identifier":"sw","type":"property","value":1,"condition":"=="}],
				"conditions": [],
				"actions": [{"deviceId":"d2","value":1}]
			}
		}
	}`)

	code, info := tm.parseTaskPayload(topic, addPayload)
	if code != ErrorCodeSuccess {
		t.Fatalf("add code = %d, want %d", code, ErrorCodeSuccess)
	}
	if info.ProductKey != "pk1" || info.DeviceId != "dev1" || info.Id != "msg-1" {
		t.Fatalf("unexpected msgInfo: %+v", info)
	}

	// Duplicate add with identical actions should be idempotent (no duplicate DB row).
	code, _ = tm.parseTaskPayload(topic, addPayload)
	if code != ErrorCodeSuccess {
		t.Fatalf("duplicate add code = %d, want %d", code, ErrorCodeSuccess)
	}
	rules, err := linkage.LoadRuleTB()
	if err != nil || len(rules) != 1 {
		t.Fatalf("LoadRuleTB = (%d, %v), want (1, nil)", len(rules), err)
	}

	// Update with changed actions should remove old south scene and store new one.
	updatePayload := []byte(`{
		"id": "msg-2",
		"version": "1.0",
		"params": {
			"operateType": "update",
			"id": "rule-1",
			"data": {
				"triggers": [{"deviceId":"d1","identifier":"sw","type":"property","value":0,"condition":"=="}],
				"conditions": [],
				"actions": [{"deviceId":"d2","value":0}]
			}
		}
	}`)
	code, _ = tm.parseTaskPayload(topic, updatePayload)
	if code != ErrorCodeSuccess {
		t.Fatalf("update code = %d, want %d", code, ErrorCodeSuccess)
	}

	// Stop and run.
	stopPayload := []byte(`{"id":"msg-3","params":{"operateType":"stop","id":"rule-1"}}`)
	if code, _ := tm.parseTaskPayload(topic, stopPayload); code != ErrorCodeSuccess {
		t.Fatalf("stop code = %d, want %d", code, ErrorCodeSuccess)
	}
	runPayload := []byte(`{"id":"msg-4","params":{"operateType":"run","id":"rule-1"}}`)
	if code, _ := tm.parseTaskPayload(topic, runPayload); code != ErrorCodeSuccess {
		t.Fatalf("run code = %d, want %d", code, ErrorCodeSuccess)
	}

	// Delete.
	deletePayload := []byte(`{"id":"msg-5","params":{"operateType":"delete","id":"rule-1"}}`)
	if code, _ := tm.parseTaskPayload(topic, deletePayload); code != ErrorCodeSuccess {
		t.Fatalf("delete code = %d, want %d", code, ErrorCodeSuccess)
	}
	if _, ok := tm.taskList.Load("rule-1"); ok {
		t.Fatal("rule-1 still in taskList after delete")
	}
	rules, err = linkage.LoadRuleTB()
	if err != nil || len(rules) != 0 {
		t.Fatalf("LoadRuleTB after delete = (%d, %v), want (0, nil)", len(rules), err)
	}

	// Verify south topics included store -> remove -> store -> remove.
	wantTopics := []string{
		utils.PublishToDeviceSceneStoreTopic,
		utils.PublishToDeviceSceneRemoveTopic,
		utils.PublishToDeviceSceneStoreTopic,
		utils.PublishToDeviceSceneRemoveTopic,
	}
	if len(southTopics) != len(wantTopics) {
		t.Fatalf("southTopics = %v, want %v", southTopics, wantTopics)
	}
	for i := range wantTopics {
		if southTopics[i] != wantTopics[i] {
			t.Fatalf("southTopics[%d] = %q, want %q", i, southTopics[i], wantTopics[i])
		}
	}
}

func TestParseSceneSyncDeletesStaleLocalRules(t *testing.T) {
	tm := setupTaskManager(t, func(topic string, payload []byte) error {
		return nil
	}, nil)

	topic := "/sys/pk1/dev1/thing/edge/get_reply"
	initial := []byte(`{
		"id": "sync-1",
		"code": "200",
		"data": {
			"scenes": [
				{"id": "r-keep", "triggers": [], "conditions": [], "actions": [{"a":1}]},
				{"id": "r-stale", "triggers": [], "conditions": [], "actions": [{"a":2}]}
			]
		}
	}`)
	if code := tm.parseSceneSync(topic, initial); code != ErrorCodeSuccess {
		t.Fatalf("initial sync code = %d, want %d", code, ErrorCodeSuccess)
	}

	next := []byte(`{
		"id": "sync-2",
		"code": "200",
		"data": {
			"scenes": [
				{"id": "r-keep", "triggers": [], "conditions": [], "actions": [{"a":1}]},
				{"id": "r-new", "triggers": [], "conditions": [], "actions": [{"a":3}]}
			]
		}
	}`)
	if code := tm.parseSceneSync(topic, next); code != ErrorCodeSuccess {
		t.Fatalf("second sync code = %d, want %d", code, ErrorCodeSuccess)
	}

	if _, ok := tm.taskList.Load("r-stale"); ok {
		t.Fatal("stale rule r-stale was not deleted during sync")
	}
	if _, ok := tm.taskList.Load("r-keep"); !ok {
		t.Fatal("r-keep missing after sync")
	}
	if _, ok := tm.taskList.Load("r-new"); !ok {
		t.Fatal("r-new missing after sync")
	}

	rules, err := linkage.LoadRuleTB()
	if err != nil || len(rules) != 2 {
		t.Fatalf("LoadRuleTB = (%d, %v), want (2, nil)", len(rules), err)
	}
}

func TestMalformedTopicsAndNilCallbacksDoNotPanic(t *testing.T) {
	tm := setupTaskManager(t, nil, nil)

	if code, _ := tm.parseTaskPayload("/short/topic", []byte(`{}`)); code != ErrorCodeFail {
		t.Fatalf("short notify topic code = %d, want %d", code, ErrorCodeFail)
	}
	if code := tm.parseSceneSync("/short/topic", []byte(`{}`)); code != ErrorCodeFail {
		t.Fatalf("short sync topic code = %d, want %d", code, ErrorCodeFail)
	}

	// Nil WriteToNorth must not panic.
	notifyRespond(&msgInfo{Id: "1", ProductKey: "pk", DeviceId: "dev"}, ErrorCodeSuccess, nil)

	// Non-nil WriteToNorth should be invoked with expected topic.
	var gotTopic string
	notifyRespond(&msgInfo{Id: "1", ProductKey: "pk", DeviceId: "dev"}, ErrorCodeSuccess, func(topic string, payload []byte) error {
		gotTopic = topic
		return errors.New("simulated error")
	})
	if gotTopic != "/sys/pk/dev/thing/edge/notify_reply" {
		t.Fatalf("notifyRespond topic = %q", gotTopic)
	}
}
