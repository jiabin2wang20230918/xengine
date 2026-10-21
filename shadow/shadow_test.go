package shadow

import "testing"

func TestUpdatePropertyShadow(t *testing.T) {
	UpdatePropertyShadow("123", "onOff", 1)
	for deviceId, shadow := range DevicesShadow {
		t.Logf("%s - %v", deviceId, shadow)
	}
	UpdatePropertyShadow("123", "onOff", 0)
	for deviceId, shadow := range DevicesShadow {
		t.Logf("%s - %v", deviceId, shadow)
	}
	UpdatePropertyShadow("123", "name", "gk")
	for deviceId, shadow := range DevicesShadow {
		t.Logf("%s - %v", deviceId, shadow)
	}
	UpdatePropertyShadow("444", "onOff", 1)
	for deviceId, shadow := range DevicesShadow {
		t.Logf("%s - %v", deviceId, shadow)
	}
}
