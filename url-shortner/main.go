package main

import (
	"log"
	"net/http"
	"url-shortner/handlers"
	"url-shortner/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	store, err := storage.NewStore("urls.db")
	if err != nil {
		log.Fatal(err)
	}
	handler := handlers.NewUrlHandler(store)
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Post("/shorten", handler.Shortern)
	r.Get("/{code}", handler.Redirect)
	r.Get("/stats/{code}", handler.Stats)
	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
