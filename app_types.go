package main

// === СТРУКТУРЫ ===

type ChapterResponse struct {
	Path            string   `json:"path"`
	Title           string   `json:"title"`
	Images          []string `json:"images"`
	ImageCount      int      `json:"imageCount"`
	DetectedTitleID *uint    `json:"detected_title_id"`
}

type CreatePageResponse struct {
	Success   bool   `json:"success"`
	Url       string `json:"url"`
	HistoryID uint   `json:"history_id"`
	Error     string `json:"error"`
}

type TelegramChannel struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	AccessHash string `json:"access_hash"`
}

type FrontendSettings struct {
	Resize           bool   `json:"resize"`
	ResizeTo         int    `json:"resize_to"`
	WebpQuality      int    `json:"webp_quality"`
	LastChannelID    string `json:"last_channel_id"`
	LastChannelHash  string `json:"last_channel_hash"`
	LastChannelTitle string `json:"last_channel_title"`
}
