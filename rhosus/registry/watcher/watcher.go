package watcher

import (
	"github.com/parasource/rhosus/rhosus/util/tickers"
	"time"
)

type Watcher struct {
	// Watch interval
	interval int32
}

func (w *Watcher) Watch() {
	ticker := tickers.SetTicker(time.Second * time.Duration(w.interval))
	defer tickers.ReleaseTicker(ticker)

	for {
		select {
		case <-ticker.C:

		}
	}
}
