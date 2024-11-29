package jsonstore

import (
	"context"
	"encoding/json"
	"errors"
)

// JsonStorer interface implements the needed actions to Store and retrieve json Values identified by a key
type JsonStorer interface {
	Set(ctx context.Context, collection, key string, value json.RawMessage) error
	Get(ctx context.Context, collection, key string, value *json.RawMessage) error
	Delete(ctx context.Context, collection, key string) (bool, error)
	List(ctx context.Context, collection string, limit, page int) (map[string]json.RawMessage, int64, error)
}

// Todo, verify that the implementations return the proper errors

var CollectionNotFoundErr = errors.New("collection not found")
var ItemNotFoundErr = errors.New("item not found")
