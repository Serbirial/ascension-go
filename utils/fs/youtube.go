package fs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"ascension/models"
	"io"
	"log"
	"os"

	"github.com/kkdai/youtube/v2"
	//"github.com/serbirial/goutubedl"
)

const AUDIO_FOLDER string = "audio_temp"
const FILE_ENDING string = "dca"

func DownloadDetached(DownloaderUrl, url string) (*models.SongInfo, error) {
	type DownloadRequest struct {
		URL string `json:"url"`
	}

	reqBody := DownloadRequest{URL: url}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Println("[DETACHED-DOWNLOADER] Failed to marshal request JSON:", err)
		return nil, err
	}

	resp, err := http.Post(DownloaderUrl+"/download", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("[DETACHED-DOWNLOADER] Failed to POST to detached downloader server:", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[DETACHED-DOWNLOADER] Download server responded with status: %d\n", resp.StatusCode)
		return nil, errors.New("download server returned non-OK status")
	}

	var song models.SongInfo
	if err := json.NewDecoder(resp.Body).Decode(&song); err != nil {
		log.Println("[DETACHED-DOWNLOADER] Failed to decode SongInfo from response:", err)
		return nil, err
	}

	log.Printf("[DETACHED-DOWNLOADER] Successfully downloaded and received info: %s\n", song.Title)
	return &song, nil
}

func RemoveDownloadedSong(song models.SongInfo) {

}

func saveSongInfoToFile(songInfo models.SongInfo, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty-print the JSON
	if err := encoder.Encode(songInfo); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return nil
}

func loadSongInfoFromFile(filename string) (*models.SongInfo, error) {
	var songInfo models.SongInfo

	file, err := os.Open(filename)
	if err != nil {
		return &songInfo, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&songInfo); err != nil {
		return &songInfo, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return &songInfo, nil
}

func ExtractYouTubeVideoID(parsedURL *url.URL) (string, error) {
	host := parsedURL.Hostname()

	switch {
	case strings.Contains(host, "youtube.com"):
		// Example: https://www.youtube.com/watch?v=VIDEOID
		videoID := parsedURL.Query().Get("v")
		if videoID == "" {
			return "", fmt.Errorf("no video ID found in youtube.com URL")
		}
		return videoID, nil

	case strings.Contains(host, "youtu.be"):
		// Example: https://youtu.be/VIDEOID
		videoID := strings.Trim(parsedURL.Path, "/")
		if videoID == "" {
			return "", fmt.Errorf("no video ID found in youtu.be URL")
		}
		return videoID, nil

	default:
		return "", fmt.Errorf("unsupported YouTube domain: %s", host)
	}
}

func DownloadYoutubeURLToFile(rawurl string, folder string) (*models.SongInfo, error) {
	parsedURL, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	videoID, err := ExtractYouTubeVideoID(parsedURL)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("%s/%s.json", AUDIO_FOLDER, videoID)
	log.Println("Looking for metadata at " + path)

	if _, err := os.Stat(path); err == nil {
		if _, err := os.Stat(fmt.Sprintf("%s/%s.%s", AUDIO_FOLDER, videoID, FILE_ENDING)); err == nil {
			return loadSongInfoFromFile(path)
		}
	}

	log.Println("[yt-dlp] Downloading metadata")
	client := youtube.Client{}

	video, err := client.GetVideo(videoID)
	//result, err := goutubedl.New(context.Background(), rawurl, *goutubeOptions)
	if err != nil {
		log.Fatal(err)
	}
	filePath := fmt.Sprintf("%s/%s.%s", AUDIO_FOLDER, videoID, FILE_ENDING)
	// check if we havent downloaded and converted it
	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		log.Println("[yt-dlp] Downloading video")
		filePath := fmt.Sprintf("%s/%s", AUDIO_FOLDER, videoID)
		client := youtube.Client{}
		formats := video.Formats.WithAudioChannels() // only get videos with audio
		stream, _, err := client.GetStream(video, &formats[0])
		if err != nil {
			log.Fatal(err)
		}
		defer stream.Close()

		f, err := os.Create(filePath)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		songInfo := models.SongInfo{
			FilePath: filePath + "." + FILE_ENDING,

			Title:    video.Title,
			Uploader: video.Author,
			ID:       videoID,
			Duration: video.Duration,
		}
		saveSongInfoToFile(songInfo, fmt.Sprintf("%s/%s.%s", AUDIO_FOLDER, videoID, "json"))
		io.Copy(f, stream)
		log.Println("[yt-dlp] Downloaded")

		// Convert the opus to discord accepted DCA and get the new path
		_, err = convertToDCA(filePath)
		if err != nil {
			os.Remove(path) // Remove JSON metadata
			return nil, err
		}

	}

	songInfo := models.SongInfo{
		FilePath: filePath,

		Title:    video.Title,
		Uploader: video.Author,
		ID:       videoID,
	}
	saveSongInfoToFile(songInfo, fmt.Sprintf("%s/%s.%s", AUDIO_FOLDER, videoID, "json"))

	fmt.Println(filePath)
	if err != nil {
		log.Fatal(err)
	}

	return &songInfo, nil
}
