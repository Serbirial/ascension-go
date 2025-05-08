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
	downloadOptions := new(goutubedl.DownloadOptions)
	downloadOptions.DownloadAudioOnly = true
	downloadOptions.AudioFormats = "best"
	goutubeOptions := new(goutubedl.Options)
	goutubeOptions.DownloadThumbnail = false
	goutubeOptions.DownloadSubtitles = false
	if _, err := os.Stat("history.txt"); errors.Is(err, os.ErrNotExist) {
		result, err := goutubedl.New(context.Background(), url, *goutubeOptions)
		if err != nil {
			log.Fatal(err)
		}
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
		cwd, err := os.Getwd()
		if err != nil {
			panic("Cant get CWD")
		}
		lf, err := os.OpenFile(cwd+"/history.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			panic(err)
		}
		defer lf.Close()

		// Log the ID so we dont have to download it again
		if _, err = lf.WriteString(songInfo.ID); err != nil {
			panic(err)
		}
		io.Copy(f, downloadResult)
		return &songInfo, nil

	} else {
		result, err := goutubedl.New(context.Background(), url, *goutubeOptions)
		if err != nil {
			log.Fatal(err)
		}
		filePath := fmt.Sprintf("%s/%s", AUDIO_FOLDER, result.Info.ID)
		songInfo := models.SongInfo{
			FilePath: filePath,

			Title:    result.Info.Title,
			Uploader: result.Info.Uploader,
			ID:       result.Info.ID,
		}
		return &songInfo, nil
	}
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
