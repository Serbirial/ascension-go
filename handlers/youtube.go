package handlers

import (
	"bufio"
	"os/exec"
	"strings"
)

func GetRelatedSong(videoID string, limit int) ([]string, error) {
	videoURL := "https://www.youtube.com/watch?v=" + videoID
	cmd := exec.Command("yt-dlp", videoURL, "--print", "%(related_urls)s")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var related []string
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		links := strings.Split(line, " ")
		for _, link := range links {
			if strings.Contains(link, "youtube.com/watch?v=") {
				related = append(related, link)
				if limit > 0 && len(related) >= limit {
					break
				}
			}
		}
		if limit > 0 && len(related) >= limit {
			break
		}
	}

	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	return related, nil
}

// Searches youtube for 'query' and returns a video ID. Should only be used in detached downloader.
func SearchYouTube(query string) (string, error) {
	cmd := exec.Command("yt-dlp", "ytsearch1:"+query, "--print", "id", "--skip-download")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
