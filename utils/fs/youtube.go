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
	result, err := goutubedl.New(context.Background(), url, *goutubeOptions)
	if err != nil {
		log.Fatal(err)
	}
	// check if we havent downloaded it
	if _, err := os.Stat(fmt.Sprintf("%s/%s", AUDIO_FOLDER, result.Info.ID)); errors.Is(err, os.ErrNotExist) {
		filePath := fmt.Sprintf("%s/%s", AUDIO_FOLDER, result.Info.ID)

		downloadOptions := new(goutubedl.DownloadOptions)
		downloadOptions.DownloadAudioOnly = true
		downloadOptions.AudioFormats = "best"
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

	}

	filePath := fmt.Sprintf("%s/%s", AUDIO_FOLDER, result.Info.ID)
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
