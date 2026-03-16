package handlers

import (
	"encoding/json"
	"net/http"
	"url-shortner/encoder"
	"url-shortner/models"
	"url-shortner/storage"

	"github.com/go-chi/chi/v5"
)

type URLHandler struct {
	store storage.Store
}

func NewUrlHandler(store *storage.Store) *URLHandler {
	return &URLHandler{store: *store}
}

func (h *URLHandler) Shortern(w http.ResponseWriter, r *http.Request) {
	var req models.ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}
	id, err := h.store.GetNextID()
	if err != nil {
		http.Error(w, "Database error!!", http.StatusInternalServerError)
		return
	}
	code := encoder.Encode(uint64(id))
	if err := h.store.Save(code, req.URL); err != nil {
		http.Error(w, "Failed to save url", http.StatusInternalServerError)
		return
	}
	res := models.ShortenResponse{
		ShortUrl: "http://localhost:8080/" + code,
		Code:     code,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)

}

func (h *URLHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	url, err := h.store.Get(code)
	if err != nil {
		http.Error(w, "url not found ", http.StatusNotFound)
		return
	}
	go h.store.IncrementClicks(code)
	http.Redirect(w, r, url, http.StatusMovedPermanently)
}

func (h *URLHandler) Stats(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	url, clicks, createdAt, err := h.store.GetStats(code)
	if err != nil {
		http.Error(w, "url not found", http.StatusNotFound)
		return
	}
	res := models.StatsResponse{
		Code:      code,
		URL:       url,
		Clicks:    clicks,
		CreatedAt: createdAt,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}
