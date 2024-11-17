package ignore

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"sync"
)

type Db struct {
	file    string
	mutex   sync.Mutex
	content map[string]interface{}

	// flags
	inMemory      bool
	ManualFlush   bool
	humanReadable bool
}

type DbFlag int

const (
	MinimizedJson DbFlag = iota
	ManualFlush          // force manual flush instead of automatically write/read
)

const InMemoryDb = "memory"

func isFlagSet(in []DbFlag, search DbFlag) bool {
	for i := 0; i < len(in); i++ {
		if in[i] == search {
			return true
		}
	}
	return false
}

func New(file string, flags ...DbFlag) (*Db, error) {

	db := Db{
		file:          file,
		content:       map[string]interface{}{},
		inMemory:      true,
		ManualFlush:   isFlagSet(flags, ManualFlush),
		humanReadable: !isFlagSet(flags, MinimizedJson),
	}

	// create a file
	if file != "" && file != "memory" {
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

func (db *Db) DelCollection(in string) error {
	delete(db.content, in)
	if !db.inMemory && !db.ManualFlush {
		return db.flushToFile()
	}
	return nil
}

// colExists returns true if the specific key for the collection exits, else it returns false
func (db *Db) colExists(name string) bool {
	if _, ok := db.content[name]; !ok {
		return false
	}
	return true
}

func (db *Db) flushToFile() error {

	bytes := db.Json()
	err := os.WriteFile(db.file, bytes, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (db *Db) Json() []byte {
	var bytes []byte
	var err error
	// json.Marshal function can return two types of errors: UnsupportedTypeError or UnsupportedValueError
	// both cases are handled when adding data with Set, hence omitting error handling here
	if db.humanReadable {
		bytes, err = json.MarshalIndent(db.content, "", "    ")
		if err != nil {
			panic(err)
		}
	} else {
		bytes, err = json.Marshal(db.content)
		if err != nil {
			panic(err)
		}
	}
	return bytes
}

func (db *Db) readFile() error {
	f, err := os.Open(db.file)
	if err != nil {
		return fmt.Errorf("unable to open file: %v", err)
	}
	defer f.Close()

	bytes, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("unable to read file: %v", err)
	}

	if len(bytes) == 0 {
		return fmt.Errorf("file is empty")
	}

	var data map[string]interface{}
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return fmt.Errorf("unable to unmarshal file: %v", err)
	}

	for k, v := range data {
		db.content[k] = v
	}

	return nil
}

type payloadT int

const (
	payloadMultiple payloadT = iota
	payloadSingleStruct
	payloadSingleItem
	payloadNotSupported
)

func payloadType(in any) payloadT {
	rt := reflect.TypeOf(in)
	switch rt.Kind() {
	case reflect.Slice, reflect.Array:
		return payloadMultiple
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return payloadSingleItem
	case reflect.Struct:
		return payloadSingleStruct
	default:
		return payloadNotSupported
	}
}

type NonExistentCollectionErr struct{}

func (e NonExistentCollectionErr) Error() string {
	return "collection does not exists"
}

type ValueNotAPointer struct{}

func (e ValueNotAPointer) Error() string {
	return "the passed value is not a pointer"
}

type UnsupportedData struct{}

func (e UnsupportedData) Error() string {
	return "the provided type of data is not supported"
}

// baseKv is the base struct on top what other collection types extend
type baseKv struct {
	name    string
	db      *Db
	content map[string]interface{}
}

func (kv *baseKv) set(key string, value interface{}) error {

	dataType := payloadType(value)
	// early data type check
	if dataType == payloadNotSupported {
		return UnsupportedData{}
	}

	kv.db.mutex.Lock() // optimisation opportunity, make one mutex per collection instead of a global one
	defer kv.db.mutex.Unlock()

	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	switch dataType {
	case payloadSingleStruct:
		var data map[string]interface{}
		err = json.Unmarshal(b, &data)
		if err != nil {
			return err
		}
		kv.content[key] = data
	case payloadMultiple:
		var data []interface{}
		err = json.Unmarshal(b, &data)
		if err != nil {
			return err
		}
		kv.content[key] = data
	case payloadSingleItem:
		var data interface{}
		err = json.Unmarshal(b, &data)
		if err != nil {
			return err
		}
		kv.content[key] = data
	}

	if !kv.db.inMemory && !kv.db.ManualFlush {
		return kv.db.flushToFile()
	}
	return nil
}

func (kv *baseKv) get(key string, value interface{}) error {
	if reflect.ValueOf(value).Kind() != reflect.Ptr {
		return ValueNotAPointer{}
	}
	if !kv.db.colExists(kv.name) {
		return NonExistentCollectionErr{}
	}

	kv.db.mutex.Lock()
	defer kv.db.mutex.Unlock()

	if !kv.db.inMemory {
		err := kv.db.readFile()
		if err != nil {
			return err
		}
	}

	jsonData, err := json.Marshal(kv.content[key])
	if err != nil {
		return err
	}
	err = json.Unmarshal(jsonData, &value)
	if err != nil {
		return err
	}
	return nil
}

// append adds one item of the type slice stored in value
func (kv *baseKv) append(key string, value interface{}) error {

	dataType := payloadType(value)

	// early data type check
	if dataType == payloadNotSupported {
		return UnsupportedData{}
	}

	kv.db.mutex.Lock() // optimisation opportunity, make one mutex per collection instead of a global one
	defer kv.db.mutex.Unlock()

	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	if kv.content[key] == nil {
		kv.content[key] = []interface{}{}
	}

	switch dataType {
	case payloadSingleStruct, payloadSingleItem:

		var data interface{}
		err = json.Unmarshal(b, &data)
		if err != nil {
			return err
		}
		kv.content[key] = append(kv.content[key].([]interface{}), data)

	case payloadMultiple:
		var data []interface{}
		err = json.Unmarshal(b, &data)
		if err != nil {
			return err
		}
		kv.content[key] = append(kv.content[key].([]interface{}), data...)
	default:
		return UnsupportedData{}

	}

	return nil
}
