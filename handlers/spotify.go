package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
)

func GetSpotifyAccessToken(clientID, clientSecret string) (string, error) {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, _ := http.NewRequest("POST", "https://accounts.spotify.com/api/token", bytes.NewBufferString(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(clientID+":"+clientSecret)))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	var jsonResp map[string]interface{}
	json.Unmarshal(body, &jsonResp)

	return jsonResp["access_token"].(string), nil
}

func GetTrackTitleAndArtist(trackID, token string) (string, string, error) {
	req, _ := http.NewRequest("GET", "https://api.spotify.com/v1/tracks/"+trackID, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	var result struct {
		Name    string `json:"name"`
		Artists []struct {
			Name string `json:"name"`
		} `json:"artists"`
	}

	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Artists) == 0 {
		return "", "", fmt.Errorf("no artist found")
	}

	return result.Name, result.Artists[0].Name, nil
}

func GetPlaylistTitlesAndArtists(playlistID, token string) ([]string, error) {
	var results []string
	url := "https://api.spotify.com/v1/playlists/" + playlistID + "/tracks"

	for url != "" {
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		var page struct {
			Items []struct {
				Track struct {
					Name    string `json:"name"`
					Artists []struct {
						Name string `json:"name"`
					} `json:"artists"`
				} `json:"track"`
			} `json:"items"`
			Next string `json:"next"`
		}

		json.NewDecoder(resp.Body).Decode(&page)

		for _, item := range page.Items {
			if len(item.Track.Artists) > 0 {
				results = append(results, fmt.Sprintf("%s - %s", item.Track.Artists[0].Name, item.Track.Name))
			}
		}

		url = page.Next
	}

	return results, nil
}

func ParseSpotifyURL(spotifyURL string) (typ, id string, err error) {
	re := regexp.MustCompile(`open\.spotify\.com/(track|playlist)/([a-zA-Z0-9]+)`)
	match := re.FindStringSubmatch(spotifyURL)
	if len(match) != 3 {
		return "", "", fmt.Errorf("invalid Spotify URL")
	}
	return match[1], match[2], nil
}

func ContainsSpotify(s string) bool {
	re := regexp.MustCompile(`\bspotify\b`)
	return re.MatchString(s)
}
