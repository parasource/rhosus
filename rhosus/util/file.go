package util

import (
	"errors"
	"github.com/rs/zerolog/log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

func GetFileSize(file *os.File) (size int64, err error) {
	var fi os.FileInfo
	if fi, err = file.Stat(); err == nil {
		size = fi.Size()
	}
	return
}

func TestDirWritable(folder string) (err error) {
	fileInfo, err := os.Stat(folder)
	if err != nil {
		return err
	}
	if !fileInfo.IsDir() {
		return errors.New("Not a valid folder!")
	}
	perm := fileInfo.Mode().Perm()
	if 0200&perm != 0 {
		return nil
	}
	return errors.New("Not writable!")
}

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func CheckFile(filename string) (exists, canRead, canWrite bool, modTime time.Time, fileSize int64) {
	exists = true
	fi, err := os.Stat(filename)
	if os.IsNotExist(err) {
		exists = false
		return
	}
	if err != nil {
		log.Error().Err(err).Str("file", filename).Msg("check error")
		return
	}
	if fi.Mode()&0400 != 0 {
		canRead = true
	}
	if fi.Mode()&0200 != 0 {
		canWrite = true
	}
	modTime = fi.ModTime()
	fileSize = fi.Size()
	return
}

func ResolvePath(path string) string {

	if !strings.Contains(path, "~") {
		return path
	}

	usr, _ := user.Current()
	dir := usr.HomeDir

	if path == "~" {
		// In case of "~", which won't be caught by the "else if"
		path = dir
	} else if strings.HasPrefix(path, "~/") {
		// Use strings.HasPrefix so we don't match paths like
		// "/something/~/something/"
		path = filepath.Join(dir, path[2:])
	}

	return path
}
