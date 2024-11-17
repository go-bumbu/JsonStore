package jsonstore_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"os"
	"testing"
	"time"

	"github.com/go-bumbu/jsonstore"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var _ = spew.Dump //keep the dependency

func getTargetDBs(t *testing.T) map[string]*gorm.DB {
	databases := make(map[string]*gorm.DB)

	sqliteDb := newSqliteDb(t)
	databases["sqlite"] = sqliteDb

	_, skipTestCont := os.LookupEnv("SKIP_TESTCONTAINERS")
	if testing.Short() || skipTestCont {
		return databases
	}

	// discard testcontainer messages
	testcontainers.Logger = testcontainers.TestLogger(t)

	// Initialize MySQL and add it to the map
	_, skipMysql := os.LookupEnv("SKIP_MYSQL")
	if !skipMysql {
		mysqlDb := newMySQLDb(t)
		databases["mysql"] = mysqlDb
	}

	// Initialize PostgresSQL and add it to the map
	_, skipPostgres := os.LookupEnv("SKIP_POSTGRES")
	if !skipPostgres {
		postgresDb := newPostgresDb(t)
		databases["postgres"] = postgresDb
	}

	return databases
}

func newSqliteDb(t *testing.T) *gorm.DB {
	// Set up an in-memory SQLite database for testing
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		//Logger: logger.Discard, // discard in tests
	})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	return db
}

func newMySQLDb(t *testing.T) *gorm.DB {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "mysql:8.0",
		ExposedPorts: []string{"3306/tcp"},
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": "password",
			"MYSQL_DATABASE":      "testdb",
			"MYSQL_USER":          "testuser",
			"MYSQL_PASSWORD":      "password",
		},
		WaitingFor: wait.ForListeningPort("3306/tcp").WithStartupTimeout(60 * time.Second),
	}

	mysqlContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start MySQL container: %v", err)
	}

	host, err := mysqlContainer.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get MySQL container host: %v", err)
	}

	port, err := mysqlContainer.MappedPort(ctx, "3306")
	if err != nil {
		t.Fatalf("failed to get MySQL container port: %v", err)
	}

	dsn := fmt.Sprintf("testuser:password@tcp(%s:%s)/testdb?charset=utf8mb4&parseTime=True&loc=Local", host, port.Port())
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("failed to connect to MySQL test database: %v", err)
	}

	t.Cleanup(func() {
		if err := mysqlContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate MySQL container: %v", err)
		}
	})

	return db
}

func newPostgresDb(t *testing.T) *gorm.DB {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:13",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "password",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}

	postgresContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start PostgreSQL container: %v", err)
	}

	host, err := postgresContainer.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get PostgreSQL container host: %v", err)
	}

	port, err := postgresContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get PostgreSQL container port: %v", err)
	}

	dsn := fmt.Sprintf("host=%s port=%s user=testuser dbname=testdb password=password sslmode=disable", host, port.Port())
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("failed to connect to PostgreSQL test database: %v", err)
	}

	t.Cleanup(func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate PostgreSQL container: %v", err)
		}
	})

	return db
}

// dbDocument is a copy of the object used internally by gorm in the db.go package
// this is used for testing purposes only
type dbDocument struct {
	ID         string          `gorm:"primaryKey"`
	Collection string          `gorm:"primaryKey"`
	Value      json.RawMessage `gorm:"type:json"`
}

// group all test executions on a single test, to re-use the testcontainer instance
// this test will run all the db tests on the variety of db implementations
func TestDb(t *testing.T) {
	dbs := getTargetDBs(t)
	for dbName, db := range dbs {
		t.Run(dbName, func(t *testing.T) {
			t.Run("test new DB", func(t *testing.T) {
				// verify to create a new DB
				testNewDb(t, db)
			})

			t.Run("test action set", func(t *testing.T) {
				// verify that we can set data
				testActionSet(t, db)
			})

			t.Run("test action get", func(t *testing.T) {
				testActionGet(t, db)
			})

			t.Run("test action List", func(t *testing.T) {
				testActionList(t, db)
			})
		})
	}
}

func testNewDb(t *testing.T, db *gorm.DB) {
	_, err := jsonstore.NewDbStore(db)
	if err != nil {
		t.Fatalf("NewDbStore returned an error: %v", err)
	}

	// dbDocument represents the data columns to be stored using gorm
	type dbDocument struct {
		ID         string          `gorm:"primaryKey"`
		Collection string          `gorm:"primaryKey"`
		Value      json.RawMessage `gorm:"type:jsonb"`
	}

	// Verify the dbDocument table was created by checking for its existence
	if !db.Migrator().HasTable(&dbDocument{}) {
		t.Fatal("expected dbDocument table to be created, but it does not exist")
	}
}

