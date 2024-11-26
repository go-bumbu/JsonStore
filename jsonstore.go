package jsonstore

import (
	"context"
	"encoding/json"
)

// JsonStore interface implements the needed actions to Store and retrieve json Values identified by a key
type JsonStore interface {
	Set(ctx context.Context, key, collection string, value json.RawMessage) error
	Get(ctx context.Context, key, collection string, value *json.RawMessage) error
	Delete(ctx context.Context, key, collection string) (bool, error)
	List(ctx context.Context, collection string, limit, page int) (map[string]json.RawMessage, int64, error)
}

var NonExistedCollectionErr error
