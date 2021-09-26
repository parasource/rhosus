package util

import "time"

func RunForever(fn func()) {
	for {
		fn()

		time.Sleep(time.Millisecond * 300)
	}
}
