package jsonstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
)

// dbDocument represents the data columns to be stored using gorm
type dbDocument struct {
	ID         string          `gorm:"primaryKey"`
	Collection string          `gorm:"primaryKey"`
	Value      json.RawMessage `gorm:"type:json"`
}

func (d dbDocument) Validate() error {
	if d.ID == "" {
		return fmt.Errorf("id cannot be empty")
	}
	if d.Collection == "" {
		return fmt.Errorf("collection cannot be empty")
	}
	return nil
}

const columnId = "ID"
const columnValue = "value"
const columnCollection = "collection"

// DbStore does a setup to use a DB to store kv data
type DbStore struct {
	db *gorm.DB
}

// make sure the DB store fulfills the JsonStoreList interface
var _ JsonStorer = &DbStore{}

const DefaultCollection = "default"

func NewDbStore(db *gorm.DB) (*DbStore, error) {
	err := db.AutoMigrate(&dbDocument{})
	if err != nil {
		return nil, err
	}
	store := DbStore{
		db: db,
	}
	return &store, nil
}

func (store *DbStore) Set(ctx context.Context, collection, key string, value json.RawMessage) error {
	if collection == "" {
		collection = DefaultCollection
	}
	doc := dbDocument{
		ID:         key,
		Collection: collection,
		Value:      value,
	}

	err := doc.Validate()
	if err != nil {
		return err
	}

	err = store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Save(&doc).Error; err != nil {
			return fmt.Errorf("failed to save document: %v", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (store *DbStore) Get(ctx context.Context, collection, key string, value *json.RawMessage) error {
	if collection == "" {
		collection = DefaultCollection
	}

	item := dbDocument{}
	err := store.db.Model(&dbDocument{}).
		Select(columnValue).
		WithContext(ctx).
		Where(fmt.Sprintf("%s = ? AND %s = ?", columnId, columnCollection), key, collection).
		First(&item).Error
	*value = item.Value

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return fmt.Errorf("failed to retrieve document: %v", err)
	}
	return nil
}

const MaxListItems = 20

func (store *DbStore) List(ctx context.Context, collection string, limit, page int) (map[string]json.RawMessage, int64, error) {
	if collection == "" {
		collection = DefaultCollection
	}
	if limit == 0 || limit > MaxListItems {
		limit = MaxListItems
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	var count int64
	// Perform a count query based on the collection column.
	err := store.db.Model(&dbDocument{}).
		WithContext(ctx).
		Where(fmt.Sprintf("%s = ? ", columnCollection), collection).
		Count(&count).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count items in collection %s: %v", collection, err)
	}

	items := []dbDocument{}
	// Query the database to get all the documents in the collection
	err = store.db.
		Model(&dbDocument{}).
		WithContext(ctx).
		Where(fmt.Sprintf("%s = ? ", columnCollection), collection).
		Order("id ASC").
		Limit(limit).
		Offset(offset).
		Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve documents: %v", err)
	}

	result := map[string]json.RawMessage{}
	for _, item := range items {
		result[item.ID] = item.Value
	}
	return result, count, nil
}

func (store *DbStore) Delete(ctx context.Context, collection, key string) (bool, error) {
	if collection == "" {
		collection = DefaultCollection
	}
	result := store.db.
		WithContext(ctx).
		Where(fmt.Sprintf("%s = ? AND %s = ?", columnId, columnCollection), key, collection).
		Delete(&dbDocument{})

	// Check if there was an error during the deletion
	if result.Error != nil {
		return false, fmt.Errorf("failed to delete document with ID %s: %v", key, result.Error)
	}
	switch result.RowsAffected {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return true, fmt.Errorf("unexpected amount of deleted rows, expected 1 or 0, got: %d", result.RowsAffected)
	}

}
