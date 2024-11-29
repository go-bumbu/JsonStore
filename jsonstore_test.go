package jsonstore_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-bumbu/jsonstore"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"path/filepath"
	"testing"
)

func newJsonFile(t *testing.T) *jsonstore.FileStore {
	store, _ := getjsonFileStore(t)
	return store
}

func newDbStore(t *testing.T) *jsonstore.DbStore {

	// NOTE: in memory database does not work well with concurrency, if not used with shared
	tmpDir := t.TempDir()
	db, err := gorm.Open(sqlite.Open(filepath.Join(tmpDir, "testdb.sqlite")), &gorm.Config{
		Logger: logger.Discard, // discard in tests
	})

	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying DB: %v", err)
	}

	t.Cleanup(func() {
		sqlDB.Close() // Ensure all connections are closed after the test
	})

	store, err := jsonstore.NewDbStore(db)
	if err != nil {
		t.Fatalf("NewDbStore returned an error: %v", err)
	}
	return store
}

func TestJsonStorerImplementations(t *testing.T) {
	implementations := []struct {
		name   string
		storer jsonstore.JsonStorer
	}{

		{"mock", &MockStorer{}},
		{"jsonfile", newJsonFile(t)},
		{"db", newDbStore(t)},
	}

	collection := "test-collection"
	key := "test-key"
	value := json.RawMessage(`{"name":"test-item"}`)

	for _, impl := range implementations {
		t.Run(impl.name+" - Set and Get", func(t *testing.T) {
			ctx := context.Background()

			if err := impl.storer.Set(ctx, collection, key, value); err != nil {
				t.Fatalf("Set failed: %v", err)
			}

			var result json.RawMessage
			err := impl.storer.Get(ctx, collection, key, &result)
			if err != nil {
				t.Fatalf("Get failed: %v", err)
			}

			if string(result) != string(value) {
				t.Errorf("expected %s, got %s", value, result)
			}
		})

		t.Run(impl.name+" - Delete", func(t *testing.T) {
			ctx := context.Background()

			deleted, err := impl.storer.Delete(ctx, collection, key)
			if err != nil {
				t.Fatalf("Delete failed: %v", err)
			}
			if !deleted {
				t.Errorf("expected true, got false")
			}
		})

		t.Run(impl.name+" - List", func(t *testing.T) {
			ctx := context.Background()

			for i := 0; i < 9; i++ {
				impl.storer.Set(ctx, collection, fmt.Sprintf("%s-%d", key, i), value)
			}

			items, total, err := impl.storer.List(ctx, collection, 2, 2)
			if err != nil {
				t.Fatalf("List failed: %v", err)
			}

			expectedTotal := int64(9)
			if total != expectedTotal {
				t.Errorf("expected total %d, got %d", expectedTotal, total)
			}
			expectedItems := 2
			if len(items) != expectedItems {
				t.Errorf("expected items %d items, got %d", expectedTotal, total)
			}

		})
	}
}
