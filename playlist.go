package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
)

type PlaylistAttributes struct {
    Name        string `json:"name"`
    Description string `json:"description"`
}

type PlaylistData struct {
    Attributes PlaylistAttributes `json:"attributes"`
}

type CreatePlaylistRequest struct {
    Data PlaylistData `json:"data"`
}

func createPlaylist(jwtToken, playlistName string) (string, error) {
    userToken := os.Getenv("USER_TOKEN")
    url := "https://api.music.apple.com/v1/me/library/playlists"

    playlist := CreatePlaylistRequest{
        Data: PlaylistData{
            Attributes: PlaylistAttributes{
                Name:        playlistName,
                Description: "Generated Playlist",
            },
        },
    }

    requestBody, err := json.Marshal(playlist)
    if err != nil {
        return "", err
    }

    req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
    if err != nil {
        return "", err
    }

    req.Header.Set("Authorization", "Bearer "+jwtToken)
    req.Header.Set("Music-User-Token", userToken)
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    var result map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", err
    }

    data := result["data"].([]interface{})[0].(map[string]interface{})
    playlistID := data["id"].(string)

    return playlistID, nil
}
