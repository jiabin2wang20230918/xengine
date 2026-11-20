package taskconf

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"gorm.io/gorm"

	"github.com/jiabin2wang20230918/xengine/linkage"
	"github.com/jiabin2wang20230918/xengine/utils"
	"github.com/jiabin2wang20230918/xengine/utils/log"
	"github.com/jiabin2wang20230918/xengine/watch"
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
		enable:   true,
		taskList: sync.Map{},
	}
	if pn != nil {
		t.publishToNorth = *pn
	}
	if ps != nil {
		t.publishToSouth = *ps
	}
	if err := linkage.InitRuleDb(db); err != nil {
		log.Errorf("InitRuleDb failed: %v", err)
	}
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
		link := &linkTask{
			linkRule:    &rule,
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
			select {
			case <-beehiveContext.Done():
				return
			default:
				continue
			}
		}
		operation := beehiveMessage.GetOperation()
		topic := beehiveMessage.GetResource()

		switch operation {
		case "cloudToDeviceMQTTMessage":
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
			payload, ok := beehiveMessage.GetContent().([]byte)
			if ok {
				if err := watch.Watch(topic, payload); err != nil {
					log.Errorf("%v", err)
				}
			}
		case utils.SceneActionRecallOperation:
			sceneId, ok := beehiveMessage.GetContent().(uint16)
			if t.publishToSouth != nil && ok {
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
				payload, err := json.Marshal(recall)
				if err != nil {
					log.Errorf("marshal recall failed: %v", err)
					break
				}
				if err := t.publishToSouth(utils.PublishToDeviceSceneRecallTopic, payload); err != nil {
					log.Errorf("%v", err)
				}
			}
		default:
			log.Warnf("unexpect beehiveMessage:%v", beehiveMessage)
		}
	}
}

func (t *TaskManager) parseTaskPayload(topic string, payload []byte) (int, *msgInfo) {
	errCode := ErrorCodeSuccess
	devInfo := &msgInfo{}
	seps := strings.Split(topic, "/")
	if len(seps) < 7 {
		log.Warnf("unexpect topic=%s", topic)
		return ErrorCodeFail, devInfo
	}
	devInfo.ProductKey = seps[2]
	devInfo.DeviceId = seps[3]

	if seps[5] != "edge" || seps[6] != "notify" {
		log.Warnf("unexpect topic=%s", topic)
		return ErrorCodeFail, devInfo
	}

	msg := parseTaskParams(payload)
	if msg == nil {
		return ErrorCodeFail, devInfo
	}
	devInfo.Id = msg.Id

	existing, ok := t.taskList.Load(msg.Params.Id)
	if !ok && msg.Params.OperateType != "add" {
		log.Warnf("Not found taskId=%s in map", msg.Params.Id)
		return ErrorCodeFail, devInfo
	}

	switch msg.Params.OperateType {
	case "add":
		if ok {
			// Idempotent add: stop existing scene and sync in place to avoid leaking crons or duplicate rules.
			if link, valid := existing.(*linkTask); valid {
				link.StopTask()
				link.linkRule.Enable = true
				link.sceneConfig.MsgId = msg.Id
				if err := link.SyncTask(&msg.Params.Data); err == nil {
					t.taskList.Store(msg.Params.Id, link)
					link.StartTask()
				} else {
					log.Errorf("%v", err)
					errCode = ErrorCodeFail
				}
			}
			break
		}
		link := newLinkTask(msg.Params.Id, &msg.Params.Data, &t.publishToSouth)
		link.sceneConfig.MsgId = msg.Id
		if err := link.AddTask(); err == nil {
			t.taskList.Store(msg.Params.Id, link)
			link.StartTask()
		} else {
			errCode = ErrorCodeFail
		}
	case "update":
		if link, valid := existing.(*linkTask); valid {
			link.StopTask()
			link.sceneConfig.MsgId = msg.Id
			if err := link.SyncTask(&msg.Params.Data); err == nil {
				t.taskList.Store(msg.Params.Id, link)
				if link.linkRule.Enable {
					link.StartTask()
				}
			} else {
				log.Errorf("%v", err)
				errCode = ErrorCodeFail
			}
		}
	case "delete":
		if link, valid := existing.(*linkTask); valid {
			link.StopTask()
			link.sceneConfig.MsgId = msg.Id
			if err := link.DeleteTask(); err != nil {
				log.Errorf("%v", err)
				errCode = ErrorCodeFail
			} else {
				t.taskList.Delete(link.linkRule.RuleId)
			}
		}
	case "stop":
		if link, valid := existing.(*linkTask); valid {
			link.linkRule.Enable = false
			if err := link.linkRule.SetRule(); err != nil {
				log.Errorf("%v", err)
				errCode = ErrorCodeFail
			} else {
				link.StopTask()
			}
		}
	case "run":
		if link, valid := existing.(*linkTask); valid {
			link.linkRule.Enable = true
			if err := link.linkRule.SetRule(); err != nil {
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

	return errCode, devInfo
}

func (t *TaskManager) parseSceneSync(topic string, payload []byte) int {
	errCode := ErrorCodeSuccess

	seps := strings.Split(topic, "/")
	if len(seps) < 7 || seps[5] != "edge" || seps[6] != "get_reply" {
		log.Warnf("unexpect topic=%s", topic)
		return ErrorCodeFail
	}

	msg := parseScenesData(payload)
	if msg == nil {
		log.Errorf("parseScenesData fail")
		return ErrorCodeFail
	}

	remoteIDs := make(map[string]struct{}, len(msg.Data.Scenes))
	for _, sc := range msg.Data.Scenes {
		remoteIDs[sc.Id] = struct{}{}
		existing, ok := t.taskList.Load(sc.Id)
		if ok {
			if link, valid := existing.(*linkTask); valid {
				link.StopTask()
				link.sceneConfig.MsgId = msg.Id
				if err := link.SyncTask(&sc.taskData); err == nil {
					t.taskList.Store(sc.Id, link)
					if link.linkRule.Enable {
						link.StartTask()
					}
				} else {
					log.Errorf("%v", err)
					errCode = ErrorCodeFail
				}
			}
		} else {
			link := newLinkTask(sc.Id, &sc.taskData, &t.publishToSouth)
			link.sceneConfig.MsgId = msg.Id
			if err := link.AddTask(); err == nil {
				t.taskList.Store(sc.Id, link)
				link.StartTask()
			} else {
				log.Errorf("%v", err)
				errCode = ErrorCodeFail
			}
		}
	}

	// Remove local scenes that no longer exist in the cloud snapshot.
	var staleIDs []string
	t.taskList.Range(func(key, _ interface{}) bool {
		if id, ok := key.(string); ok {
			if _, present := remoteIDs[id]; !present {
				staleIDs = append(staleIDs, id)
			}
		}
		return true
	})
	for _, id := range staleIDs {
		if existing, ok := t.taskList.Load(id); ok {
			if link, valid := existing.(*linkTask); valid {
				link.StopTask()
				link.sceneConfig.MsgId = msg.Id
				if err := link.DeleteTask(); err != nil {
					log.Errorf("delete stale scene %s failed: %v", id, err)
					errCode = ErrorCodeFail
				} else {
					t.taskList.Delete(id)
				}
			}
		}
	}

	return errCode
}
