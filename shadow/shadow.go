// Package shadow maintains an in-memory property cache for devices.
package shadow

import "sync"

type deviceShadow struct {
	propertyShadow map[string]interface{}
}

var (
	mu            sync.RWMutex
	devicesShadow = make(map[string]*deviceShadow)
)

// UpdatePropertyShadow stores or overwrites the cached value for a device property.
func UpdatePropertyShadow(deviceId string, propertyName string, propertyValue interface{}) {
	mu.Lock()
	defer mu.Unlock()

	s, ok := devicesShadow[deviceId]
	if !ok {
		s = &deviceShadow{propertyShadow: make(map[string]interface{})}
		devicesShadow[deviceId] = s
	}
	s.propertyShadow[propertyName] = propertyValue
}

// GetPropertyShadow returns the cached property value and whether it exists.
func GetPropertyShadow(deviceId string, propertyName string) (interface{}, bool) {
	mu.RLock()
	defer mu.RUnlock()

	c, ok := devicesShadow[deviceId]
	if !ok {
		return nil, false
	}
	v, ok := c.propertyShadow[propertyName]
	if !ok {
		return nil, false
	}
	return v, true
}

// reset clears all cached shadows. Intended for tests.
func reset() {
	mu.Lock()
	defer mu.Unlock()
	devicesShadow = make(map[string]*deviceShadow)
}
