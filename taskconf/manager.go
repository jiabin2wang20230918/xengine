package taskconf

import (
	"encoding/json"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/jiabin2wang20230918/xengine/linkage"
	"github.com/jiabin2wang20230918/xengine/utils"
	"github.com/jiabin2wang20230918/xengine/utils/log"
	"github.com/jiabin2wang20230918/xengine/watch"
	"strings"
	"sync"
)

type WriteToNorth func(topic string, payload []byte) error

type TaskManager struct {
	enable         bool
	taskList       sync.Map
	publishToNorth WriteToNorth
	publishToSouth linkage.WriteToSouth
}

func Register(db *gorm.DB, ps *linkage.WriteToSouth, pn *WriteToNorth) {
	t := &TaskManager{
		enable:         true,
		taskList:       sync.Map{},
		publishToNorth: *pn,
		publishToSouth: *ps,
	}
	linkage.InitRuleDb(db)
	core.Register(t)
}

func (*TaskManager) Name() string {
	return TaskManagerName
}

func (*TaskManager) Group() string {
	return TaskManagerGroup
}

func (t *TaskManager) Enable() bool {
	return t.enable
}

func (t *TaskManager) Start() {
	t.loadAndRun()
	t.taskProcess()
}

func (t *TaskManager) loadAndRun() {
	rules, err := linkage.LoadRuleTB()
	if err != nil {
		return
	}
	for _, r := range rules {
		rule := r
		link := &LinkTask{linkRule: &rule,
			sceneConfig: &linkage.SceneConfig{PublishToDevice: t.publishToSouth},
		}
		t.taskList.Store(rule.RuleId, link)
		if rule.Enable {
			link.StartTask()
			log.Infof("load and run ===> task[%s]", r.RuleId)
		}
	}
}

func (t *TaskManager) taskProcess() {
	for {
		select {
		case <-beehiveContext.Done():
			log.Infof("Task Process handler end")
			return
		default:
		}

		beehiveMessage, err := beehiveContext.Receive(t.Name())
		if err != nil {
			log.Errorf("receive err:%v", err)
		}
		operation := beehiveMessage.GetOperation()
		topic := beehiveMessage.GetResource()

		switch operation {
		case "cloudToDeviceMQTTMessage":
			// notify/get_reply
			split := strings.Split(topic, "/")
			switch split[len(split)-1] {
			case "notify":
				payload, ok := beehiveMessage.GetContent().([]byte)
				if ok {
					ret, devInfo := t.parseTaskPayload(topic, payload)
					notifyRespond(devInfo, ret, t.publishToNorth)
				}
			case "get_reply":
				payload, ok := beehiveMessage.GetContent().([]byte)
				if ok {
					_ = t.parseSceneSync(topic, payload)

				}
			default:
				log.Errorf("not supported topic:%s", topic)
			}
		case "deviceToCloudMQTTMessage":
			// event/+/post
			payload, ok := beehiveMessage.GetContent().([]byte)
			if ok {
				err := watch.Watch(topic, payload)
				if err != nil {
					log.Errorf("%v", err)
				}
			}
		case utils.SceneActionRecallOperation:
			sceneId, ok := beehiveMessage.GetContent().(uint16)
			if &t.publishToSouth != nil && ok {
				r, err := linkage.QueryRuleBySceneId(sceneId)
				if err != nil {
					log.Errorf("%v", err)
					break
				}
				recall := &struct {
					Id      string          `json:"id"`
					SceneId uint16          `json:"sceneId"`
					Actions json.RawMessage `json:"actions"`
				}{
					Id:      fmt.Sprintf("Recall-%d", sceneId),
					SceneId: sceneId,
					Actions: r.Actions,
				}
				payload, _ := json.Marshal(recall)
				topic = utils.PublishToDeviceSceneRecallTopic
				err = t.publishToSouth(topic, payload)
				if err != nil {
					log.Errorf("%v", err)
				}
			}
		default:
			log.Warnf("unexpect beehiveMessage:%v", beehiveMessage)

		}

	}
}

