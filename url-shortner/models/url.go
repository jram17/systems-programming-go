package models

import "time"

type URL struct {
	ID        int64     `json:"id"`
	Code      string    `json:"code"`
	URL       string    `json:"url"`
	Clicks    int       `json:"clicks"`
	CreatedAt time.Time `json:"created_at"`
}

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	ShortUrl string `json:"short_url"`
	Code     string `json:"code"`
}

type StatsResponse struct {
	Code      string    `json:"code"`
	URL       string    `json:"url"`
	Clicks    int       `json:"clicks"`
	CreatedAt time.Time `json:"created_at"`
}
