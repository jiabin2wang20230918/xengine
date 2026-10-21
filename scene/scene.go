// Package scene 提供场景的计算能力。
// 计算场景是否匹配。重点关注触发器（trigger）,条件（condition）
//
//  - trigger:  或 的关系，满足其一即可。
//  - condition: 和 的关系，所有条件满足。
//
// 当任意trigger满足时，检查场景旗下所有condition，满足条件后，通知其他模块执行。
// trigger和condition的文档参考 http://doc.danbay.cn/web/#/3?page_id=322
//
// 提供接口:
// 	- AddScene 用于添加场景的监控
//  - RemoveScene 用于移除场景的监控
//  - PropertyTriggerCheck 属性触发器检查，用于检查这个属性是否匹配场景中的属性触发器。
//  - EventTriggerCheck 事件触发器检查，用于检查这个事件是否匹配场景中的场景触发器。
//
// 依赖：
//  - compare 用于对比是否匹配
//  - shadow 用于条件的对比
package scene

import (
	"encoding/json"
	"fmt"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/robfig/cron/v3"
	"github.com/jiabin2wang20230918/xengine/utils"
	"github.com/jiabin2wang20230918/xengine/utils/log"
	"github.com/jiabin2wang20230918/xengine/utils/message"
	"strings"
)

// Scene scene模块一个场景的基础单位，缓存要关心的 triggers 和 conditions
type Scene struct {
	Id uint16

	// PropertyTriggers 属性触发器 key deviceId
	PropertyTriggers map[string][]*PropertyCondition

	// EventTriggers 事件触发器 key deviceId
	EventTriggers map[string][]*EventCondition

	// PropertyConditions 属性条件
	PropertyConditions []*PropertyCondition

	// TimeConditions 时间条件
	TimeConditions []*TimeCondition

	// TickerTriggers 时间触发器
	TickerTriggers []*TickerTrigger

	// corn 时间触发器执行器
	corn *cron.Cron

	// ActionNotifyModuleName action时发向消息的模块
	ActionNotifyModuleName string
}

// defaultCacheScene 场景缓存，key sceneId
var defaultCacheScene = make(map[uint16]*Scene)

// SceneParams 场景的参数
type SceneParams struct {
	Id                     uint16          `json:"id"`
	Triggers               json.RawMessage `json:"triggers"`
	Conditions             json.RawMessage `json:"conditions"`
	ActionNotifyModuleName string
}

// AddScene 添加场景监控
// scene模块的id不关联业务信息（是云端sceneId还是南向的sceneId都不关心），只是用作key存储和通知别的模块执行此id的场景。
// ActionNotifyModuleName 参数用于指向场景满足时发向的模块名称。
// Triggers/Conditions 和云端协议保持一致的原始json数据。
func AddScene(params *SceneParams) error {

	s := &Scene{
		Id:                     params.Id,
		ActionNotifyModuleName: params.ActionNotifyModuleName,
		PropertyTriggers:       make(map[string][]*PropertyCondition),
		EventTriggers:          make(map[string][]*EventCondition),
		PropertyConditions:     make([]*PropertyCondition, 0),
	}
	// 条件部分处理
	err := s.initConditions(params.Conditions)
	if err != nil {
		return err
	}

	// 触发器部分处理
	err = s.initTriggers(params.Triggers)
	if err != nil {
		return err
	}

	defaultCacheScene[params.Id] = s

	s.tickerTriggerStart()
	return nil
}

// RemoveScene 移除场景监控
func RemoveScene(id uint16) {
	s, ok := defaultCacheScene[id]
	if ok {
		if s.corn != nil {
			s.corn.Stop()
		}
		delete(defaultCacheScene, id)
	}
}

// PropertyTriggerCheck 属性触发器检查
// 检查所有场景是否触发
func PropertyTriggerCheck(deviceId string, propertyName string, propertyValue interface{}) {
	for _, scene := range defaultCacheScene {
		scene.PropertyTriggerCheck(deviceId, propertyName, propertyValue)
	}
}

// EventTriggerCheck 事件触发器检查
// 检查所有场景是否触发
func EventTriggerCheck(deviceId string, eventName string) {
	for _, scene := range defaultCacheScene {
		scene.EventTriggerCheck(deviceId, eventName)
	}
}

// PropertyTriggerCheck 属性触发器检查
// 检查本场景是否触发
func (s *Scene) PropertyTriggerCheck(deviceId string, propertyName string, propertyValue interface{}) {
	triggers, ok := s.PropertyTriggers[deviceId]
	if !ok {
		return
	}
	for _, trigger := range triggers {
		if trigger.Match(propertyName, propertyValue) {
			log.Infof("property trigger of scene(%v) match.device id:%v, property name:%v, property value:%v", s.Id, deviceId, propertyName, propertyValue)
			// check condition
			s.checkAndAction()
		}
	}
}

// EventTriggerCheck 事件触发器检查
// 检查本场景是否触发
func (s *Scene) EventTriggerCheck(deviceId string, eventName string) {
	triggers, ok := s.EventTriggers[deviceId]
	if !ok {
		return
	}
	for _, trigger := range triggers {
		if trigger.Match(eventName) {
			log.Infof("event trigger of scene(%v) match.device id:%v, event name:%v", s.Id, deviceId, eventName)
			// check condition
			s.checkAndAction()
		}
	}
}

