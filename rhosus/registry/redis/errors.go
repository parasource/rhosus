package rhosus_redis

import "errors"

var (
	RedisWriteTimeoutError = errors.New("redis write timeout error")
)
