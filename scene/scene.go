// Package scene evaluates scene triggers and conditions and dispatches recall notifications.
//
// Triggers use OR semantics (any matching trigger evaluates the scene).
// Conditions use AND semantics (all conditions must match to fire the scene action).
package scene

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/robfig/cron/v3"

	"github.com/jiabin2wang20230918/xengine/utils"
	"github.com/jiabin2wang20230918/xengine/utils/log"
	"github.com/jiabin2wang20230918/xengine/utils/message"
)

type sceneEntry struct {
	id                     uint16
	propertyTriggers       map[string][]*propertyCondition
	eventTriggers          map[string][]*eventCondition
	propertyConditions     []*propertyCondition
	timeConditions         []*timeCondition
	tickerTriggers         []*tickerTrigger
	cron                   *cron.Cron
	actionNotifyModuleName string
}

var (
	cacheMu           sync.RWMutex
	defaultCacheScene = make(map[uint16]*sceneEntry)

	sendActionMu sync.RWMutex
	sendAction   = defaultSendAction
)

func defaultSendAction(moduleName string, sceneID uint16) {
	beehiveContext.Send(moduleName, *message.BuildMsg("", "", "scene", "", utils.SceneActionRecallOperation, sceneID))
}

// SceneParams defines inputs for registering a scene in the engine.
type SceneParams struct {
	Id                     uint16          `json:"id"`
	Triggers               json.RawMessage `json:"triggers"`
	Conditions             json.RawMessage `json:"conditions"`
	ActionNotifyModuleName string
}

// AddScene parses and registers a scene. If a scene with the same Id already exists,
// its previous cron scheduler is stopped before replacement.
func AddScene(params *SceneParams) error {
	if params == nil {
		return errors.New("scene params is nil")
	}

	s := &sceneEntry{
		id:                     params.Id,
		actionNotifyModuleName: params.ActionNotifyModuleName,
		propertyTriggers:       make(map[string][]*propertyCondition),
		eventTriggers:          make(map[string][]*eventCondition),
		propertyConditions:     make([]*propertyCondition, 0),
	}

	if err := s.initConditions(params.Conditions); err != nil {
		return err
	}
	if err := s.initTriggers(params.Triggers); err != nil {
		return err
	}
	if err := s.prepareTickerTriggers(); err != nil {
		return err
	}

	cacheMu.Lock()
	old := defaultCacheScene[params.Id]
	defaultCacheScene[params.Id] = s
	if s.cron != nil {
		s.cron.Start()
	}
	cacheMu.Unlock()

	if old != nil && old.cron != nil {
		old.cron.Stop()
	}
	return nil
}

// RemoveScene stops any scheduled triggers for id and removes the scene from the cache.
func RemoveScene(id uint16) {
	cacheMu.Lock()
	s, ok := defaultCacheScene[id]
	if ok {
		delete(defaultCacheScene, id)
	}
	cacheMu.Unlock()

	if ok && s.cron != nil {
		s.cron.Stop()
	}
}

func snapshotScenes() []*sceneEntry {
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	scenes := make([]*sceneEntry, 0, len(defaultCacheScene))
	for _, s := range defaultCacheScene {
		scenes = append(scenes, s)
	}
	return scenes
}

// PropertyTriggerCheck checks all registered scenes for a matching property trigger.
func PropertyTriggerCheck(deviceId string, propertyName string, propertyValue interface{}) {
	for _, s := range snapshotScenes() {
		s.propertyTriggerCheck(deviceId, propertyName, propertyValue)
	}
}

// EventTriggerCheck checks all registered scenes for a matching event trigger.
func EventTriggerCheck(deviceId string, eventName string) {
	for _, s := range snapshotScenes() {
		s.eventTriggerCheck(deviceId, eventName)
	}
}

func (s *sceneEntry) propertyTriggerCheck(deviceId string, propertyName string, propertyValue interface{}) {
	triggers, ok := s.propertyTriggers[deviceId]
	if !ok {
		return
	}
	for _, trigger := range triggers {
		if trigger.match(propertyName, propertyValue) {
			log.Infof("property trigger of scene(%v) match.device id:%v, property name:%v, property value:%v", s.id, deviceId, propertyName, propertyValue)
			s.checkAndAction()
		}
	}
}

func (s *sceneEntry) eventTriggerCheck(deviceId string, eventName string) {
	triggers, ok := s.eventTriggers[deviceId]
	if !ok {
		return
	}
	for _, trigger := range triggers {
		if trigger.match(eventName) {
			log.Infof("event trigger of scene(%v) match.device id:%v, event name:%v", s.id, deviceId, eventName)
			s.checkAndAction()
		}
	}
}

