package jsonstore_test

import (
	"context"
	"encoding/json"
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

		err := store.Set(context.Background(), item.ID, item.Collection, item.Value)
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

		err := store.Set(context.Background(), item.ID, item.Collection, item.Value)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}

		// modify the data
		item.Value = json.RawMessage(`{"item": "my value changed"}`)

		err = store.Set(context.Background(), item.ID, item.Collection, item.Value)
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

		err := store.Set(context.Background(), item1.ID, item1.Collection, item1.Value)
		if err != nil {
			t.Fatalf("action: Set,  returned an error: %v", err)
		}

		// set the same key on another collection
		item2 := dbDocument{
			ID:         "item1",
			Collection: "collection2",
			Value:      json.RawMessage(`{"item": "my value2"}`),
		}

		err = store.Set(context.Background(), item2.ID, item2.Collection, item2.Value)
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
}

func TestJsonfileDelete(t *testing.T) {
}

func TestJsonfileList(t *testing.T) {
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
