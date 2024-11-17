package jsonstore

import (
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

func NewDbStore(db *gorm.DB) (*DbStore, error) {
	err := db.AutoMigrate(&dbDocument{})
	if err != nil {
		return nil, err
	}
	return &DbStore{db: db}, nil
}

const defColName = "default"

func (store *DbStore) Use(collection string) *DbCollection {
	if collection == "" {
		collection = defColName
	}
	return &DbCollection{
		db:  store.db,
		col: collection,
	}
}

// DbCollection represents a single collection of data
type DbCollection struct {
	db  *gorm.DB
	col string
}

func (store *DbCollection) Set(key string, value json.RawMessage) error {
	doc := dbDocument{
		ID:         key,
		Collection: store.col,
		Value:      value,
	}

	err := doc.Validate()
	if err != nil {
		return err
	}

	// Create or update document
	if err = store.db.Save(&doc).Error; err != nil {
		return fmt.Errorf("failed to save document: %v", err)
	}
	return nil
}

func (store *DbCollection) Get(key string, value *json.RawMessage) error {

	item := dbDocument{}

	err := store.db.Model(&dbDocument{}).
		Select(columnValue).
		Where(fmt.Sprintf("%s = ? AND %s = ?", columnId, columnCollection), key, store.col).
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

func (store *DbCollection) List(limit, page int) (map[string]json.RawMessage, int64, error) {

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
		Where(fmt.Sprintf("%s = ? ", columnCollection), store.col).
		Count(&count).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count items in collection %s: %v", store.col, err)
	}

	items := []dbDocument{}
	// Query the database to get all the documents in the collection
	err = store.db.
		Model(&dbDocument{}).
		Where(fmt.Sprintf("%s = ? ", columnCollection), store.col).
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

func (store *DbCollection) Delete(key string) (int64, error) {
	// Perform the delete operation based on the document's ID and collection name
	result := store.db.
		Where(fmt.Sprintf("%s = ? AND %s = ?", columnId, columnCollection), key, store.col).
		Delete(&dbDocument{})

	// Check if there was an error during the deletion
	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete document with ID %s: %v", key, result.Error)
	}
	return result.RowsAffected, nil
}