// tickerTriggerStart 时间触发器开启，使用定时器，定时触发
func (s *Scene) tickerTriggerStart() {
	if len(s.TickerTriggers) == 0 {
		return
	}
	if s.corn == nil {
		s.corn = cron.New(cron.WithSeconds())
	}
	for _, ticker := range s.TickerTriggers {
		_, err := s.corn.AddFunc(ticker.CronString, func() {
			log.Infof("timer trigger of scene(%v) match.", s.Id)
			s.checkAndAction()
		})
		if err != nil {
			log.Errorf("err:%v", err)
			return
		}
	}
	s.corn.Start()
}

// checkAndAction 检查条件并响应
// 如果所有条件满足则执行场景
func (s *Scene) checkAndAction() {
	if s.allConditionMatch() {
		go s.action()
	}
}

// allConditionMatch 所有条件是否匹配
func (s *Scene) allConditionMatch() bool {
	for _, condition := range s.PropertyConditions {
		if !condition.IsMatch() {
			return false
		}
	}
	for _, condition := range s.TimeConditions {
		if !condition.IsMatch() {
			return false
		}
	}
	return true
}

// action 响应action
// 本模块的action负责将要执行的 id 发送指定模块。
func (s *Scene) action() {
	log.Infof("will execute scene id:%v", s.Id)
	beehiveContext.Send(s.ActionNotifyModuleName, *message.BuildMsg("", "", "scene", "", utils.SceneActionRecallOperation, s.Id))
}

// initConditions 初始化条件相关信息，加载信息到缓存中。
func (s *Scene) initConditions(conditionsBytes []byte) error {
	if conditionsBytes == nil || len(conditionsBytes) == 0 {
		log.Infof("empty condition")
		return nil
	}
	conditions := []CloudSceneConditionParams{}
	err := json.Unmarshal(conditionsBytes, &conditions)
	if err != nil {
		return err
	}
	for _, condition := range conditions {
		if condition.Type == utils.CloudConditionTypeProperty {
			propertyCondition := &PropertyCondition{
				deviceId:      condition.DeviceId,
				propertyName:  condition.Identifier,
				propertyValue: condition.Value,
				symbol:        condition.Condition,
			}
			s.PropertyConditions = append(s.PropertyConditions, propertyCondition)
		} else if condition.Type == utils.CloudConditionTypeTimer {
			timeCondition := &TimeCondition{
				startTime: condition.StartTime,
				endTime:   condition.EndTime,
				time:      condition.Time,
			}
			s.TimeConditions = append(s.TimeConditions, timeCondition)
		} else {
			log.Errorf("unsupported type! data:%v", condition)
		}
	}
	return nil
}

// initTriggers 初始化trigger相关信息，加载信息到缓存中。
func (s *Scene) initTriggers(triggersBytes []byte) error {
	triggersParams := []CloudSceneConditionParams{}
	err := json.Unmarshal(triggersBytes, &triggersParams)
	if err != nil {
		return err
	}
	for _, trigger := range triggersParams {
		if trigger.Type == utils.CloudTriggerTypeProperty {
			propertyCondition := &PropertyCondition{
				deviceId:      trigger.DeviceId,
				propertyName:  trigger.Identifier,
				propertyValue: trigger.Value,
				symbol:        trigger.Condition,
			}
			triggers, ok := s.PropertyTriggers[trigger.DeviceId]
			if !ok {
				triggers = make([]*PropertyCondition, 0)
			}
			triggers = append(triggers, propertyCondition)
			s.PropertyTriggers[trigger.DeviceId] = triggers
		} else if trigger.Type == utils.CloudTriggerTypeEvent {
			e := &EventCondition{eventName: trigger.Identifier}
			triggers, ok := s.EventTriggers[trigger.DeviceId]
			if !ok {
				triggers = make([]*EventCondition, 0)
			}
			triggers = append(triggers, e)
			s.EventTriggers[trigger.DeviceId] = triggers
		} else if trigger.Type == utils.CloudTriggerTypeTimer {
			temp := s.parseTriggerTime(trigger.Time)
			log.Debugf("cron:%s", temp)
			t := &TickerTrigger{CronString: temp}
			if s.TickerTriggers == nil {
				s.TickerTriggers = make([]*TickerTrigger, 0)
			}
			s.TickerTriggers = append(s.TickerTriggers, t)
		} else {
			log.Errorf("unsupported type! data:%v", trigger)
		}
	}
	return nil
}

// parseTriggerTime 转换时间
// robfig/cron库的协议与云端的协议并非一致，需要对时间进行换算。
//
// 1. 去掉年，robfig/cron库不支持年
// 2. 周日-周六 库为0-6 云端协议为1-7 将云端的数据-1
func (s Scene) parseTriggerTime(time string) string {
	split := strings.Split(time, " ")
	if len(split) < 7 {
		return ""
	}

	weekString := split[5]

	result := fmt.Sprintf("%s %s %s %s %s ", split[0], split[1], split[2], split[3], split[4])
	for _, c := range weekString {
		switch c {
		case '1':
			result += "0"
		case '2':
			result += "1"
		case '3':
			result += "2"
		case '4':
			result += "3"
		case '5':
			result += "4"
		case '6':
			result += "5"
		case '7':
			result += "6"
		default:
			result += string(c)
		}
	}
	return result
}

type CloudSceneConditionParams struct {
	StartTime  string      `json:"startTime,omitempty"`
	EndTime    string      `json:"endTime,omitempty"`
	Time       string      `json:"time,omitempty"`
	Type       string      `json:"type"`
	DeviceId   string      `json:"deviceId,omitempty"`
	Identifier string      `json:"identifier,omitempty"`
	Condition  string      `json:"condition,omitempty"`
	Value      interface{} `json:"value,omitempty"`
	ValueType  string      `json:"valueType,omitempty"`
}
