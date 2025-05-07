package models

import "github.com/wader/goutubedl"

type SongInfo struct {
	FilePath string
	MetaData *goutubedl.Info
}
