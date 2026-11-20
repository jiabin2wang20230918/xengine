package linkage

import (
	"encoding/json"
	"errors"

	"github.com/jiabin2wang20230918/xengine/utils"
)

// WriteToSouth publishes a scene management message to the south-bound device layer.
type WriteToSouth func(topic string, payload []byte) error

type SceneConfig struct {
	PublishToDevice WriteToSouth
	MsgId           string
}

func (z *SceneConfig) ConfigScene(s *Rule) error {
	return z.storeScene(s)
}

func (z *SceneConfig) storeScene(s *Rule) error {
	if s == nil {
		return errors.New("linkage: rule is nil")
	}
	if z.PublishToDevice == nil {
		return nil
	}
	recall := &struct {
		Id      string          `json:"id"`
		SceneId uint16          `json:"sceneId"`
		Actions json.RawMessage `json:"actions"`
	}{
		Id:      z.MsgId,
		SceneId: s.SceneId,
		Actions: s.Actions,
	}
	payload, err := json.Marshal(recall)
	if err != nil {
		return err
	}
	return z.PublishToDevice(utils.PublishToDeviceSceneStoreTopic, payload)
}

func (z *SceneConfig) RemoveScene(s *Rule) error {
	if s == nil {
		return errors.New("linkage: rule is nil")
	}
	if z.PublishToDevice == nil {
		return nil
	}
	rm := &struct {
		Id      string          `json:"id"`
		SceneId uint16          `json:"sceneId"`
		Actions json.RawMessage `json:"actions"`
	}{
		Id:      z.MsgId,
		SceneId: s.SceneId,
		Actions: s.Actions,
	}
	payload, err := json.Marshal(rm)
	if err != nil {
		return err
	}
	return z.PublishToDevice(utils.PublishToDeviceSceneRemoveTopic, payload)
}