func (t *TaskManager) parseTaskPayload(topic string, payload []byte) (int, *MsgInfo) {
	var errCode = ErrorCodeSuccess
	devInfo := &MsgInfo{}
	seps := strings.Split(topic, "/")
	devInfo.DeviceId = seps[3]
	devInfo.ProductKey = seps[2]

	if seps[5] == "edge" && seps[6] == "notify" {
		msg := parseTaskParams(payload)
		if msg == nil {
			errCode = ErrorCodeFail
			return errCode, devInfo
		}
		devInfo.Id = msg.Id

		task, ok := t.taskList.Load(msg.Params.Id)
		if !ok && msg.Params.OperateType != "add" {
			log.Warnf("!! Not found taskId=%s in map", msg.Params.Id)
			errCode = ErrorCodeFail
			return errCode, devInfo
		}

		switch msg.Params.OperateType {
		case "add":
			link := NewLinkTask(msg.Params.Id, &msg.Params.Data, &t.publishToSouth)
			link.sceneConfig.MsgId = msg.Id
			if err := link.AddTask(); err == nil {
				t.taskList.Store(msg.Params.Id, link)
				link.StartTask()
			} else { //fail
				errCode = ErrorCodeFail
			}
		case "update":
			if link, ok := task.(*LinkTask); ok {
				link.StopTask()
				//如果是修改，需先删除设备端旧数据再新增
				link.sceneConfig.MsgId = msg.Id
				if err := link.SyncTask(&msg.Params.Data); err == nil {
					t.taskList.Store(msg.Params.Id, link)
					link.StartTask()
				} else {
					log.Errorf("%v", err)
					errCode = ErrorCodeFail
				}
			}
		case "delete":
			if link, ok := task.(*LinkTask); ok {
				link.StopTask()
				if err := link.DeleteTask(); err != nil {
					errCode = ErrorCodeFail
				}
				t.taskList.Delete(link.linkRule.RuleId)
			}

		case "stop":
			if link, ok := task.(*LinkTask); ok {
				link.linkRule.Enable = false
				err := link.linkRule.SetRule() //更新数据库
				if err != nil {
					log.Errorf("%v", err)
					errCode = ErrorCodeFail
				} else {
					link.StopTask()
				}
			}
		case "run":
			if link, ok := task.(*LinkTask); ok {
				link.linkRule.Enable = true
				err := link.linkRule.SetRule()
				if err != nil {
					log.Errorf("%v", err)
					errCode = ErrorCodeFail
				} else {
					link.StartTask()
				}
			}
		default:
			log.Warnf("not support operateType=%s", msg.Params.OperateType)
			errCode = ErrorCodeFail
		}

	} else {
		log.Warnf("!! unexpect topic=%s", topic)
		errCode = ErrorCodeFail
	}

	return errCode, devInfo
}

func (t *TaskManager) parseSceneSync(topic string, payload []byte) int {
	var errCode = ErrorCodeSuccess

	seps := strings.Split(topic, "/")
	if seps[5] == "edge" && seps[6] == "get_reply" {
		msg := parseScenesData(payload)
		if msg == nil {
			log.Errorf("parseScenesData fail !!")
			errCode = ErrorCodeFail
			return errCode
		}

		for _, scene := range msg.Data.Scenes {
			task, ok := t.taskList.Load(scene.Id)
			if ok {
				if link, ok := task.(*LinkTask); ok {
					link.StopTask()
					link.sceneConfig.MsgId = msg.Id
					if err := link.SyncTask(&scene.taskData); err == nil {
						t.taskList.Store(scene.Id, link)
						link.StartTask()
					} else {
						log.Errorf("%v", err)
						errCode = ErrorCodeFail
					}
				}
			} else {
				link := NewLinkTask(scene.Id, &scene.taskData, &t.publishToSouth)
				link.sceneConfig.MsgId = msg.Id
				if err := link.AddTask(); err == nil {
					t.taskList.Store(scene.Id, link)
					link.StartTask()
				} else { //fail
					log.Errorf("%v", err)
					errCode = ErrorCodeFail
				}
			}
		}
		//Todo here: 删除部分也需要对比处理
	} else {
		log.Warnf("!! unexpect topic=%s", topic)
		errCode = ErrorCodeFail
	}

	return errCode
}
