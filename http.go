package jsonstore

import "encoding/json"

type JsonStore interface {
	Set(key, collection string, value json.RawMessage) error
	Get(key, collection string, value *json.RawMessage) error
	List(collection string, limit, page int) (map[string]json.RawMessage, int64, error)
	Delete(key, collection string) (int64, error)
}