func testActionSet(t *testing.T, db *gorm.DB) {

	t.Run("set value", func(t *testing.T) {
		store, err := jsonstore.NewDbStore(db)
		if err != nil {
			t.Fatalf("NewDbStore returned an error: %v", err)
		}
		item := dbDocument{
			ID:         "item1",
			Collection: "test_set_value",
			Value:      json.RawMessage(`{"item": "my value"}`),
		}

		err = store.Set(item.ID, item.Collection, item.Value)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}

		// Retrieve the document from the database and verify its content
		var got dbDocument
		if err := db.First(&got, "ID = ? AND Collection = ?", item.ID, item.Collection).Error; err != nil {
			t.Fatalf("failed to retrieve document: %v", err)
		}

		if diff := cmp.Diff(got, item); diff != "" {
			t.Errorf("unexpected value (-got +want)\n%s", diff)
		}
	})

	t.Run("update value", func(t *testing.T) {
		store, err := jsonstore.NewDbStore(db)
		if err != nil {
			t.Fatalf("NewDbStore returned an error: %v", err)
		}
		item := dbDocument{
			ID:         "item1",
			Collection: "test_update_value",
			Value:      json.RawMessage(`{"item": "my value"}`),
		}

		err = store.Set(item.ID, item.Collection, item.Value)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}

		// Update the document with new data
		item.Value = json.RawMessage(`{"item": "updated value"}`)
		err = store.Set(item.ID, item.Collection, item.Value)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}

		// Retrieve the document from the database and verify its content
		var got dbDocument
		if err := db.First(&got, "ID = ? AND Collection = ?", item.ID, item.Collection).Error; err != nil {
			t.Fatalf("failed to retrieve document: %v", err)
		}

		if diff := cmp.Diff(got, item); diff != "" {
			t.Errorf("unexpected value (-got +want)\n%s", diff)
		}
	})

	t.Run("different entries per collection", func(t *testing.T) {
		store, err := jsonstore.NewDbStore(db)
		if err != nil {
			t.Fatalf("NewDbStore returned an error: %v", err)
		}
		item1 := dbDocument{
			ID:         "item1",
			Collection: "collection1",
			Value:      json.RawMessage(`{"item": "my value1"}`),
		}

		err = store.Set(item1.ID, item1.Collection, item1.Value)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}

		// set the same key on another collection
		item2 := dbDocument{
			ID:         "item1",
			Collection: "collection2",
			Value:      json.RawMessage(`{"item": "my value2"}`),
		}

		err = store.Set(item2.ID, item2.Collection, item2.Value)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}

		// Retrieve both the document from the database and verify its content
		for _, item := range []dbDocument{item1, item2} {
			var got dbDocument
			if err := db.First(&got, "ID = ? AND Collection = ?", item.ID, item.Collection).Error; err != nil {
				t.Fatalf("failed to retrieve document: %v", err)
			}

			if diff := cmp.Diff(got, item); diff != "" {
				t.Errorf("unexpected value (-got +want)\n%s", diff)
			}
		}

	})

}

