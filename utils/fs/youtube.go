package fs

import (
	"context"
	"fmt"
	"gobot/models"
	"io"
	"log"
	"os"

	"github.com/wader/goutubedl"
)

const AUDIO_FOLDER string = "audio_temp"

func DownloadYoutubeURLToFile(url string, folder string) (*models.SongInfo, error) {
	result, err := goutubedl.New(context.Background(), "https://www.youtube.com/watch?v=jgVhBThJdXc", goutubedl.Options{})
	if err != nil {
		log.Fatal(err)
	}
	downloadResult, err := result.Download(context.Background(), "best")
	if err != nil {
		log.Fatal(err)
	}
	defer downloadResult.Close()

	filePath := fmt.Sprintf("%s/%s", AUDIO_FOLDER, result.Info.Title)
	songInfo := models.SongInfo{
		FilePath: filePath,

		Title:    result.Info.Title,
		Uploader: result.Info.Uploader,
	}

	f, err := os.Create(filePath)
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
