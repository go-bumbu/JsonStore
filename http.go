package jsonstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
)

// Handler is a sample implementation of an http handler that is capable of storing json data into a jsonStorer
// note that the handler intentionally extends the HttpStorer to allow more flexibility in the ServeHTTP method;
// e.g. if you want to use a different mux, like gorilla you don't need to use the basic GetReqKey function
// you might also want to implement some collection name change, e.g. to make one collection per user
// for the same endpoint.
type Handler struct {
	HttpStorer
	Collection string
}

// ServeHTTP is the main handler function
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	key := GetReqKey(r)

	switch {
	case r.Method == http.MethodPost:
		h.Set(w, r, h.Collection, key)
	case r.Method == http.MethodGet:
		if key == "" {
			h.List(w, r, h.Collection)
		} else {
			h.Get(w, r, h.Collection, key)
		}
	case r.Method == http.MethodDelete:
		h.Delete(w, r, h.Collection, key)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// GetReqKey extracts the last item from the url path to be used as key
func GetReqKey(r *http.Request) string {
	if strings.HasSuffix(r.URL.Path, "/") {
		return ""
	}
	return path.Base(r.URL.Path)
}

// HttpStorer extends the default JsonStorer and adds HTTP methods to interact with the json store
type HttpStorer struct {
	Storer JsonStorer
}

// Set handles requests to create or update a document, normally this would be a POST request
func (h *HttpStorer) Set(w http.ResponseWriter, r *http.Request, collection, key string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	err = h.Storer.Set(r.Context(), collection, key, body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to store data: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// Get handles requests to read a single item in the collection, normally this would be a GET on /path/<itemKey>
func (h *HttpStorer) Get(w http.ResponseWriter, r *http.Request, collection, key string) {
	var value json.RawMessage
	err := h.Storer.Get(r.Context(), collection, key, &value)
	if err != nil {
		if errors.Is(err, ItemNotFoundErr) {
			http.Error(w, fmt.Sprintf("Failed to retrieve item: %v", err), http.StatusNotFound)
			return
		}

		http.Error(w, fmt.Sprintf("Failed to retrieve item: %v", err), http.StatusInternalServerError)
		return
	}

	// Write the response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(value)
}

// List handles requests to read a list of items in the collection, normally this would be a GET on /path/
// note that the methods makes use of query parameters limit and page to allow for pagination
// it will also return the total amount of items to facilitate navigation to the last page
func (h *HttpStorer) List(w http.ResponseWriter, r *http.Request, collection string) {

	query := r.URL.Query()
	limit := 10 // Default limit
	page := 1   // Default page

	if l, err := strconv.Atoi(query.Get("limit")); err == nil && l > 0 {
		limit = l
	}
	if p, err := strconv.Atoi(query.Get("page")); err == nil && p > 0 {
		page = p
	}

	// Call the List method on the Storer
	items, total, err := h.Storer.List(r.Context(), collection, limit, page)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch items: %v", err), http.StatusInternalServerError)
		return
	}

	// Construct the response
	response := map[string]interface{}{
		"items": items,
		"total": total,
		"page":  page,
		"limit": limit,
	}

	// Respond with JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// Delete handles requests to delete an item in the collection, normally this would be a DELETE on /path/<key>
func (h *HttpStorer) Delete(w http.ResponseWriter, r *http.Request, collection, key string) {

	deleted, err := h.Storer.Delete(r.Context(), collection, key)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete data: %v", err), http.StatusInternalServerError)
		return
	}
	if !deleted {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)

}
