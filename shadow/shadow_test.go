package shadow

import (
	"sync"
	"testing"
)

func TestUpdatePropertyShadow(t *testing.T) {
	resetStoreForTest()
	t.Cleanup(resetStoreForTest)

	UpdatePropertyShadow("123", "onOff", 1)
	if v, ok := GetPropertyShadow("123", "onOff"); !ok || v != 1 {
		t.Fatalf("expected onOff=1, got %v ok=%v", v, ok)
	}

	UpdatePropertyShadow("123", "onOff", 0)
	if v, ok := GetPropertyShadow("123", "onOff"); !ok || v != 0 {
		t.Fatalf("expected onOff=0, got %v ok=%v", v, ok)
	}

	UpdatePropertyShadow("123", "name", "gk")
	if v, ok := GetPropertyShadow("123", "name"); !ok || v != "gk" {
		t.Fatalf("expected name=gk, got %v ok=%v", v, ok)
	}

	UpdatePropertyShadow("444", "onOff", 1)
	if v, ok := GetPropertyShadow("444", "onOff"); !ok || v != 1 {
		t.Fatalf("expected device 444 onOff=1, got %v ok=%v", v, ok)
	}
}

func TestPropertyShadowConcurrentAccess(t *testing.T) {
	resetStoreForTest()
	t.Cleanup(resetStoreForTest)

	const goroutines = 32
	const iterations = 200

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)
	for i := 0; i < goroutines; i++ {
		go func(n int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				UpdatePropertyShadow("dev", "onOff", n+j)
				UpdatePropertyShadow("dev", "temp", float64(j))
			}
		}(i)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				GetPropertyShadow("dev", "onOff")
				GetPropertyShadow("dev", "temp")
				GetPropertyShadow("missing", "x")
			}
		}()
	}
	wg.Wait()
}
