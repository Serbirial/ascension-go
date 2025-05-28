package handlers

import (
	"ascension/utils/fs"
	"encoding/json"
	"io"
	"log"
	"net/http"
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
