package taskconf

import (
	"crypto/md5"
	"github.com/jiabin2wang20230918/xengine/linkage"
	"github.com/jiabin2wang20230918/xengine/scene"
	"github.com/jiabin2wang20230918/xengine/utils/log"
)

type LinkTask struct {
	sceneConfig *linkage.SceneConfig
	linkRule    *linkage.Rule
}

func NewLinkTask(ruleId string, data *taskData, fn *linkage.WriteToSouth) *LinkTask {
	t := &LinkTask{
		sceneConfig: &linkage.SceneConfig{PublishToDevice: *fn},
		linkRule: &linkage.Rule{
			SceneId:    linkage.GenerateSceneId(),
			RuleId:     ruleId,
			Enable:     true,
			Triggers:   data.Triggers,
			Conditions: data.Conditions,
			Actions:    data.Actions,
		},
	}
	return t
}

func (t *LinkTask) StartTask() bool {
	s := &scene.SceneParams{
		Id:                     t.linkRule.SceneId,
		Conditions:             t.linkRule.Conditions,
		Triggers:               t.linkRule.Triggers,
		ActionNotifyModuleName: TaskManagerName,
	}
	err := scene.AddScene(s)
	if err != nil {
		log.Errorf("%v", err)
		return false
	}
	return true
}

func (t *LinkTask) StopTask() bool {
	scene.RemoveScene(t.linkRule.SceneId)
	return true
}

func (t *LinkTask) AddTask() error {
	err := t.linkRule.SetRule()
	if err != nil {
		log.Errorf("%v", err)
		return err
	}
	err = t.sceneConfig.ConfigScene(t.linkRule)
	if err != nil {
		t.linkRule.DeleteRule()
	}
	return err
}

func (t *LinkTask) SyncTask(newTask *taskData) error {
	oldMd5 := md5.Sum(t.linkRule.Actions)
	newMd5 := md5.Sum(newTask.Actions)

	if oldMd5 != newMd5 { // actions change, need to delete and add again
		err := t.DeleteTask()
		if err != nil {
			return err
		}
		t.linkRule.Triggers = newTask.Triggers
		t.linkRule.Conditions = newTask.Conditions
		t.linkRule.Actions = newTask.Actions
		err = t.AddTask()
		return err
	}
	//just update database here
	t.linkRule.Triggers = newTask.Triggers
	t.linkRule.Conditions = newTask.Conditions
	t.linkRule.Actions = newTask.Actions
	err := t.linkRule.SetRule()
	return err
}

func (t *LinkTask) DeleteTask() error {
	err := t.sceneConfig.RemoveScene(t.linkRule)
	if err != nil {
		return err
	}
	err = t.linkRule.DeleteRule()
	return err
}
