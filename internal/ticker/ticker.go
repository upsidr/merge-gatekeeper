package ticker

import (
	"sync/atomic"
	"time"
)

type InstantTicker interface {
	Stop()
	C() <-chan time.Time
}

type instantTicker struct {
	tic        *time.Ticker
	tch        chan time.Time
	instTicked int32
	stopped    int32
}

func NewInstantTicker(d time.Duration) InstantTicker {
	return &instantTicker{
		tic: time.NewTicker(d),
		tch: make(chan time.Time, 1),
	}
}

func (it *instantTicker) Stop() {
	it.tic.Stop()
	if atomic.CompareAndSwapInt32(&it.stopped, 0, 1) {
		close(it.tch)
	}
}

func (it *instantTicker) C() <-chan time.Time {
	if atomic.CompareAndSwapInt32(&it.instTicked, 0, 1) {
		if atomic.LoadInt32(&it.stopped) != 1 {
			it.tch <- time.Now()
			return it.tch
		}
	}
	return it.tic.C
}
