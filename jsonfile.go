package jsonstore

import (
	"context"
	"encoding/json"
	"os"
	"sync"
)

type FileStore struct {
	file      string
	mutex     sync.RWMutex
	fileMutex sync.Mutex
	content   map[string]map[string]json.RawMessage

	// flags
	inMemory      bool
	ManualFlush   bool
	humanReadable bool
}

// make sure the jsonfile store fulfills the JsonStore interface
var _ JsonStore = &FileStore{}

type FileStoreFlag int

const (
	MinimizedJson FileStoreFlag = iota
	ManualFlush                 // force manual flush instead of automatically write/read
)
const InMemoryDb = "memory"

func NewFileStore(file string, flags ...FileStoreFlag) (*FileStore, error) {

	db := FileStore{
		file:          file,
		content:       map[string]map[string]json.RawMessage{},
		inMemory:      true,
		ManualFlush:   isFlagSet(flags, ManualFlush),
		humanReadable: !isFlagSet(flags, MinimizedJson),
	}

	// create a file
	if file != "" && file != InMemoryDb {
		// If the file doesn't exist, create it, or append to the file
		f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		f.Close()
		db.inMemory = false
	}

	return &db, nil
}

func isFlagSet(in []FileStoreFlag, search FileStoreFlag) bool {
	for i := 0; i < len(in); i++ {
		if in[i] == search {
			return true
		}
	}
	return false
}

func (f *FileStore) colExists(name string) bool {
	if _, ok := f.content[name]; !ok {
		return false
	}
	return true
}

func (f *FileStore) Json() []byte {
	var bytes []byte
	var err error
	// json.Marshal function can return two types of errors: UnsupportedTypeError or UnsupportedValueError
	// both cases are handled when adding data with Set, hence omitting error handling here
	if f.humanReadable {
		bytes, err = json.MarshalIndent(f.content, "", "    ")
		if err != nil {
			panic(err)
		}
	} else {
		bytes, err = json.Marshal(f.content)
		if err != nil {
			panic(err)
		}
	}
	return bytes
}

func (f *FileStore) flushToFile() error {

	bytes := f.Json()
	err := os.WriteFile(f.file, bytes, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (f *FileStore) Set(ctx context.Context, key, collection string, value json.RawMessage) error {
	// TODO handle ctx
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if !f.colExists(collection) {
		f.content[collection] = map[string]json.RawMessage{}
	}
	f.content[collection][key] = value
	if !f.inMemory && !f.ManualFlush {
		return f.flushToFile()
	}
	return nil
}

func (f *FileStore) List(ctx context.Context, collection string, limit, page int) (map[string]json.RawMessage, int64, error) {
	//TODO implement me
	panic("implement me")
}

func (f *FileStore) Get(ctx context.Context, key, collection string, value *json.RawMessage) error {
	//TODO implement me
	panic("implement me")
}

func (f *FileStore) Delete(ctx context.Context, key, collection string) (bool, error) {
	//TODO implement me
	panic("implement me")
}
