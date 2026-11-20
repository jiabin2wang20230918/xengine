package taskconf

import (
	"crypto/md5"

	"github.com/jiabin2wang20230918/xengine/linkage"
	"github.com/jiabin2wang20230918/xengine/scene"
	"github.com/jiabin2wang20230918/xengine/utils/log"
)

type linkTask struct {
	sceneConfig *linkage.SceneConfig
	linkRule    *linkage.Rule
}

var _ task = (*linkTask)(nil)

func newLinkTask(ruleId string, data *taskData, fn *linkage.WriteToSouth) *linkTask {
	var publish linkage.WriteToSouth
	if fn != nil {
		publish = *fn
	}
	rule := &linkage.Rule{
		RuleId: ruleId,
		Enable: true,
	}
	if data != nil {
		rule.Triggers = data.Triggers
		rule.Conditions = data.Conditions
		rule.Actions = data.Actions
	}
	return &linkTask{
		sceneConfig: &linkage.SceneConfig{PublishToDevice: publish},
		linkRule:    rule,
	}
}

func (t *linkTask) StartTask() bool {
	s := &scene.SceneParams{
		Id:                     t.linkRule.SceneId,
		Conditions:             t.linkRule.Conditions,
		Triggers:               t.linkRule.Triggers,
		ActionNotifyModuleName: TaskManagerName,
	}
	if err := scene.AddScene(s); err != nil {
		log.Errorf("%v", err)
		return false
	}
	return true
}

func (t *linkTask) StopTask() bool {
	scene.RemoveScene(t.linkRule.SceneId)
	return true
}

func (t *linkTask) AddTask() error {
	if err := t.linkRule.SetRule(); err != nil {
		log.Errorf("%v", err)
		return err
	}
	if err := t.sceneConfig.ConfigScene(t.linkRule); err != nil {
		if delErr := t.linkRule.DeleteRule(); delErr != nil {
			log.Errorf("rollback DeleteRule(%s) failed: %v", t.linkRule.RuleId, delErr)
		}
		return err
	}
	return nil
}

func (t *linkTask) SyncTask(newTask *taskData) error {
	oldMd5 := md5.Sum(t.linkRule.Actions)
	newMd5 := md5.Sum(newTask.Actions)

	if oldMd5 != newMd5 {
		if err := t.DeleteTask(); err != nil {
			return err
		}
		// Reset SceneId so AddTask allocates or recycles a valid ID.
		t.linkRule.SceneId = 0
		t.linkRule.Triggers = newTask.Triggers
		t.linkRule.Conditions = newTask.Conditions
		t.linkRule.Actions = newTask.Actions
		return t.AddTask()
	}

	t.linkRule.Triggers = newTask.Triggers
	t.linkRule.Conditions = newTask.Conditions
	t.linkRule.Actions = newTask.Actions
	return t.linkRule.SetRule()
}

func (t *linkTask) DeleteTask() error {
	if err := t.sceneConfig.RemoveScene(t.linkRule); err != nil {
		return err
	}
	return t.linkRule.DeleteRule()
}
