package fileutil

import (
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
	"os"
	"sync/atomic"
)

var (
	fallocFlags = [...]uint32{
		unix.FALLOC_FL_KEEP_SIZE,                             // Default
		unix.FALLOC_FL_KEEP_SIZE | unix.FALLOC_FL_PUNCH_HOLE, // for ZFS #3066
	}
	fallocFlagsIndex int32
)

func preallocExtend(f *os.File, sizeInBytes int64) error {
	if err := preallocFixed(f, sizeInBytes); err != nil {
		return err
	}
	return preallocExtendTrunc(f, sizeInBytes)
}

func preallocFixed(f *os.File, sizeInBytes int64) error {
	index := atomic.LoadInt32(&fallocFlagsIndex)
again:
	if index >= int32(len(fallocFlags)) {
		return nil // Fallocate is disabled
	}
	flags := fallocFlags[index]
	err := unix.Fallocate(int(f.Fd()), flags, 0, sizeInBytes)
	if err == unix.ENOTSUP {
		// Try the next flags combination
		index++
		atomic.StoreInt32(&fallocFlagsIndex, index)
		log.Error().Err(err).Msg("preAllocate: got error on fallocate")
		goto again

	}
	// FIXME could be doing something here
	// if err == unix.ENOSPC {
	//  log.Printf("No space")
	// }
	return err
}
