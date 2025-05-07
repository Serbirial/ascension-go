package fs

import (
	"fmt"
	"gobot/models"
	"io"
	"os"

	"github.com/kkdai/youtube/v2"
)

const AUDIO_FOLDER string = "audio_temp"

func DownloadYoutubeURLToFile(url string, folder string) (*models.SongInfo, error) {

	client := youtube.Client{}

	video, err := client.GetVideo(url)
	if err != nil {
		panic(err)
	}

	filePath := fmt.Sprintf("%s/%s", AUDIO_FOLDER, video.Title)

	songInfo := models.SongInfo{
		FilePath: filePath,

		Title:    video.Title,
		Uploader: video.Author,
	}

	formats := video.Formats.WithAudioChannels() // only get videos with audio
	stream, _, err := client.GetStream(video, &formats[0])
	if err != nil {
		panic(err)
	}
	defer stream.Close()

	file, err := os.Create(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	_, err = io.Copy(file, stream)
	if err != nil {
		panic(err)
	}

	return &songInfo, nil
}
