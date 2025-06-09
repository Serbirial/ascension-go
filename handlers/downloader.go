package handlers

import (
	"ascension/utils/fs"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
)

type DownloadRequest struct {
	URL string `json:"url"`
}

func HandleDownloader(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Use POST", http.StatusMethodNotAllowed)
		return
	}

	var req DownloadRequest
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("[DL] Received download request for %s", req.URL)
	songInfo, err := fs.DownloadYoutubeURLToFile(req.URL, fs.AUDIO_FOLDER)
	if err != nil {
		http.Error(w, "Download failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Marshal the SongInfo into JSON and return it
	respJSON, err := json.Marshal(songInfo)
	if err != nil {
		http.Error(w, "Failed to marshal SongInfo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respJSON)
}

func HandleDownloaderSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Use POST", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Query string `json:"query"` // Rename from "url" to "query"
	}
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err := json.Unmarshal(body, &req); err != nil || strings.TrimSpace(req.Query) == "" {
		http.Error(w, "Invalid or missing query", http.StatusBadRequest)
		return
	}

	log.Printf("[DL] Received search request for query: %s", req.Query)

	videoID, err := SearchYouTube(req.Query)
	if err != nil {
		http.Error(w, "Search failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with {"id": "..."} to match client code
	resp := map[string]string{"id": videoID}
	respJSON, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respJSON)
}

func HandleDownloaderGetRelated(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Use POST", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID    string `json:"id"`
		Limit int    `json:"limit"`
	}

	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err := json.Unmarshal(body, &req); err != nil || strings.TrimSpace(req.ID) == "" {
		http.Error(w, "Invalid JSON or missing 'id'", http.StatusBadRequest)
		return
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 5
	}

	log.Printf("[DL] Getting related videos for %s (limit %d)", req.ID, limit)

	related, err := GetRelatedSong(req.ID, limit)
	if err != nil {
		http.Error(w, "Failed to get related songs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	respJSON, _ := json.Marshal(related)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respJSON)
}
