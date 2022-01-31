package backend

import "fmt"

func blockFromKey(uid string) string {
	return fmt.Sprintf("%v.from", uid)
}

func blockToKey(uid string) string {
	return fmt.Sprintf("%v.to", uid)
}

func blockSizeKey(uid string) string {
	return fmt.Sprintf("%v.size", uid)
}
