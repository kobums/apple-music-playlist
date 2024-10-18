package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/kobums/playlist/controllers"
)

type SongController struct {
	controllers.Controller
}

type Song struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type AddTracksRequest struct {
	Data []Song `json:"data"`
}

func (c *SongController) searchSong(jwtToken, userToken, term string) (songID string, err error) {
	defer func() {
		if r := recover(); r != nil {
			songID = ""                                                         // songID를 빈 문자열로 설정
			err = fmt.Errorf("panic occurred during search for term: %s", term) // 에러 메시지 설정
		}
	}()

	url := fmt.Sprintf("https://api.music.apple.com/v1/catalog/us/search?term=%s&types=songs", url.QueryEscape(term))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Music-User-Token", userToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	songID, ok := result["results"].(map[string]interface{})["songs"].(map[string]interface{})["data"].([]interface{})[0].(map[string]interface{})["id"].(string)
	if !ok {
		return "", fmt.Errorf("failed to extract song ID")
	}

	return songID, nil
}

func (c *SongController) addTrackToPlaylist(jwtToken, userToken, playlistID, songID string) error {
	url := fmt.Sprintf("https://api.music.apple.com/v1/me/library/playlists/%s/tracks", playlistID)

	tracks := AddTracksRequest{
		Data: []Song{
			{ID: songID, Type: "songs"},
		},
	}

	requestBody, err := json.Marshal(tracks)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Music-User-Token", userToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to add track: %s", resp.Status)
	}

	return nil
}
