package fs

import (
	"ascension/error"
	"os"
	"strings"
)

func ReadFileWhole(f string) string {
	data, err := os.ReadFile(f)
	error.ErrorCheckPanic(err)
	dataString := string(data)
	return strings.TrimSuffix(dataString, "\n")
}
