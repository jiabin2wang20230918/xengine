package scene

import (
	"encoding/json"
	"github.com/jiabin2wang20230918/xengine/utils/log"
	"testing"
)

func TestAddScene(t *testing.T) {
	log.InitInnerLog(log.GetLog())
	s := `{
    "id": 123,
	"triggers": [
		{
		  "deviceId": "33402f4806031ff185034ffff190821123",
		  "identifier": "powerSwitch",
		  "type": "property",
		  "value": 1,
		  "condition": "=="
		},
		{
		  "deviceId": "33402f4806031ff185034ffff190821123",
		  "identifier": "powerSwitch",
		  "type": "property",
		  "value": 3,
		  "condition": "=="
		},
		{
		  "deviceId": "33402f4806031ff185034ffff190821123",
		  "identifier": "openNotify",
		  "type": "event"
		},
		{
		  "deviceId": "33402f4806031ff185034ffff190821123",
		  "identifier": "openNotify1",
		  "type": "event"
		},
		{
		  "time": "52 34 14 ? * 1,3,4,5,6,7 *",
		  "isPeriod": true,
		  "startTime": "14:23",
		  "endTime": "16:21",
		  "type": "timer"
		},
		{
		  "time": "0 30 10-13 ? * WED,FRI *",
		  "isPeriod": true,
		  "startTime": "14:23",
		  "endTime": "16:21",
		  "type": "timer"
		}
	],
	"conditions": [
		{
		  "deviceId": "33402f4806031ff185034ffff190821123",
		  "identifier": "powerSwitch",
		  "type": "property",
		  "value": 1,
		  "condition": "=="
		},
		{
		  "time": "52 34 14 ? * 3,4,5,6,7 *",
		  "startTime": "13:22",
		  "endTime": "17:33",
		  "type": "timer"
		}
	]
}
`
	b := []byte(s)
	p := &SceneParams{}
	err := json.Unmarshal(b, p)
	if err != nil {
		t.Errorf("err:%v", err)
		return
	}
	err = AddScene(p)
	if err != nil {
		t.Errorf("err:%v", err)
		return
	}
	for id, scene := range defaultCacheScene {
		t.Logf("id:%v scene:%v,%v,%v,%v,%v", id, len(scene.TimeConditions), len(scene.PropertyConditions), len(scene.PropertyTriggers), len(scene.EventTriggers), len(scene.TickerTriggers))
	}
}

func TestMapJson(t *testing.T) {
	b := []byte(`[{"onOff": 1}, {"online": 0}]`)
	type Struct1 struct {
		OnOff int `json:"onOff"`
	}

	type Struct2 struct {
		Online int `json:"online"`
	}
	temp := []map[string]interface{}{}
	err := json.Unmarshal(b, &temp)
	if err != nil {
		t.Errorf("err:%v", err)
		return
	}
	for _, message := range temp {
		//mapT := make(map[string]interface{})
		//err := json.Unmarshal(message, &mapT)
		//if err != nil {
		//	t.Errorf("err:%v", err)
		//	return
		//}
		t.Log(message["onOff"])
		t.Log(message["online"])
	}
}
