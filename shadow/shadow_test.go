package shadow

import (
	"fmt"
	"sync"
	"testing"
)

func TestUpdateAndGetPropertyShadow(t *testing.T) {
	reset()

	if _, ok := GetPropertyShadow("123", "onOff"); ok {
		t.Fatal("expected missing property to return ok=false")
	}

	UpdatePropertyShadow("123", "onOff", 1)
	if v, ok := GetPropertyShadow("123", "onOff"); !ok || v != 1 {
		t.Fatalf("got (%v, %v), want (1, true)", v, ok)
	}

	UpdatePropertyShadow("123", "onOff", 0)
	if v, ok := GetPropertyShadow("123", "onOff"); !ok || v != 0 {
		t.Fatalf("got (%v, %v), want (0, true)", v, ok)
	}

	UpdatePropertyShadow("123", "name", "gk")
	if v, ok := GetPropertyShadow("123", "name"); !ok || v != "gk" {
		t.Fatalf("got (%v, %v), want (gk, true)", v, ok)
	}

	UpdatePropertyShadow("444", "onOff", 1)
	if v, ok := GetPropertyShadow("444", "onOff"); !ok || v != 1 {
		t.Fatalf("got (%v, %v), want (1, true)", v, ok)
	}
	if _, ok := GetPropertyShadow("444", "name"); ok {
		t.Fatal("expected missing property on device 444 to return ok=false")
	}
}

func TestShadowConcurrentAccess(t *testing.T) {
	reset()

	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			dev := fmt.Sprintf("dev-%d", id%3)
			for i := 0; i < 200; i++ {
				UpdatePropertyShadow(dev, "temp", i)
				_, _ = GetPropertyShadow(dev, "temp")
			}
		}(g)
	}
	wg.Wait()

	for i := 0; i < 3; i++ {
		dev := fmt.Sprintf("dev-%d", i)
		if _, ok := GetPropertyShadow(dev, "temp"); !ok {
			t.Fatalf("expected device %s to have cached temp", dev)
		}
	}
}
