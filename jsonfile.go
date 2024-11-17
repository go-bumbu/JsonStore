package jsonstore

import (
	"os"
	"sync"
)

type FileStore struct {
	file    string
	mutex   sync.Mutex
	content map[string]interface{}

	// flags
	inMemory      bool
	ManualFlush   bool
	humanReadable bool
}

type FileStoreFlag int

const (
	MinimizedJson FileStoreFlag = iota
	ManualFlush                 // force manual flush instead of automatically write/read
)
const InMemoryDb = "memory"

func NewFileStore(file string, flags ...FileStoreFlag) (*FileStore, error) {

	db := FileStore{
		file:          file,
		content:       map[string]interface{}{},
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
