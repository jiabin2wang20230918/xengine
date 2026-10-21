package watch

import (
	"encoding/json"
	"github.com/jiabin2wang20230918/xengine/scene"
	"github.com/jiabin2wang20230918/xengine/utils/log"
	"testing"
	"time"
)

func TestWatch(t *testing.T) {
	log.InitInnerLog(log.GetLog())
	s := `{
    "id": 123,
	"triggers": [
		{
		  "deviceId": "d-1",
		  "identifier": "onOff",
		  "type": "property",
		  "value": 1,
		  "condition": "=="
		},
		{
		  "deviceId": "d-2",
		  "identifier": "onOff",
		  "type": "property",
		  "value": 0,
		  "condition": "=="
		},
		{
		  "deviceId": "d-3",
		  "identifier": "warn",
		  "type": "event"
		},
		{
		  "deviceId": "d-4",
		  "identifier": "boom",
		  "type": "event"
		},
		{
		  "time": "0 0/1 * * * * *",
		  "type": "timer"
		}
	],
	"conditions": [
		{
		  "deviceId": "d-5",
		  "identifier": "lightness",
		  "type": "property",
		  "value": 50,
		  "condition": ">="
		},
		{
		  "time": "* * * * * 3,4,5,6,7 *",
		  "startTime": "13:22",
		  "endTime": "17:33",
		  "type": "timer"
		}
	]
}
`
	b := []byte(s)
	p := &scene.SceneParams{}
	err := json.Unmarshal(b, p)
	if err != nil {
		t.Errorf("err:%v", err)
		return
	}
	err = scene.AddScene(p)
	if err != nil {
		t.Errorf("err:%v", err)
		return
	}

	// 定时触发触发器
	go func() {
		for {
			var topic string
			var payload []byte
			// /sys/%v/%v/thing/event/property/post
			// /sys/%v/%v/thing/event/%v/post
			topic = "/sys/pk/d-1/thing/event/property/post"
			payload = []byte(`
{
	"Id":"123",
    "version":"1.0",
    "timezone":"Asia/Shanghai",
    "params":{
        "onOff":{
            "value":1,
            "time":1524448722000
        }
	}

}
`)
			t.Logf("watch topic:%s payload:%s", topic, payload)
			err := Watch(topic, payload)
			if err != nil {
				t.Errorf("err:%v", err)
				return
			}
			time.Sleep(5 * time.Second)
			// ---------------
			topic = "/sys/pk/d-2/thing/event/property/post"
			payload = []byte(`
{
	"Id":"123",
    "version":"1.0",
    "timezone":"Asia/Shanghai",
    "params":{
        "onOff":{
            "value":0,
            "time":1524448722000
        }
	}

}
`)
			t.Logf("watch topic:%s payload:%s", topic, payload)
			err = Watch(topic, payload)
			if err != nil {
				t.Errorf("err:%v", err)
				return
			}
			time.Sleep(5 * time.Second)
			// ---------------
			topic = "/sys/pk/d-3/thing/event/warn/post"
			payload = []byte(`
{
	
}
`)
			t.Logf("watch topic:%s payload:%s", topic, payload)
			err = Watch(topic, payload)
			if err != nil {
				t.Errorf("err:%v", err)
				return
			}
			time.Sleep(5 * time.Second)

			// ---------------
			topic = "/sys/pk/d-4/thing/event/boom/post"
			payload = []byte(`
{
	
}
`)
			t.Logf("watch topic:%s payload:%s", topic, payload)
			err = Watch(topic, payload)
			if err != nil {
				t.Errorf("err:%v", err)
				return
			}
			time.Sleep(5 * time.Second)

		}
	}()

	// 定时更新条件
	go func() {
		for {
			var topic string
			var payload []byte
			// /sys/%v/%v/thing/event/property/post
			// /sys/%v/%v/thing/event/%v/post
			topic = "/sys/pk/d-5/thing/event/property/post"
			payload = []byte(`
{
	"Id":"123",
    "version":"1.0",
    "timezone":"Asia/Shanghai",
    "params":{
        "lightness":{
            "value":1,
            "time":1524448722000
        }
	}
}
`)
			t.Logf("watch topic:%s payload:%s", topic, payload)
			err := Watch(topic, payload)
			if err != nil {
				t.Errorf("err:%v", err)
				return
			}
			time.Sleep(30 * time.Second)
			topic = "/sys/pk/d-5/thing/event/property/post"
			payload = []byte(`
{
	"Id":"123",
    "version":"1.0",
    "timezone":"Asia/Shanghai",
    "params":{
        "lightness":{
            "value":100,
            "time":1524448722000
        }
	}
}
`)
			t.Logf("watch topic:%s payload:%s", topic, payload)
			err = Watch(topic, payload)
			if err != nil {
				t.Errorf("err:%v", err)
				return
			}
			time.Sleep(30 * time.Second)
		}
	}()

	select {}
}
