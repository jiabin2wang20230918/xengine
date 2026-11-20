package taskconf

import (
	"encoding/json"
	"fmt"

	"github.com/jiabin2wang20230918/xengine/utils/log"
)

const (
	ErrorCodeSuccess = 200
	ErrorCodeFail    = 400
)

type msgInfo struct {
	Id         string
	DeviceId   string
	ProductKey string
}

type taskData struct {
	Triggers   json.RawMessage `json:"triggers"`
	Conditions json.RawMessage `json:"conditions"`
	Actions    json.RawMessage `json:"actions"`
}

type taskParams struct {
	NotifyType  string   `json:"notifyType"`
	OperateType string   `json:"operateType"`
	Id          string   `json:"id"`
	Data        taskData `json:"data"`
}

type notifyPayload struct {
	Id      string     `json:"id"`
	Version string     `json:"version"`
	Params  taskParams `json:"params"`
}

func parseTaskParams(payload []byte) *notifyPayload {
	p := &notifyPayload{}
	if err := json.Unmarshal(payload, &p); err != nil {
		log.Errorf("%v", err)
		return nil
	}
	return p
}

func notifyRespond(dev *msgInfo, errCode int, fn WriteToNorth) {
	if dev == nil || fn == nil {
		return
	}
	respond := &struct {
		Id   string `json:"id"`
		Code int    `json:"code"`
		Data string `json:"data"`
	}{
		Id:   dev.Id,
		Code: errCode,
		Data: `{}`,
	}

	resource := fmt.Sprintf("/sys/%s/%s/thing/edge/notify_reply", dev.ProductKey, dev.DeviceId)
	content, err := json.Marshal(respond)
	if err != nil {
		log.Errorf("marshal notifyRespond failed: %v", err)
		return
	}
	if err := fn(resource, content); err != nil {
		log.Errorf("publish notifyRespond failed: %v", err)
	}
}

type sceneData struct {
	Id string `json:"id"`
	taskData
}

type sceneReply struct {
	Id      string `json:"id"`
	Code    string `json:"code"`
	Version string `json:"version"`
	Data    struct {
		GetType string      `json:"getType"`
		Scenes  []sceneData `json:"scenes"`
	} `json:"data"`
}

func parseScenesData(payload []byte) *sceneReply {
	reply := &sceneReply{}
	if err := json.Unmarshal(payload, &reply); err != nil {
		log.Errorf("%v", err)
		return nil
	}
	return reply
}
