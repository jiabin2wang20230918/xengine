package linkage

import (
	"encoding/json"
	"github.com/jiabin2wang20230918/xengine/utils"
)

//type Scene struct {
//	Rule
//}

//type SceneConfig interface {
//	ConfigScene(s *Scene) error
//	StoreScene(sceneId uint16) error
//	RemoveScene(sceneId uint16) error
//	RemoveFromScene(s *Scene) error
//	RecallScene(sceneId uint16) error
//}

//topic和子设备控制相同，payload为场景配置参数
type WriteToSouth func(topic string, payload []byte) error

type SceneConfig struct {
	PublishToDevice WriteToSouth
	MsgId           string
}

func (z *SceneConfig) ConfigScene(s *Rule) error {
	//here to call storScene() for every device
	return z.storeScene(s)
}

func (z *SceneConfig) storeScene(s *Rule) error {
	var err error
	if z.PublishToDevice != nil {
		recall := &struct {
			Id      string          `json:"id"`
			SceneId uint16          `json:"sceneId"`
			Actions json.RawMessage `json:"actions"`
		}{
			Id:      z.MsgId,
			SceneId: s.SceneId,
			Actions: s.Actions,
		}
		payload, _ := json.Marshal(recall)
		err = z.PublishToDevice(utils.PublishToDeviceSceneStoreTopic, payload)
	}
	return err
}

func (z *SceneConfig) RemoveScene(s *Rule) error {
	var err error
	if z.PublishToDevice != nil {
		rm := &struct {
			Id      string          `json:"id"`
			SceneId uint16          `json:"sceneId"`
			Actions json.RawMessage `json:"actions"`
		}{
			Id:      z.MsgId,
			SceneId: s.SceneId,
			Actions: s.Actions,
		}
		payload, _ := json.Marshal(rm)
		err = z.PublishToDevice(utils.PublishToDeviceSceneRemoveTopic, payload)
	}
	return err
}

//func (z *SceneConfig) RecallScene(sceneId uint16) error {
//	var err error
//	if z.PublishToDevice != nil {
//
//		err = z.PublishToDevice("", []byte(""))
//	}
//	return err
//}
