package jsonstore_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-bumbu/jsonstore"
	"github.com/google/go-cmp/cmp"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetKey(t *testing.T) {
	tcs := []struct {
		name           string
		urlPath        string
		expectedKey    string
		expectedErrMsg string
	}{
		{
			name:        "Valid key in path",
			urlPath:     "/db/collection/key123",
			expectedKey: "key123",
		},
		{
			name:           "Path ending with slash",
			urlPath:        "/db/collection/",
			expectedKey:    "",
			expectedErrMsg: "",
		},
		{
			name:           "Invalid path format",
			urlPath:        "/",
			expectedKey:    "",
			expectedErrMsg: "invalid path format; no parts in path",
		},
		{
			name:        "Root-level key",
			urlPath:     "/keyOnly",
			expectedKey: "keyOnly",
		},
	}

	for _, tt := range tcs {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.urlPath, nil)

			key := jsonstore.GetReqKey(req)
			if key != tt.expectedKey {
				t.Errorf("Expected key %q, got %q", tt.expectedKey, key)
			}
		})
	}
}

func TestHandlerGet(t *testing.T) {
	mockStorer := &MockStorer{
		Data: make(map[string]map[string]json.RawMessage),
	}
	handler := jsonstore.Handler{
		HttpStorer: jsonstore.HttpStorer{Storer: mockStorer},
		Collection: "test_collection",
	}

	// Pre-populate mock data
	mockStorer.Data["test_collection"] = map[string]json.RawMessage{
		"key1": json.RawMessage(`{"foo":"bar"}`),
	}

	t.Run("Get - valid request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/key1", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		res := rec.Result()
		defer res.Body.Close()

		// Check status code
		if res.StatusCode != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, res.StatusCode)
		}

		// Check response body
		body, _ := io.ReadAll(res.Body)
		expected := `{"foo":"bar"}`
		if diff := cmp.Diff(string(body), expected); diff != "" {
			t.Errorf("unexpected response body (-want +got):\n%s", diff)
		}
	})

	t.Run("Get - key not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/keyNotFound", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		res := rec.Result()
		defer res.Body.Close()

		// Check status code
		if res.StatusCode != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, res.StatusCode)
		}

		// Check error message
		body, _ := io.ReadAll(res.Body)
		expectedError := "Failed to retrieve item: item not found\n"
		if string(body) != expectedError {
			t.Errorf("expected body %q, got %q", expectedError, string(body))
		}
	})

	t.Run("Get - error from storage", func(t *testing.T) {
		mockStorer.Err = fmt.Errorf("storage error")
		req := httptest.NewRequest(http.MethodGet, "/key1", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		res := rec.Result()
		defer res.Body.Close()

		// Check status code
		if res.StatusCode != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, res.StatusCode)
		}

		// Check error message
		body, _ := io.ReadAll(res.Body)
		expectedError := "Failed to retrieve item: storage error\n"
		if string(body) != expectedError {
			t.Errorf("expected body %q, got %q", expectedError, string(body))
		}
	})
}

func TestHandlerSet(t *testing.T) {
	mockStorer := &MockStorer{
		Data: make(map[string]map[string]json.RawMessage),
	}
	handler := jsonstore.Handler{
		HttpStorer: jsonstore.HttpStorer{Storer: mockStorer},
		Collection: "test_collection",
	}

	t.Run("Set - valid request", func(t *testing.T) {
		reqBody := []byte(`{"foo":"bar"}`)
		req := httptest.NewRequest(http.MethodPost, "/key1", bytes.NewReader(reqBody))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		res := rec.Result()
		defer res.Body.Close()

		// Check status code
		if res.StatusCode != http.StatusCreated {
			t.Errorf("expected status %d, got %d", http.StatusCreated, res.StatusCode)
		}

		// Check stored data
		expectedData := json.RawMessage(`{"foo":"bar"}`)
		if diff := cmp.Diff(mockStorer.Data["test_collection"]["key1"], expectedData); diff != "" {
			t.Errorf("unexpected stored data (-want +got):\n%s", diff)
		}
	})

	t.Run("Set - error reading request body", func(t *testing.T) {
		// Use a broken reader

		brokenReader := BrokenReader{}
		req := httptest.NewRequest(http.MethodPost, "/key1", &brokenReader)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		res := rec.Result()
		defer res.Body.Close()

		// Check status code
		if res.StatusCode != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, res.StatusCode)
		}

		// Check error message
		body, _ := io.ReadAll(res.Body)
		expectedError := "Failed to read request body\n"
		if string(body) != expectedError {
			t.Errorf("expected body %q, got %q", expectedError, string(body))
		}
	})

	t.Run("Set - storage error", func(t *testing.T) {
		mockStorer.Err = fmt.Errorf("storage error")
		reqBody := []byte(`{"baz":"qux"}`)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(reqBody))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		res := rec.Result()
		defer res.Body.Close()

		// Check status code
		if res.StatusCode != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, res.StatusCode)
		}

		// Check error message
		body, _ := io.ReadAll(res.Body)
		expectedError := "Failed to store data: storage error\n"
		if string(body) != expectedError {
			t.Errorf("expected body %q, got %q", expectedError, string(body))
		}

		// Ensure data was not stored
		if _, exists := mockStorer.Data["test_collection"]["key2"]; exists {
			t.Errorf("data should not have been stored due to error")
		}
	})
}