func (s *sceneEntry) prepareTickerTriggers() error {
	if len(s.tickerTriggers) == 0 {
		return nil
	}
	c := cron.New(cron.WithSeconds())
	for _, ticker := range s.tickerTriggers {
		if _, err := c.AddFunc(ticker.cronString, func() {
			log.Infof("timer trigger of scene(%v) match.", s.id)
			s.checkAndAction()
		}); err != nil {
			log.Errorf("invalid timer trigger cron(%s): %v", ticker.cronString, err)
			return fmt.Errorf("invalid timer trigger cron %q: %w", ticker.cronString, err)
		}
	}
	s.cron = c
	return nil
}

func (s *sceneEntry) checkAndAction() {
	if s.allConditionMatch() {
		go s.action()
	}
}

func (s *sceneEntry) allConditionMatch() bool {
	for _, condition := range s.propertyConditions {
		if !condition.isMatch() {
			return false
		}
	}
	for _, condition := range s.timeConditions {
		if !condition.isMatch() {
			return false
		}
	}
	return true
}

func (s *sceneEntry) action() {
	log.Infof("will execute scene id:%v", s.id)
	sendActionMu.RLock()
	fn := sendAction
	sendActionMu.RUnlock()
	if fn != nil {
		fn(s.actionNotifyModuleName, s.id)
	}
}

func (s *sceneEntry) initConditions(conditionsBytes []byte) error {
	if len(conditionsBytes) == 0 {
		log.Infof("empty condition")
		return nil
	}
	var conditions []cloudSceneConditionParams
	if err := json.Unmarshal(conditionsBytes, &conditions); err != nil {
		return err
	}
	for _, condition := range conditions {
		switch condition.Type {
		case utils.CloudConditionTypeProperty:
			s.propertyConditions = append(s.propertyConditions, &propertyCondition{
				deviceId:      condition.DeviceId,
				propertyName:  condition.Identifier,
				propertyValue: condition.Value,
				symbol:        condition.Condition,
			})
		case utils.CloudConditionTypeTimer:
			s.timeConditions = append(s.timeConditions, &timeCondition{
				startTime: condition.StartTime,
				endTime:   condition.EndTime,
				time:      condition.Time,
			})
		default:
			log.Errorf("unsupported condition type! data:%v", condition)
		}
	}
	return nil
}

func (s *sceneEntry) initTriggers(triggersBytes []byte) error {
	var triggersParams []cloudSceneConditionParams
	if err := json.Unmarshal(triggersBytes, &triggersParams); err != nil {
		return err
	}
	for _, trigger := range triggersParams {
		switch trigger.Type {
		case utils.CloudTriggerTypeProperty:
			pc := &propertyCondition{
				deviceId:      trigger.DeviceId,
				propertyName:  trigger.Identifier,
				propertyValue: trigger.Value,
				symbol:        trigger.Condition,
			}
			s.propertyTriggers[trigger.DeviceId] = append(s.propertyTriggers[trigger.DeviceId], pc)
		case utils.CloudTriggerTypeEvent:
			ec := &eventCondition{eventName: trigger.Identifier}
			s.eventTriggers[trigger.DeviceId] = append(s.eventTriggers[trigger.DeviceId], ec)
		case utils.CloudTriggerTypeTimer:
			cronExpr, err := parseTriggerTime(trigger.Time)
			if err != nil {
				return err
			}
			log.Debugf("cron:%s", cronExpr)
			s.tickerTriggers = append(s.tickerTriggers, &tickerTrigger{cronString: cronExpr})
		default:
			log.Errorf("unsupported trigger type! data:%v", trigger)
		}
	}
	return nil
}

// parseTriggerTime converts a 7-field cloud cron string into a 6-field robfig/cron expression:
// 1. drops the trailing year field (unsupported by robfig/cron);
// 2. shifts numeric weekday values from 1-7 (SUN-SAT) to 0-6 without altering any /step divisor.
func parseTriggerTime(expr string) (string, error) {
	split := strings.Fields(expr)
	if len(split) < 7 {
		return "", fmt.Errorf("invalid trigger cron %q: expected 7 fields, got %d", expr, len(split))
	}

	convertedWeek := convertCloudWeekdayField(split[5])
	return fmt.Sprintf("%s %s %s %s %s %s", split[0], split[1], split[2], split[3], split[4], convertedWeek), nil
}

func convertCloudWeekdayField(weekField string) string {
	parts := strings.Split(weekField, ",")
	for i, part := range parts {
		base := part
		suffix := ""
		if idx := strings.IndexByte(part, '/'); idx >= 0 {
			base = part[:idx]
			suffix = part[idx:]
		}
		var b strings.Builder
		for _, c := range base {
			if c >= '1' && c <= '7' {
				b.WriteRune(c - 1)
			} else {
				b.WriteRune(c)
			}
		}
		b.WriteString(suffix)
		parts[i] = b.String()
	}
	return strings.Join(parts, ",")
}

type cloudSceneConditionParams struct {
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
