package storage

type EntryType string

type Entry struct {
	Type  EntryType
	Key   string
	value []byte
}

type Backend interface {
	Put(entry *Entry) error
	Get(t EntryType, key string) (*Entry, error)
	Delete(t EntryType, key string) error
	List(entryType EntryType) ([]*Entry, error)
}