func TestHandlerDelete(t *testing.T) {
	t.Run("Delete - successful", func(t *testing.T) {

		mockStorer := &MockStorer{
			Data: map[string]map[string]json.RawMessage{
				"test_collection": {
					"key1": []byte(`{"foo":"bar"}`),
				},
			},
		}
		handler := jsonstore.Handler{
			HttpStorer: jsonstore.HttpStorer{Storer: mockStorer},
			Collection: "test_collection",
		}

		req := httptest.NewRequest(http.MethodDelete, "/key1", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		res := rec.Result()
		defer res.Body.Close()

		// Verify the response status code
		if res.StatusCode != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, res.StatusCode)
		}

		// Ensure the item is deleted
		if _, exists := mockStorer.Data["test_collection"]["key1"]; exists {
			t.Errorf("expected item to be deleted")
		}

		// Check response body
		body, _ := io.ReadAll(res.Body)
		expectedResponse := ""
		if diff := cmp.Diff(string(body), expectedResponse); diff != "" {
			t.Errorf("unexpected response body (-want +got):\n%s", diff)
		}
	})

	t.Run("Delete - item not found", func(t *testing.T) {

		mockStorer := &MockStorer{
			Data: map[string]map[string]json.RawMessage{
				"test_collection": {
					"key1": []byte(`{"foo":"bar"}`),
				},
			},
		}
		handler := jsonstore.Handler{
			HttpStorer: jsonstore.HttpStorer{Storer: mockStorer},
			Collection: "test_collection",
		}

		req := httptest.NewRequest(http.MethodDelete, "/key2", nil) // Non-existent key
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		res := rec.Result()
		defer res.Body.Close()

		// Check if the correct status code and error message are returned
		if res.StatusCode != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, res.StatusCode)
		}

		// Check the response body for the "Item not found" message
		body, _ := io.ReadAll(res.Body)
		expectedError := "Item not found\n"
		if diff := cmp.Diff(string(body), expectedError); diff != "" {
			t.Errorf("unexpected error message (-want +got):\n%s", diff)
		}
	})

	t.Run("Delete - error during deletion", func(t *testing.T) {
		mockStorer := &MockStorer{
			Data: map[string]map[string]json.RawMessage{
				"test_collection": {
					"key1": []byte(`{"foo":"bar"}`),
				},
			},
		}
		handler := jsonstore.Handler{
			HttpStorer: jsonstore.HttpStorer{Storer: mockStorer},
			Collection: "test_collection",
		}

		mockStorer.Err = fmt.Errorf("storage error") // Simulate an error during deletion
		req := httptest.NewRequest(http.MethodDelete, "/key1", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		res := rec.Result()
		defer res.Body.Close()

		// Check if the correct status code and error message are returned
		if res.StatusCode != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, res.StatusCode)
		}

		// Check the error message in the response body
		body, _ := io.ReadAll(res.Body)
		expectedError := "Failed to delete data: storage error\n"
		if diff := cmp.Diff(string(body), expectedError); diff != "" {
			t.Errorf("unexpected error message (-want +got):\n%s", diff)
		}
	})
}
func TestHandlerList(t *testing.T) {
	mockStorer := &MockStorer{
		Data: map[string]map[string]json.RawMessage{
			"test_collection": {
				"key1": []byte(`{"name":"item1"}`),
				"key2": []byte(`{"name":"item2"}`),
			},
		},
	}

	handler := jsonstore.Handler{
		HttpStorer: jsonstore.HttpStorer{Storer: mockStorer},
		Collection: "test_collection",
	}

	t.Run("List - default pagination", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test-collection/", nil)
		rec := httptest.NewRecorder()

		handler.List(rec, req, "test_collection")

		res := rec.Result()
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, res.StatusCode)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		expectedTotal := int64(2)
		if int64(response["total"].(float64)) != expectedTotal {
			t.Errorf("expected total %d, got %d", expectedTotal, int64(response["total"].(float64)))
		}
		if int(response["page"].(float64)) != 1 {
			t.Errorf("expected page 1, got %d", int(response["page"].(float64)))
		}
		if int(response["limit"].(float64)) != 10 {
			t.Errorf("expected limit 10, got %d", int(response["limit"].(float64)))
		}
	})

	t.Run("List - custom pagination", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test-collection/?limit=1&page=2", nil)
		rec := httptest.NewRecorder()

		handler.List(rec, req, "test_collection")

		res := rec.Result()
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, res.StatusCode)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if int(response["page"].(float64)) != 2 {
			t.Errorf("expected page 2, got %d", int(response["page"].(float64)))
		}
		if int(response["limit"].(float64)) != 1 {
			t.Errorf("expected limit 1, got %d", int(response["limit"].(float64)))
		}
		items := response["items"].(map[string]interface{})
		if len(items) != 1 {
			t.Errorf("expected 1 item, got %d", len(items))
		}
	})

	t.Run("List - error fetching items", func(t *testing.T) {
		mockStorer.Err = fmt.Errorf("storage error") // Simulate an error during deletion

		req := httptest.NewRequest(http.MethodGet, "/list", nil)
		rec := httptest.NewRecorder()

		handler.List(rec, req, "test_collection")

		res := rec.Result()
		defer res.Body.Close()

		// Check status code
		if res.StatusCode != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, res.StatusCode)
		}

		// Check error message
		body := rec.Body.String()
		expectedError := "Failed to fetch items: storage error\n"
		if body != expectedError {
			t.Errorf("expected error %q, got %q", expectedError, body)
		}
	})

}

