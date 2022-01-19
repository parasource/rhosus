package fileutil

import "os"

const (
	// PrivateDirMode grants owner to make/remove files inside the directory.
	PrivateDirMode = 0700
)

// OpenDir opens a directory for syncing.
func OpenDir(path string) (*os.File, error) { return os.Open(path) }
