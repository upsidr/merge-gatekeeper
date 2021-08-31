package ticker

import (
	"testing"
	"time"
)

func TestTicker(t *testing.T) {
	delta := 500 * time.Millisecond

	ticker := NewInstantTicker(delta)
	t1 := time.Now()
	t2 := <-ticker.C()
	if t1.Sub(t2) > delta {
		t.Errorf("not instant tic")
	}
	<-ticker.C()
	defer func() {
		if p := recover(); p != nil {
			t.Errorf("panic occurs: %v", p)
		}
	}()
	ticker.Stop()
	ticker.Stop()
	ticker.Stop()

	time.Sleep(2 * delta)

	select {
	case <-ticker.C():
		t.Error("ticker is not stopped")
	default:
	}
}
