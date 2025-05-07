package fs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/wader/goutubedl"
)

func DownloadYoutubeURLToFile(url string, folder string) (string, error) {
	result, err := goutubedl.New(context.Background(), url, goutubedl.Options{})
	if err != nil {
		return "", errors.New("Error while initializing goutube")
	}
	downloadResult, err := result.Download(context.Background(), "best")
	if err != nil {
		return "", errors.New("Error while downloading url")
	}
	defer downloadResult.Close()
	var filename string = fmt.Sprintf("%s/%s", folder, result.Info.Title)
	f, err := os.Create(filename)
	if err != nil {
		return "", errors.New("Error while creating output")
	}

	defer f.Close()
	io.Copy(f, downloadResult)
	return filename, nil
}
