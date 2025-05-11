package fs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gobot/models"
	"io"
	"log"
	"os"
	"strings"

	"github.com/serbirial/goutubedl"
)

const AUDIO_FOLDER string = "audio_temp"
const FILE_ENDING string = "dca"

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

func DownloadYoutubeURLToFile(url string, folder string) (*models.SongInfo, error) {
	parts := strings.Split(url, "?v=")
	if len(parts) > 1 {
		path := fmt.Sprintf("%s/%s.%s", AUDIO_FOLDER, parts[1], ".json")
		_, err := os.Stat(path)
		if err == nil {
			return loadSongInfoFromFile(path)
		}

	} else {
		fmt.Println("No '?v=' found in URL")
	}

	goutubeOptions := new(goutubedl.Options)
	goutubeOptions.DownloadThumbnail = false
	goutubeOptions.DownloadSubtitles = false
	goutubeOptions.Downloader = "aria2c"
	fmt.Println("[yt-dlp] Downloading metadata")
	result, err := goutubedl.New(context.Background(), url, *goutubeOptions)
	if err != nil {
		log.Fatal(err)
	}
	filePath := fmt.Sprintf("%s/%s.%s", AUDIO_FOLDER, result.Info.ID, FILE_ENDING)
	// check if we havent downloaded and converted it
	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		fmt.Println("[yt-dlp] Downloading video")
		filePath := fmt.Sprintf("%s/%s", AUDIO_FOLDER, result.Info.ID)

		downloadOptions := new(goutubedl.DownloadOptions)
		downloadOptions.DownloadAudioOnly = true
		downloadOptions.AudioFormats = "opus"
		downloadOptions.Filter = "ba"
		downloadOptions.PlaylistIndex = 1
		downloadResult, err := result.DownloadWithOptions(context.Background(), *downloadOptions)
		if err != nil {
			log.Fatal(err)
		}
		defer downloadResult.Close()
		f, err := os.Create(filePath)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		songInfo := models.SongInfo{
			FilePath: filePath + "." + FILE_ENDING,

			Title:    result.Info.Title,
			Uploader: result.Info.Uploader,
			ID:       result.Info.ID,
		}
		saveSongInfoToFile(songInfo, fmt.Sprintf("%s/%s.%s", AUDIO_FOLDER, result.Info.ID, "json"))
		io.Copy(f, downloadResult)
		fmt.Println("[yt-dlp] Downloaded")

		// Convert the opus to discord accepted DCA and get the new path
		_ = convertToDCA(filePath)

	}

	songInfo := models.SongInfo{
		FilePath: filePath,

		Title:    result.Info.Title,
		Uploader: result.Info.Uploader,
		ID:       result.Info.ID,
	}
	saveSongInfoToFile(songInfo, fmt.Sprintf("%s/%s.%s", AUDIO_FOLDER, result.Info.ID, "json"))

	fmt.Println(filePath)
	if err != nil {
		log.Fatal(err)
	}

	return &songInfo, nil
	//	client := youtube.Client{}
	//
	//	video, err := client.GetVideo(url)
	//	if err != nil {
	//		panic(err)
	//	}

	//	filePath := fmt.Sprintf("%s/%s", AUDIO_FOLDER, video.Title)

	//	songInfo := models.SongInfo{
	//		FilePath: filePath,
	//
	//		Title:    video.Title,
	//		Uploader: video.Author,
	//	}
	//
	//	formats := video.Formats.WithAudioChannels() // only get videos with audio
	//	stream, _, err := client.GetStream(video, &formats[0])
	//	if err != nil {
	//		panic(err)
	//	}
	//	defer stream.Close()
	//
	//	file, err := os.Create(filePath)
	//	if err != nil {
	//		panic(err)
	//	}
	//	defer file.Close()
	//
	//	_, err = io.Copy(file, stream)
	//	if err != nil {
	//		panic(err)
	//	}
	//
	//	return &songInfo, nil
}
