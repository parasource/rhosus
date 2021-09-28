package timers

import (
	"sync"
	"time"
)

var timerPool sync.Pool

func SetTimer(d time.Duration) *time.Timer {
	v := timerPool.Get()
	if v == nil {
		return time.NewTimer(d)
	}

	tm := v.(*time.Timer)
	if tm.Reset(d) {
		panic("Received an active timer from the pool!")
	}
	return tm
}

func ReleaseTimer(tm *time.Timer) {
	if !tm.Stop() {
		return
	}
	timerPool.Put(tm)
}
