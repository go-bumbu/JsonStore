package jsonstore

import (
	"context"
	"encoding/json"
)

type JsonStore interface {
	Set(ctx context.Context, key, collection string, value json.RawMessage) error
	Get(ctx context.Context, key, collection string, value *json.RawMessage) error
	List(ctx context.Context, collection string, limit, page int) (map[string]json.RawMessage, int64, error)
	Delete(ctx context.Context, key, collection string) (bool, error)
}
