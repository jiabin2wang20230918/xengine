package linkage

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/jiabin2wang20230918/xengine/utils"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	SetMaxSceneId(DefaultMaxSceneId)
	dbPath := filepath.Join(t.TempDir(), "linkage_test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := InitRuleDb(db); err != nil {
		t.Fatalf("InitRuleDb: %v", err)
	}
	t.Cleanup(func() {
		SetMaxSceneId(DefaultMaxSceneId)
		_ = InitRuleDb(nil)
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func TestRuleCRUDAndUniqueRuleId(t *testing.T) {
	openTestDB(t)

	triggers := []byte(`[{"type":"event","deviceId":"d1","identifier":"ev"}]`)
	conditions := []byte(`[]`)
	actions := []byte(`[{"deviceId":"d2","value":1}]`)

	r0 := &Rule{
		RuleId:     "rule-a",
		Enable:     true,
		Triggers:   triggers,
		Conditions: conditions,
		Actions:    actions,
	}
	if err := r0.SetRule(); err != nil {
		t.Fatalf("r0.SetRule: %v", err)
	}
	if r0.SceneId != 1 {
		t.Fatalf("r0.SceneId = %d, want 1", r0.SceneId)
	}

	r1 := &Rule{
		RuleId:     "rule-b",
		Enable:     true,
		Triggers:   triggers,
		Conditions: conditions,
		Actions:    actions,
	}
	if err := r1.SetRule(); err != nil {
		t.Fatalf("r1.SetRule: %v", err)
	}
	if r1.SceneId != 2 {
		t.Fatalf("r1.SceneId = %d, want 2", r1.SceneId)
	}

	// Duplicate RuleId with a new SceneId must be rejected by unique constraint.
	dup := &Rule{
		RuleId:   "rule-a",
		Enable:   true,
		Triggers: triggers,
		Actions:  actions,
	}
	if err := dup.SetRule(); err == nil {
		t.Fatal("expected error when inserting duplicate rule_id")
	}
	if dup.SceneId != 0 {
		t.Fatalf("failed insert should roll back SceneId to 0, got %d", dup.SceneId)
	}

	// Query by SceneId should return raw JSON bytes intact.
	got, err := QueryRuleBySceneId(r0.SceneId)
	if err != nil {
		t.Fatalf("QueryRuleBySceneId: %v", err)
	}
	if got.RuleId != "rule-a" || !bytes.Equal(got.Actions, actions) {
		t.Fatalf("unexpected queried rule: %+v", got)
	}

	// Missing SceneId must return gorm.ErrRecordNotFound.
	if _, err := QueryRuleBySceneId(99); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("QueryRuleBySceneId(99) err = %v, want gorm.ErrRecordNotFound", err)
	}

	rules, err := LoadRuleTB()
	if err != nil {
		t.Fatalf("LoadRuleTB: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("LoadRuleTB len = %d, want 2", len(rules))
	}

	// Delete r0 and confirm SceneId=1 is recycled on next allocation.
	if err := r0.DeleteRule(); err != nil {
		t.Fatalf("r0.DeleteRule: %v", err)
	}
	r2 := &Rule{RuleId: "rule-c", Enable: true, Actions: actions}
	if err := r2.SetRule(); err != nil {
		t.Fatalf("r2.SetRule: %v", err)
	}
	if r2.SceneId == r1.SceneId || r2.SceneId == 0 {
		t.Fatalf("unexpected r2.SceneId = %d", r2.SceneId)
	}
}

func TestSceneIdPoolExhaustionAndReload(t *testing.T) {
	db := openTestDB(t)
	SetMaxSceneId(2)

	r1 := &Rule{RuleId: "r1", Actions: []byte(`[]`)}
	r2 := &Rule{RuleId: "r2", Actions: []byte(`[]`)}
	if err := r1.SetRule(); err != nil {
		t.Fatalf("r1: %v", err)
	}
	if err := r2.SetRule(); err != nil {
		t.Fatalf("r2: %v", err)
	}

	r3 := &Rule{RuleId: "r3", Actions: []byte(`[]`)}
	if err := r3.SetRule(); !errors.Is(err, ErrSceneIdExhausted) {
		t.Fatalf("expected ErrSceneIdExhausted, got %v", err)
	}

	// Re-initializing DB must preserve existing used SceneIds from persisted rows.
	if err := InitRuleDb(db); err != nil {
		t.Fatalf("re-InitRuleDb: %v", err)
	}
	if _, err := AllocateSceneId(); !errors.Is(err, ErrSceneIdExhausted) {
		t.Fatalf("expected pool still exhausted after reload, got %v", err)
	}

	if err := r1.DeleteRule(); err != nil {
		t.Fatalf("delete r1: %v", err)
	}
	if err := r3.SetRule(); err != nil {
		t.Fatalf("r3 after recycle: %v", err)
	}
	if r3.SceneId != r1.SceneId {
		t.Fatalf("r3.SceneId = %d, want recycled %d", r3.SceneId, r1.SceneId)
	}
}

func TestSceneConfigPublishAndErrors(t *testing.T) {
	var gotTopic string
	var gotPayload []byte
	cfg := &SceneConfig{
		MsgId: "m-1",
		PublishToDevice: func(topic string, payload []byte) error {
			gotTopic = topic
			gotPayload = payload
			return nil
		},
	}

	if err := cfg.ConfigScene(nil); err == nil {
		t.Fatal("expected error for nil rule")
	}

	r := &Rule{SceneId: 3, RuleId: "r1", Actions: []byte(`[{"k":1}]`)}
	if err := cfg.ConfigScene(r); err != nil {
		t.Fatalf("ConfigScene: %v", err)
	}
	if gotTopic != utils.PublishToDeviceSceneStoreTopic || len(gotPayload) == 0 {
		t.Fatalf("unexpected store publish: topic=%q payload=%s", gotTopic, gotPayload)
	}

	if err := cfg.RemoveScene(r); err != nil {
		t.Fatalf("RemoveScene: %v", err)
	}
	if gotTopic != utils.PublishToDeviceSceneRemoveTopic {
		t.Fatalf("unexpected remove topic=%q", gotTopic)
	}

	// Invalid JSON in Actions should surface json.Marshal error instead of silently sending empty bytes.
	bad := &Rule{SceneId: 4, RuleId: "r2", Actions: []byte(`{invalid`)}
	if err := cfg.ConfigScene(bad); err == nil {
		t.Fatal("expected marshal error for invalid JSON actions")
	}
}
