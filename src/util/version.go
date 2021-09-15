package util

import (
	"fmt"
)

const (
	VERSION = "0.5.5"
	COMMIT  = ""
)

func Version() string {
	return fmt.Sprintf("%v %v", VERSION, COMMIT)
}
