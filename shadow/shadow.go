// Package shadow 是影子模块，用于更新和获取缓存。
//
// 目前支持属性缓存与获取
package shadow

import "sync"

type DeviceShadow struct {
	// PropertyShadow key 为属性名称 value位属性值
	PropertyShadow map[string]interface{}
}

type devicesStore struct {
	mu      sync.RWMutex
	devices map[string]*DeviceShadow
}

var store = &devicesStore{
	devices: make(map[string]*DeviceShadow),
}

// UpdatePropertyShadow 更新属性的影子值
func UpdatePropertyShadow(deviceId string, propertyName string, propertyValue interface{}) {
	store.mu.Lock()
	defer store.mu.Unlock()
	shadow, ok := store.devices[deviceId]
	if !ok {
		shadow = &DeviceShadow{PropertyShadow: make(map[string]interface{})}
		store.devices[deviceId] = shadow
	}
	shadow.PropertyShadow[propertyName] = propertyValue
}

// GetPropertyShadow 获取属性的影子值
// 返回：值，是否有缓存
func GetPropertyShadow(deviceId string, propertyName string) (interface{}, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	c, ok := store.devices[deviceId]
	if !ok {
		return nil, false
	}
	v, ok := c.PropertyShadow[propertyName]
	if !ok {
		return nil, false
	}
	return v, true
}

// resetStoreForTest clears all shadow state. Test-only.
func resetStoreForTest() {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.devices = make(map[string]*DeviceShadow)
}