func testActionGet(t *testing.T, db *gorm.DB) {

	t.Run("get value", func(t *testing.T) {
		store, err := jsonstore.NewDbStore(db)
		if err != nil {
			t.Fatalf("NewDbStore returned an error: %v", err)
		}
		item := dbDocument{
			ID:         "item1",
			Collection: "test_set_value",
			Value:      json.RawMessage(`{"item": "my value"}`),
		}

		err = store.Set(item.ID, item.Collection, item.Value)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}

		var got json.RawMessage
		err = store.Get(item.ID, item.Collection, &got)
		if err != nil {
			t.Fatalf("action: Get,  returned an error: %v", err)
		}
		if diff := cmp.Diff(got, item.Value); diff != "" {
			t.Errorf("unexpected value (-got +want)\n%s", diff)
		}
	})

	t.Run("get updated value", func(t *testing.T) {
		store, err := jsonstore.NewDbStore(db)
		if err != nil {
			t.Fatalf("NewDbStore returned an error: %v", err)
		}
		item := dbDocument{
			ID:         "item1",
			Collection: "test_update_value",
			Value:      json.RawMessage(`{"item": "my value"}`),
		}

		err = store.Set(item.ID, item.Collection, item.Value)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}

		// Update the document with new data
		item.Value = json.RawMessage(`{"item": "updated value"}`)
		err = store.Set(item.ID, item.Collection, item.Value)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}

		var got json.RawMessage
		err = store.Get(item.ID, item.Collection, &got)
		if err != nil {
			t.Fatalf("action: Get,  returned an error: %v", err)
		}
		if diff := cmp.Diff(got, item.Value); diff != "" {
			t.Errorf("unexpected value (-got +want)\n%s", diff)
		}
	})

	t.Run("get different entries per collection", func(t *testing.T) {
		store, err := jsonstore.NewDbStore(db)
		if err != nil {
			t.Fatalf("NewDbStore returned an error: %v", err)
		}
		item1 := dbDocument{
			ID:         "item1",
			Collection: "collection1",
			Value:      json.RawMessage(`{"item": "my value"}`),
		}

		err = store.Set(item1.ID, item1.Collection, item1.Value)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}

		// set the same key on another collection
		item2 := dbDocument{
			ID:         "item1",
			Collection: "collection2",
			Value:      json.RawMessage(`{"item": "my value2"}`),
		}

		err = store.Set(item2.ID, item2.Collection, item2.Value)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}

		// retrieve item 1
		var got json.RawMessage
		err = store.Get(item1.ID, item1.Collection, &got)
		if err != nil {
			t.Fatalf("action: Get,  returned an error: %v", err)
		}
		if diff := cmp.Diff(got, item1.Value); diff != "" {
			t.Errorf("unexpected value (-got +want)\n%s", diff)
		}

		// retrieve item 2
		err = store.Get(item2.ID, item2.Collection, &got)
		if err != nil {
			t.Fatalf("action: Get,  returned an error: %v", err)
		}
		if diff := cmp.Diff(got, item2.Value); diff != "" {
			t.Errorf("unexpected value (-got +want)\n%s", diff)
		}
	})

}

func testActionList(t *testing.T, db *gorm.DB) {
	store, err := jsonstore.NewDbStore(db)
	if err != nil {
		t.Fatalf("NewDbStore returned an error: %v", err)
	}

	// add 3 items to collection 1

	for i := 1; i <= 3; i++ {
		err = store.Set(fmt.Sprintf("item%d", i), "col1",
			json.RawMessage(fmt.Sprintf("{\"item\": \"collection1 item%d\"}", i)))
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}
	}
	// add 5 items to collection 1
	for i := 1; i <= 5; i++ {
		err = store.Set(fmt.Sprintf("item%d", i), "col2",
			json.RawMessage(fmt.Sprintf("{\"item\": \"collection2 item%d\"}", i)))
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}
	}

	t.Run("asert collection length", func(t *testing.T) {
		_, len1, err := store.List("col1", 0, 1)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}
		if diff := cmp.Diff(len1, int64(3)); diff != "" {
			t.Errorf("unexpected value (-got +want)\n%s", diff)
		}

		_, len2, err := store.List("col2", 0, 1)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}
		if diff := cmp.Diff(len2, int64(5)); diff != "" {
			t.Errorf("unexpected value (-got +want)\n%s", diff)
		}
	})

	t.Run("asert listed items", func(t *testing.T) {
		items, _, err := store.List("col1", 0, 1)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}

		want := map[string]json.RawMessage{
			"item1": json.RawMessage(`{"item": "collection1 item1"}`),
			"item2": json.RawMessage(`{"item": "collection1 item2"}`),
			"item3": json.RawMessage(`{"item": "collection1 item3"}`),
		}
		if diff := cmp.Diff(items, want); diff != "" {
			t.Errorf("unexpected value (-got +want)\n%s", diff)
		}
	})

	t.Run("asert list with limit", func(t *testing.T) {
		items, _, err := store.List("col1", 2, 1)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}

		want := map[string]json.RawMessage{
			"item1": json.RawMessage(`{"item": "collection1 item1"}`),
			"item2": json.RawMessage(`{"item": "collection1 item2"}`),
		}
		if diff := cmp.Diff(items, want); diff != "" {
			t.Errorf("unexpected value (-got +want)\n%s", diff)
		}
	})

	t.Run("asert list with limit and page", func(t *testing.T) {
		items, _, err := store.List("col1", 2, 2)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}

		want := map[string]json.RawMessage{
			"item3": json.RawMessage(`{"item": "collection1 item3"}`),
		}
		if diff := cmp.Diff(items, want); diff != "" {
			t.Errorf("unexpected value (-got +want)\n%s", diff)
		}
	})
}
