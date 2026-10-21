// Package shadow 是影子模块，用于更新和获取缓存。
//
// 目前支持属性缓存与获取
package shadow

type DeviceShadow struct {
	// PropertyShadow key 为属性名称 value位属性值
	PropertyShadow map[string]interface{}
}

// DevicesShadow key为deviceId
var DevicesShadow = make(map[string]*DeviceShadow)

// UpdatePropertyShadow 更新属性的影子值
func UpdatePropertyShadow(deviceId string, propertyName string, propertyValue interface{}) {
	shadow, ok := DevicesShadow[deviceId]
	if !ok {
		shadow = &DeviceShadow{PropertyShadow: make(map[string]interface{})}
		DevicesShadow[deviceId] = shadow
	}
	shadow.PropertyShadow[propertyName] = propertyValue
}

// GetPropertyShadow 获取属性的影子值
// 返回：值，是否有缓存
func GetPropertyShadow(deviceId string, propertyName string) (interface{}, bool) {
	c, ok := DevicesShadow[deviceId]
	if !ok {
		return nil, false
	}
	v, ok := c.PropertyShadow[propertyName]
	if !ok {
		return nil, false
	}
	return v, true
}
