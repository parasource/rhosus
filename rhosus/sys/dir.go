package sys

import (
	"bazil.org/fuse"
	"bytes"
	"os"
	"strings"
	"time"
)

type Dir struct {
	Name      string
	Parent    *Dir
	id        uint64
	Timestamp int64
}

func (dir *Dir) Id() uint64 {
	if dir.Parent == nil {
		return 1
	}
	return dir.id
}

func (dir *Dir) SetAttributes(attr *fuse.Attr) error {

	attr.Valid = time.Second
	attr.Inode = dir.Id()
	attr.Mode = os.ModeDir
	attr.Mtime = time.Unix(dir.Timestamp, 0)
	attr.Ctime = time.Unix(dir.Timestamp, 0)
	attr.Atime = time.Unix(dir.Timestamp, 0)

	return nil
}

func (dir *Dir) FullPath() string {
	var parts []string
	for p := dir; p != nil; p = p.Parent {
		if strings.HasPrefix(p.Name, "/") {
			if len(p.Name) > 1 {
				parts = append(parts, p.Name[1:])
			}
		} else {
			parts = append(parts, p.Name)
		}
	}

	if len(parts) == 0 {
		return "/"
	}

	var buf bytes.Buffer
	for i := len(parts) - 1; i >= 0; i-- {
		buf.WriteString("/")
		buf.WriteString(parts[i])
	}
	return buf.String()
}
