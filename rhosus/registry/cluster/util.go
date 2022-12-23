package cluster

import (
	"math/rand"
	"time"
)

func getIntervalMs(min int, max int) time.Duration {
	return time.Millisecond * time.Duration(rand.Intn(max-min)+min)
}
