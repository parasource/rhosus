package finder

type Finder interface {
	Put(key string, offset Offset, size Size) error
	Delete(key string, offset Offset) error
	Close()
	Destroy() error
	ContentSize() uint64
	DeletedSize() uint64
	FileCount() int
	DeletedCount() int
	MaxFileKey() string
	IndexFileSize() uint64
	Sync() error
}
