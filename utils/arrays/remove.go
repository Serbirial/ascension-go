package arrays

import "gobot/models"

func Remove(s []any, r any) []any {
	for i, v := range s {
		if v == r {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}

func RemoveFirstSong(s []*models.SongInfo) []*models.SongInfo {
	var temp []*models.SongInfo

	for i := 0; i < len(s); i++ {
		if i >= 1 {
			temp = append(temp, s[i])
		}
	}
	return temp
}
