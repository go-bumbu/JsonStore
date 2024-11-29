package jsonstore_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-bumbu/jsonstore"
	"github.com/google/go-cmp/cmp"
	"os"
	"path/filepath"
	"testing"
)

func TestJsonfileSet(t *testing.T) {
	store, jsonFile := getjsonFileStore(t)
	t.Run("set value", func(t *testing.T) {

		item := dbDocument{
			ID:         "item1",
			Collection: "test_set_value",
			Value:      json.RawMessage(`{"item": "my value"}`),
		}

		err := store.Set(context.Background(), item.Collection, item.ID, item.Value)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}

		data := readJsonFile(t, jsonFile)
		got := data.(map[string]interface{})["test_set_value"].(map[string]interface{})["item1"].(map[string]interface{})["item"]
		if diff := cmp.Diff(got, "my value"); diff != "" {
			t.Errorf("unexpected value (-got +want)\n%s", diff)
		}

	})

	t.Run("update value", func(t *testing.T) {
		item := dbDocument{
			ID:         "item1",
			Collection: "test_set_value",
			Value:      json.RawMessage(`{"item": "my value"}`),
		}

		err := store.Set(context.Background(), item.Collection, item.ID, item.Value)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}

		// modify the data
		item.Value = json.RawMessage(`{"item": "my value changed"}`)

		err = store.Set(context.Background(), item.Collection, item.ID, item.Value)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}
		data := readJsonFile(t, jsonFile)
		got := data.(map[string]interface{})["test_set_value"].(map[string]interface{})["item1"].(map[string]interface{})["item"]
		if diff := cmp.Diff(got, "my value changed"); diff != "" {
			t.Errorf("unexpected value (-got +want)\n%s", diff)
		}

	})
	t.Run("different set per collection", func(t *testing.T) {
		item1 := dbDocument{
			ID:         "item1",
			Collection: "collection1",
			Value:      json.RawMessage(`{"item": "my value1"}`),
		}

		err := store.Set(context.Background(), item1.Collection, item1.ID, item1.Value)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}

		// set the same key on another collection
		item2 := dbDocument{
			ID:         "item1",
			Collection: "collection2",
			Value:      json.RawMessage(`{"item": "my value2"}`),
		}

		err = store.Set(context.Background(), item2.Collection, item2.ID, item2.Value)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}

		// Retrieve both the document from the database and verify its content
		for _, item := range []dbDocument{item1, item2} {

			data := readJsonFile(t, jsonFile)
			got := data.(map[string]interface{})[item.Collection].(map[string]interface{})[item.ID].(map[string]interface{})["item"]

			var want map[string]interface{}

			// Unmarshal the raw JSON into the map
			if err := json.Unmarshal(item.Value, &want); err != nil {
				t.Fatalf("Error unmarshaling JSON: %v", err)
			}

			if diff := cmp.Diff(got, want["item"]); diff != "" {
				t.Errorf("unexpected value (-got +want)\n%s", diff)
			}
		}
	})

}

func TestJsonfileGet(t *testing.T) {
	store, file := getjsonFileStore(t)
	_ = file
	t.Run("get value", func(t *testing.T) {
		item := dbDocument{
			ID:         "item1",
			Collection: "test_set_value",
			Value:      json.RawMessage(`{"item":"my value"}`),
		}

		err := store.Set(context.Background(), item.Collection, item.ID, item.Value)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}

		var got json.RawMessage
		err = store.Get(context.Background(), item.Collection, item.ID, &got)
		if err != nil {
			t.Fatalf("action: Get,  returned an error: %v", err)
		}
		if diff := cmp.Diff(got, item.Value); diff != "" {
			t.Errorf("unexpected value (-got +want)\n%s", diff)
		}
	})

	t.Run("get updated value", func(t *testing.T) {
		item := dbDocument{
			ID:         "item1",
			Collection: "test_set_value",
			Value:      json.RawMessage(`{"item":"my value"}`),
		}

		err := store.Set(context.Background(), item.Collection, item.ID, item.Value)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}

		// Update the document with new data
		item.Value = json.RawMessage(`{"item":"updated value"}`)
		err = store.Set(context.Background(), item.Collection, item.ID, item.Value)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}

		var got json.RawMessage
		err = store.Get(context.Background(), item.Collection, item.ID, &got)
		if err != nil {
			t.Fatalf("action: Get,  returned an error: %v", err)
		}
		if diff := cmp.Diff(got, item.Value); diff != "" {
			t.Errorf("unexpected value (-got +want)\n%s", diff)
		}
	})

	t.Run("get differnt entries per collection", func(t *testing.T) {
		item1 := dbDocument{
			ID:         "item1",
			Collection: "collection1",
			Value:      json.RawMessage(`{"item":"my value1"}`),
		}

		err := store.Set(context.Background(), item1.Collection, item1.ID, item1.Value)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}
		item1 = dbDocument{
			ID:         "item2",
			Collection: "collection1",
			Value:      json.RawMessage(`{"item":"my value2_on col1"}`),
		}
		err = store.Set(context.Background(), item1.Collection, item1.ID, item1.Value)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}

		// set the same key on another collection
		item2 := dbDocument{
			ID:         "item1",
			Collection: "collection2",
			Value:      json.RawMessage(`{"item":"my value2"}`),
		}

		err = store.Set(context.Background(), item2.Collection, item2.ID, item2.Value)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}

		// retrieve item 1
		var got json.RawMessage
		err = store.Get(context.Background(), item1.Collection, item1.ID, &got)
		if err != nil {
			t.Fatalf("action: Get,  returned an error: %v", err)
		}
		if diff := cmp.Diff(got, item1.Value); diff != "" {
			t.Errorf("unexpected value (-got +want)\n%s", diff)
		}

		// retrieve item 2
		err = store.Get(context.Background(), item2.Collection, item2.ID, &got)
		if err != nil {
			t.Fatalf("action: Get,  returned an error: %v", err)
		}
		if diff := cmp.Diff(got, item2.Value); diff != "" {
			t.Errorf("unexpected value (-got +want)\n%s", diff)
		}
		//readJsonFile(t, file)
	})
	// TODO add test for getting data from file, e.g. store and use a new handler to get
}

