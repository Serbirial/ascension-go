package fs

import (
	"context"
	"fmt"
	"gobot/models"
	"io"
	"log"
	"os"

	"github.com/serbirial/goutubedl"
)

const AUDIO_FOLDER string = "audio_temp"

func DownloadYoutubeURLToFile(url string, folder string) (*models.SongInfo, error) {
	goutubeOptions := new(goutubedl.Options)
	goutubeOptions.DownloadThumbnail = false
	result, err := goutubedl.New(context.Background(), url, *goutubeOptions)
	if err != nil {
		log.Fatal(err)
	}
	downloadOptions := new(goutubedl.DownloadOptions)
	downloadOptions.DownloadAudioOnly = true
	downloadOptions.AudioFormats = "mp3"
	downloadResult, err := result.DownloadWithOptions(context.Background(), *downloadOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer downloadResult.Close()

	filePath := fmt.Sprintf("%s/%s", AUDIO_FOLDER, result.Info.ID)
	songInfo := models.SongInfo{
		FilePath: filePath,

		Title:    result.Info.Title,
		Uploader: result.Info.Uploader,
		ID:       result.Info.ID,
	}

	f, err := os.Create(filePath)
	fmt.Println(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	io.Copy(f, downloadResult)

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
