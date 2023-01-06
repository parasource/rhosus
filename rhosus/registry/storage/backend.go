package storage

type EntryType string

type Entry struct {
	Key   string
	value []byte
}

type Backend interface {
	Put(t EntryType, entries []*Entry) error
	Get(t EntryType, keys []string) ([]*Entry, error)
	Delete(t EntryType, keys []string) error
	List(t EntryType) ([]*Entry, error)
}
