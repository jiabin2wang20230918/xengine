// Package watch 提供监控消息并响应场景的功能。
//
// watch监听设备上报给云端的消息，进行解析：
//  - 调用 package shadow 更新缓存。
//  - 调用 package scene 检查是否触发。
// 详见 Watch
package watch

import (
	"encoding/json"
	"github.com/jiabin2wang20230918/xengine/scene"
	"github.com/jiabin2wang20230918/xengine/shadow"
	"strings"
)

// Watch 监控接口，传入要分析的消息。
//  - 信息是上报属性：更新属性的影子值(shadow.UpdatePropertyShadow)，检查是否触发属性的场景触发器(scene.PropertyTriggerCheck)。
//  - 信息是上报事件：检查是否触发事件的场景触发器(scene.EventTriggerCheck)。
func Watch(topic string, payload []byte) error {
	//`/sys/%v/%v/thing/event/property/post`
	split := strings.Split(topic, "/")
	if len(split) < 4 {
		return nil
	}
	deviceId := split[3]
	if len(split) == 8 && split[7] == "post" {
		if split[6] == "property" {
			p := &propertyPost{}
			err := json.Unmarshal(payload, p)
			if err != nil {
				return err
			}

			// 更新cache
			for pName, value := range p.Params {
				shadow.UpdatePropertyShadow(deviceId, pName, value.Value)
			}

			// trigger 检查
			for pName, value := range p.Params {
				scene.PropertyTriggerCheck(deviceId, pName, value.Value)
			}

		} else {
			// 事件监控
			eventName := split[6]
			scene.EventTriggerCheck(deviceId, eventName)
		}
	}

	return nil
}

type propertyPost struct {
	Params map[string]propertyPostValue `json:"params"`
}

type propertyPostValue struct {
	Value interface{} `json:"value"`
	Time  int64       `json:"time"`
}