// BrokenReader is a reader that always returns an error to simulate read errors
type BrokenReader struct{}

func (r *BrokenReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("broken reader error")
}

type MockStorer struct {
	Data map[string]map[string]json.RawMessage
	Err  error
}

func (m *MockStorer) Set(ctx context.Context, collection, key string, value json.RawMessage) error {
	if m.Err != nil {
		return m.Err
	}
	if m.Data == nil {
		m.Data = make(map[string]map[string]json.RawMessage)
	}
	if m.Data[collection] == nil {
		m.Data[collection] = make(map[string]json.RawMessage)
	}
	m.Data[collection][key] = value
	return nil
}

func (m *MockStorer) Get(ctx context.Context, collection, key string, value *json.RawMessage) error {
	if m.Err != nil {
		return m.Err
	}
	if _, ok := m.Data[collection]; ok {
		if val, ok2 := m.Data[collection][key]; ok2 {
			*value = val
			return nil
		} else {
			return jsonstore.ItemNotFoundErr
		}
	} else {
		return jsonstore.CollectionNotFoundErr
	}
}

func (m *MockStorer) Delete(ctx context.Context, collection, key string) (bool, error) {
	if m.Err != nil {
		return false, m.Err
	}
	if col, ok := m.Data[collection]; ok {
		if _, ok := col[key]; ok {
			delete(col, key)
			return true, nil
		}
	}
	return false, nil
}

func (m *MockStorer) List(ctx context.Context, collection string, limit, page int) (map[string]json.RawMessage, int64, error) {
	items := make(map[string]json.RawMessage)
	if m.Err != nil {
		return items, 0, m.Err
	}
	col, exists := m.Data[collection]
	if !exists {
		return nil, 0, nil
	}

	count := int64(len(col))
	start := (page - 1) * limit
	end := start + limit

	i := 0
	for k, v := range col {
		if i >= start && i < end {
			items[k] = v
		}
		i++
	}
	return items, count, nil
}