func TestJsonfileDelete(t *testing.T) {
	store, jsonFile := getjsonFileStore(t)
	_ = jsonFile

	t.Run("delete value", func(t *testing.T) {
		item := dbDocument{
			ID:         "key",
			Collection: "test_set_value",
			Value:      json.RawMessage(`{"item": "my value"}`),
		}

		err := store.Set(context.Background(), item.Collection, item.ID, item.Value)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}

		deleted, err := store.Delete(context.Background(), item.Collection, item.ID)
		if err != nil {
			t.Fatalf("action: Get,  returned an error: %v", err)
		}
		if deleted == false {
			t.Errorf("expect Delete to affect one entry, but got false for no rows affected")
		}

		data := readJsonFile(t, jsonFile)
		got := data.(map[string]interface{})["test_set_value"].(map[string]interface{})["key"]
		if got != nil {
			t.Errorf("expected item with key \"key\" to no NOT exists but got: %v", got)
		}
	})

	t.Run("delete non existent item", func(t *testing.T) {
		item := dbDocument{
			ID:         "item1",
			Collection: "test_set_value",
			Value:      json.RawMessage(`{"item": "my value"}`),
		}

		deleted, err := store.Delete(context.Background(), item.Collection, item.ID)
		if err != nil {
			t.Fatalf("action: Get,  returned an error: %v", err)
		}
		if deleted == true {
			t.Errorf("expect Delete to NOT affect any entry, but got true for 1 rows affected")
		}

		data := readJsonFile(t, jsonFile)
		got := data.(map[string]interface{})["test_set_value"].(map[string]interface{})["key"]
		if got != nil {
			t.Errorf("expected item with key \"key\" to no NOT exists but got: %v", got)
		}
	})
}

func TestJsonfileList(t *testing.T) {

	store, jsonFile := getjsonFileStore(t)
	_ = jsonFile
	var err error

	// add 3 items to collection 1

	for i := 1; i <= 3; i++ {
		err = store.Set(context.Background(), "col1", fmt.Sprintf("item%d", i),
			json.RawMessage(fmt.Sprintf("{\"item\": \"collection1 item%d\"}", i)))
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}
	}
	// add 5 items to collection 1
	for i := 1; i <= 5; i++ {
		err = store.Set(context.Background(), "col2", fmt.Sprintf("item%d", i),
			json.RawMessage(fmt.Sprintf("{\"item\": \"collection2 item%d\"}", i)))
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}
	}

	t.Run("asert collection length", func(t *testing.T) {
		_, len1, err := store.List(context.Background(), "col1", 0, 1)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}
		if diff := cmp.Diff(len1, int64(3)); diff != "" {
			t.Errorf("unexpected value (-got +want)\n%s", diff)
		}

		_, len2, err := store.List(context.Background(), "col2", 0, 1)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}
		if diff := cmp.Diff(len2, int64(5)); diff != "" {
			t.Errorf("unexpected value (-got +want)\n%s", diff)
		}
	})

	t.Run("asert listed items", func(t *testing.T) {
		items, _, err := store.List(context.Background(), "col1", 0, 1)
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
		items, _, err := store.List(context.Background(), "col1", 2, 1)
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
		items, _, err := store.List(context.Background(), "col1", 2, 2)
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

func TestJsonfileConcurrency(t *testing.T) {
}

func getjsonFileStore(t *testing.T) (*jsonstore.FileStore, string) {
	tempdir := t.TempDir()
	file := filepath.Join(tempdir, "test.json")
	jfs, err := jsonstore.NewFileStore(file)
	if err != nil {
		t.Fatal(err)
	}
	return jfs, file
}

func readJsonFile(t *testing.T, file string) interface{} {
	fileContent, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("Error reading file: %v", err)
	}

	//fmt.Println("Raw JSON content:")
	//fmt.Println(string(fileContent))

	var jsonData interface{}
	if err := json.Unmarshal(fileContent, &jsonData); err != nil {
		t.Fatalf("Error unmarshaling JSON: %v", err)
	}

	//fmt.Printf("JSON Data: %+v\n", jsonData)
	return jsonData
}
