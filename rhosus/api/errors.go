package api

import "errors"

var (
	ErrorMissingClientToken = errors.New("missing client token")
	ErrorInvalidClientToken = errors.New("invalid client token")
	ErrorInternal           = errors.New("internal error")
)
