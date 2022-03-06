package data

func isBytesAllZeros(id []byte) bool {
	for _, v := range id {
		if v != 0 {
			return false
		}
	}
	return true
}
