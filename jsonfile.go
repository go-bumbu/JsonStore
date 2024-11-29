package jsonstore

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
)

type FileStore struct {
	file    string
	mutex   sync.RWMutex
	content map[string]map[string]json.RawMessage

	// flags
	inMemory      bool
	ManualFlush   bool
	humanReadable bool
}

// make sure the jsonfile store fulfills the JsonStore interface
var _ JsonStorer = &FileStore{}

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

func (f *FileStore) Flush() error {
	if !f.inMemory && !f.ManualFlush {
		f.mutex.Lock()
		defer f.mutex.Unlock()
		return f.flushToFile()
	}
	return nil
}

func (f *FileStore) Set(ctx context.Context, collection, key string, value json.RawMessage) error {

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

func (f *FileStore) Get(ctx context.Context, collection, key string, value *json.RawMessage) error {

	if !f.colExists(collection) {
		return CollectionNotFoundErr
	}
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	if !f.inMemory {

		err := f.readFile()
		if err != nil {
			return err
		}
	}

	d := f.content[collection][key]
	*value = d

	return nil

}

func (f *FileStore) readFile() error {
	fHandle, err := os.Open(f.file)
	if err != nil {
		return fmt.Errorf("unable to open file: %v", err)
	}
	defer fHandle.Close()

	bytes, err := io.ReadAll(fHandle)
	if err != nil {
		return fmt.Errorf("unable to read file: %v", err)
	}

	if len(bytes) == 0 {
		return fmt.Errorf("file is empty")
	}

	var data map[string]map[string]any
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return fmt.Errorf("unable to unmarshal file: %v", err)
	}

	for collection, content := range data {
		for k, v := range content {
			raw, err := json.Marshal(v)
			if err != nil {
				return fmt.Errorf("failed to marshal key %q: %v", k, err)
			}
			f.content[collection][k] = raw
		}
	}

	return nil
}

func (f *FileStore) List(ctx context.Context, collection string, limit, page int) (map[string]json.RawMessage, int64, error) {

	f.mutex.RLock()
	defer f.mutex.RUnlock()
	if collection == "" {
		collection = DefaultCollection
	}
	if !f.colExists(collection) {
		return nil, 0, CollectionNotFoundErr
	}
	collen := len(f.content[collection])

	if limit == 0 || limit > MaxListItems {
		limit = MaxListItems
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	// Extract and sort the keys alphabetically
	keys := make([]string, 0, collen)
	for key := range f.content[collection] {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	end := offset + limit
	if end > len(keys) {
		end = len(keys)
	}

	// Set the resulting map with paginated keys
	result := make(map[string]json.RawMessage, end-offset)
	for _, key := range keys[offset:end] {
		result[key] = f.content[collection][key]
	}
	return result, int64(collen), nil

}

func (f *FileStore) Delete(ctx context.Context, collection, key string) (bool, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if !f.colExists(collection) {
		return false, CollectionNotFoundErr
	}

	entryDeleted := false

	if _, ok := f.content[collection][key]; ok {
		delete(f.content[collection], key)
		entryDeleted = true
	}
	if !f.inMemory && !f.ManualFlush {
		return entryDeleted, f.flushToFile()
	}
	return entryDeleted, nil
}
