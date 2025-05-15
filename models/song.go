package models

import "time"

type SongInfo struct {
	FilePath string

	Title    string
	Uploader string
	ID       string
	Duration time.Duration
}
