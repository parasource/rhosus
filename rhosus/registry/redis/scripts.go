package rhosus_redis

var (
	registerFileSource = `redis.call("HSET", KEYS[1], ARGV[1], ARGV[2])`
	removeFileSource   = `redis.call("HDEL", KEYS[1], ARGV[1])`

	addNodeSource    = `redis.call("HSET", KEYS[1], ARGV[1], ARGV[2])`
	removeNodeSource = `redis.call("HDEL", KEYS[1], ARGV[1])`

	getNodesSource = `return redis.call("HDELALL", KEYS[1])`
)
