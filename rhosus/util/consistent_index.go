package util

import "hash/fnv"

func ConsistentIndex(s string, numBuckets int) int {

	hash := fnv.New64a()
	hash.Write([]byte(s))
	key := hash.Sum64()

	var b int64 = -1
	var j int64

	for j < int64(numBuckets) {
		b = j
		key = key*2862933555777941757 + 1
		j = int64(float64(b+1) * (float64(int64(1)<<31) / float64((key>>33)+1)))
	}

	return int(b)
}
