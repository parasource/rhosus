package tickers

import (
	"sync"
	"time"
)

var tickerPool sync.Pool

func SetTicker(d time.Duration) *time.Ticker {
	v := tickerPool.Get()
	if v == nil {
		return time.NewTicker(d)
	}

	tk := v.(*time.Ticker)
	return tk
}

func ReleaseTicker(tk *time.Ticker) {
	tk.Stop()
	tickerPool.Put(tk)
}
