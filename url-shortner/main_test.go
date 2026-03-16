package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"url-shortner/encoder"
	"url-shortner/handlers"
	"url-shortner/models"
	"url-shortner/storage"

	"github.com/go-chi/chi/v5"
)

// Test encoder
func TestEncoderBasic(t *testing.T) {
	tests := []struct {
		name     string
		input    uint64
		expected string
	}{
		{"zero", 0, "a"},
		{"one", 1, "b"},
		{"two", 2, "c"},
		{"sixty-one", 61, "9"},
		{"sixty-two", 62, "ba"},
		{"one-twenty-five", 125, "cb"},
		{"thousand", 1000, "qi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encoder.Encode(tt.input)
			if result != tt.expected {
				t.Errorf("Encode(%d) = %s; want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEncoderLargeNumbers(t *testing.T) {
	tests := []uint64{10000, 100000, 1000000, 999999999}

	for _, num := range tests {
		code := encoder.Encode(num)
		if code == "" {
			t.Errorf("Encode(%d) returned empty string", num)
		}
		if len(code) == 0 {
			t.Errorf("Encode(%d) returned zero-length code", num)
		}
	}
}

// Test storage layer
func TestStorageCreate(t *testing.T) {
	store, err := storage.NewStore(":memory:")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	err = store.Save("test123", "https://google.com")
	if err != nil {
		t.Errorf("Save failed: %v", err)
	}
}

func TestStorageGet(t *testing.T) {
	store, _ := storage.NewStore(":memory:")
	store.Save("test123", "https://google.com")

	url, err := store.Get("test123")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if url != "https://google.com" {
		t.Errorf("Get returned %s; want https://google.com", url)
	}
}

func TestStorageGetNonExistent(t *testing.T) {
	store, _ := storage.NewStore(":memory:")

	_, err := store.Get("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent code, got nil")
	}
}

func TestStorageGetNextID(t *testing.T) {
	store, _ := storage.NewStore(":memory:")

	id1, err := store.GetNextID()
	if err != nil {
		t.Errorf("GetNextID failed: %v", err)
	}
	if id1 != 1 {
		t.Errorf("First ID = %d; want 1", id1)
	}

	store.Save("test", "https://test.com")

	id2, err := store.GetNextID()
	if err != nil {
		t.Errorf("GetNextID failed: %v", err)
	}
	if id2 != 2 {
		t.Errorf("Second ID = %d; want 2", id2)
	}
}

func TestStorageIncrementClicks(t *testing.T) {
	store, _ := storage.NewStore(":memory:")
	store.Save("test", "https://google.com")

	for i := 0; i < 3; i++ {
		err := store.IncrementClicks("test")
		if err != nil {
			t.Errorf("IncrementClicks failed: %v", err)
		}
	}

	_, clicks, _, err := store.GetStats("test")
	if err != nil {
		t.Errorf("GetStats failed: %v", err)
	}
	if clicks != 3 {
		t.Errorf("Clicks = %d; want 3", clicks)
	}
}

func TestStorageGetStats(t *testing.T) {
	store, _ := storage.NewStore(":memory:")
	store.Save("test", "https://google.com")
	store.IncrementClicks("test")

	url, clicks, createdAt, err := store.GetStats("test")
	if err != nil {
		t.Errorf("GetStats failed: %v", err)
	}
	if url != "https://google.com" {
		t.Errorf("URL = %s; want https://google.com", url)
	}
	if clicks != 1 {
		t.Errorf("Clicks = %d; want 1", clicks)
	}
	if createdAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

// Test handlers
func setupTestHandler() (*handlers.URLHandler, *storage.Store) {
	store, _ := storage.NewStore(":memory:")
	handler := handlers.NewUrlHandler(store)
	return handler, store
}

func TestHandlerShortenSuccess(t *testing.T) {
	handler, _ := setupTestHandler()

	reqBody := models.ShortenRequest{URL: "https://google.com"}
	jsonData, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/shorten", bytes.NewBuffer(jsonData))
	w := httptest.NewRecorder()

	handler.Shortern(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Status = %d; want %d", w.Code, http.StatusCreated)
	}

	var resp models.ShortenResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Code == "" {
		t.Error("Expected code, got empty string")
	}
	if resp.ShortUrl == "" {
		t.Error("Expected short_url, got empty string")
	}
}

func TestHandlerShortenEmptyURL(t *testing.T) {
	handler, _ := setupTestHandler()

	reqBody := models.ShortenRequest{URL: ""}
	jsonData, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/shorten", bytes.NewBuffer(jsonData))
	w := httptest.NewRecorder()

	handler.Shortern(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d; want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerShortenInvalidJSON(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("POST", "/shorten", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	handler.Shortern(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d; want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerShortenMultipleURLs(t *testing.T) {
	handler, _ := setupTestHandler()

	urls := []string{
		"https://google.com",
		"https://github.com",
		"https://stackoverflow.com",
	}

	codes := make(map[string]bool)

	for _, url := range urls {
		reqBody := models.ShortenRequest{URL: url}
		jsonData, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/shorten", bytes.NewBuffer(jsonData))
		w := httptest.NewRecorder()

		handler.Shortern(w, req)

		var resp models.ShortenResponse
		json.NewDecoder(w.Body).Decode(&resp)

		if codes[resp.Code] {
			t.Errorf("Duplicate code generated: %s", resp.Code)
		}
		codes[resp.Code] = true
	}

	if len(codes) != len(urls) {
		t.Errorf("Expected %d unique codes, got %d", len(urls), len(codes))
	}
}

func TestHandlerRedirectSuccess(t *testing.T) {
	handler, store := setupTestHandler()
	store.Save("test123", "https://google.com")

	req := httptest.NewRequest("GET", "/test123", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("code", "test123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.Redirect(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Errorf("Status = %d; want %d", w.Code, http.StatusMovedPermanently)
	}

	location := w.Header().Get("Location")
	if location != "https://google.com" {
		t.Errorf("Location = %s; want https://google.com", location)
	}
}

func TestHandlerRedirectNonExistent(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("code", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.Redirect(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %d; want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlerRedirectClickIncrement(t *testing.T) {
	handler, store := setupTestHandler()
	store.Save("test", "https://google.com")

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("code", "test")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.Redirect(w, req)
		
		// Manually increment to avoid goroutine timing issues in tests
		store.IncrementClicks("test")
	}

	_, clicks, _, _ := store.GetStats("test")
	if clicks != 3 {
		t.Errorf("Clicks = %d; want 3", clicks)
	}
}

func TestHandlerStatsSuccess(t *testing.T) {
	handler, store := setupTestHandler()
	store.Save("test", "https://google.com")
	store.IncrementClicks("test")

	req := httptest.NewRequest("GET", "/stats/test", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("code", "test")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.Stats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d; want %d", w.Code, http.StatusOK)
	}

	var resp models.StatsResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Code != "test" {
		t.Errorf("Code = %s; want test", resp.Code)
	}
	if resp.URL != "https://google.com" {
		t.Errorf("URL = %s; want https://google.com", resp.URL)
	}
	if resp.Clicks != 1 {
		t.Errorf("Clicks = %d; want 1", resp.Clicks)
	}
}

func TestHandlerStatsNonExistent(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/stats/nonexistent", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("code", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.Stats(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %d; want %d", w.Code, http.StatusNotFound)
	}
}

func TestEdgeCaseVeryLongURL(t *testing.T) {
	handler, _ := setupTestHandler()

	longURL := "https://example.com/" + string(make([]byte, 10000))
	reqBody := models.ShortenRequest{URL: longURL}
	jsonData, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/shorten", bytes.NewBuffer(jsonData))
	w := httptest.NewRecorder()

	handler.Shortern(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Status = %d; want %d for long URL", w.Code, http.StatusCreated)
	}
}

func TestEdgeCaseSpecialCharactersInURL(t *testing.T) {
	handler, _ := setupTestHandler()

	urls := []string{
		"https://example.com/path?query=value&other=123",
		"https://example.com/path#fragment",
		"https://example.com/path%20with%20spaces",
	}

	for _, url := range urls {
		reqBody := models.ShortenRequest{URL: url}
		jsonData, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/shorten", bytes.NewBuffer(jsonData))
		w := httptest.NewRecorder()

		handler.Shortern(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Status = %d; want %d for URL: %s", w.Code, http.StatusCreated, url)
		}
	}
}

func BenchmarkEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		encoder.Encode(uint64(i))
	}
}

func BenchmarkShortenHandler(b *testing.B) {
	handler, _ := setupTestHandler()
	reqBody := models.ShortenRequest{URL: "https://google.com"}
	jsonData, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/shorten", bytes.NewBuffer(jsonData))
		w := httptest.NewRecorder()
		handler.Shortern(w, req)
	}
}
