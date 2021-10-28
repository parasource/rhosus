package sys

import (
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"math"
	"os"
	"time"
)

const BlockSizeMb = 64

type File struct {
	id        uint64
	Name      string
	FullPath  string
	dir       *Dir
	Timestamp int64
	Size      uint64
	Data      []byte
}

func (f *File) SetAttributes(attr *fuse.Attr) {

	attr.Mode = f.mode()
	attr.Size = f.Size
	attr.BlockSize = BlockSizeMb
	attr.Blocks = uint64(math.Ceil(float64(f.Size / BlockSizeMb)))

	attr.Mtime = time.Unix(f.Timestamp, 0)
	attr.Ctime = time.Unix(f.Timestamp, 0)
	attr.Atime = time.Unix(f.Timestamp, 0)
}

func (file *File) size() uint64 {
	return uint64(len(file.Data))
}

func (file *File) mode() os.FileMode {
	return os.FileMode(0x644)
}

type fileNode struct {
	fs.Node
	fs.FSInodeGenerator
	path string
}

func NewFileNode(path string) fs.Node {
	return &fileNode{
		path: path,
	}
}