package ignore

import (
	"encoding/json"
	"net/http"
	"strconv"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Document struct {
	ID   string `gorm:"primaryKey" json:"id"`
	Data string `json:"data"`
}

type jsonStorer interface {
}

type Handler struct {
	DB *gorm.DB
}

// NewHandler initializes the database and returns a new handler
func NewHandler() *Handler {
	db, err := gorm.Open(sqlite.Open("documents.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to database")
	}

	// Auto-migrate the Document model
	db.AutoMigrate(&Document{})

	return &Handler{DB: db}
}

// ServeHTTP is the main handler function
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && len(r.URL.Path) > 4:
		h.createOrUpdateDocument(w, r)
	case r.Method == http.MethodGet && len(r.URL.Path) > 4:
		h.getDocument(w, r)
	case r.Method == http.MethodGet:
		h.getAllDocuments(w, r)
	case r.Method == http.MethodDelete && len(r.URL.Path) > 4:
		h.deleteDocument(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// createOrUpdateDocument handles POST requests to create or update a document
func (h *Handler) createOrUpdateDocument(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[len("/db/"):]
	var doc Document
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	doc.ID = key

	// Create or update document
	if err := h.DB.Save(&doc).Error; err != nil {
		http.Error(w, "failed to save document", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(doc)
}

// getDocument handles GET requests to retrieve a specific document
func (h *Handler) getDocument(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[len("/db/"):]
	var doc Document

	// Find document by key
	if err := h.DB.First(&doc, "id = ?", key).Error; err != nil {
		http.Error(w, "document not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(doc)
}

// getAllDocuments handles GET requests to retrieve all documents with optional query params
func (h *Handler) getAllDocuments(w http.ResponseWriter, r *http.Request) {
	var docs []Document
	var count int64

	query := h.DB.Model(&Document{})

	// Handle query parameters
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			query = query.Limit(limit)
		}
	}
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil {
			query = query.Offset((page - 1) * 10)
		}
	}
	if order := r.URL.Query().Get("order"); order != "" {
		query = query.Order(order)
	}
	if search := r.URL.Query().Get("search"); search != "" {
		query = query.Where("data LIKE ?", "%"+search+"%")
	}

	// Execute query
	query.Find(&docs).Count(&count)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"documents": docs,
		"count":     count,
	})
}

// deleteDocument handles DELETE requests to delete a specific document
func (h *Handler) deleteDocument(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[len("/db/"):]
	if err := h.DB.Delete(&Document{}, "id = ?", key).Error; err != nil {
		http.Error(w, "failed to delete document", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
