package fs

import (
	"context"
	"errors"
	"fmt"
	"gobot/models"
	"io"
	"log"
	"os"

	"github.com/serbirial/goutubedl"
)

const AUDIO_FOLDER string = "audio_temp"

func RemoveDownloadedSong(song models.SongInfo) {

}

func DownloadYoutubeURLToFile(url string, folder string) (*models.SongInfo, error) {
	goutubeOptions := new(goutubedl.Options)
	goutubeOptions.DownloadThumbnail = false
	goutubeOptions.DownloadSubtitles = false
	goutubeOptions.Downloader = "aria2c"
	fmt.Println("[yt-dlp] Downloading metadata")
	result, err := goutubedl.New(context.Background(), url, *goutubeOptions)
	if err != nil {
		log.Fatal(err)
	}
	// check if we havent downloaded it
	if _, err := os.Stat(fmt.Sprintf("%s/%s", AUDIO_FOLDER, result.Info.ID)); errors.Is(err, os.ErrNotExist) {
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

		io.Copy(f, downloadResult)
		fmt.Println("[yt-dlp] Downloaded")

	}

	filePath := fmt.Sprintf("%s/%s", AUDIO_FOLDER, result.Info.ID)
	// Convert the opus to discord accepted DCA and get the new path
	filePath = convertToDCA(filePath)

	songInfo := models.SongInfo{
		FilePath: filePath,

		Title:    result.Info.Title,
		Uploader: result.Info.Uploader,
		ID:       result.Info.ID,
	}

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
