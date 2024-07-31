package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func parseFile(filename string) (string, []string, error) {
    file, err := os.Open(filename)
    if err != nil {
        return "", nil, err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    var playlistName string
    var songs []string

    for scanner.Scan() {
        line := scanner.Text()
        if strings.HasPrefix(line, "playlistname:") {
            playlistName = strings.TrimSpace(strings.TrimPrefix(line, "playlistname:"))
        } else {
            song := regexp.MustCompile(`\s*\d+:\d+`).ReplaceAllString(line, "")
            songs = append(songs, strings.TrimSpace(song))
        }
    }

    if err := scanner.Err(); err != nil {
        return "", nil, err
    }

    return playlistName, songs, nil
}

func main() {

    loadEnv()

    jwtToken, err := generateToken()
    if err != nil {
        fmt.Println("Error generating token:", err)
        return
    }
    // fmt.Println(jwtToken)

    filename := "playlist.txt" // 입력 파일 이름
    playlistName, songs, err := parseFile(filename)
    // fmt.Print(playlistName)

    if err != nil {
        fmt.Println("Error reading file:", err)
        return
    }

    playlists, err := listPlaylists(jwtToken)
    if err != nil {
        fmt.Println("Error retrieving playlists:", err)
        return
    }

    var playlistID string

    // 기존 플레이리스트 검색
    found := false
    for _, playlist := range playlists {
        fmt.Println(playlist["attributes"].(map[string]interface{})["name"].(string))
        if playlistName == playlist["attributes"].(map[string]interface{})["name"].(string) {
            if id, ok := playlist["id"].(string); ok {
                playlistID = id
                found = true
                break
            } else {
                fmt.Println("Failed to type assert playlist ID")
            }
        }
    }

    // 플레이리스트가 없으면 새로 생성
    if !found {
        playlistID, err = createPlaylist(jwtToken, playlistName)
        if err != nil {
            fmt.Println("Error creating playlist:", err)
            return
        }
    }


    if err != nil {
        fmt.Println("Error creating playlist:", err)
        return
    }
    for _, song := range songs {
        songDetails := strings.Split(song, " - ")
        if len(songDetails) != 2 {
            fmt.Println("Invalid song format:", song)
            continue
        }

        searchTerm := fmt.Sprintf("%s %s", songDetails[0], songDetails[1])
        songID, err := searchSong(jwtToken, searchTerm)

        if err != nil {
            fmt.Println("Error searching song:", err, songDetails[0], songDetails[1])
            continue
        }

        err = addTrackToPlaylist(jwtToken, playlistID, songID)
        if err != nil {
            fmt.Printf("Not Added song: %s - %s\n", songDetails[0], songDetails[1])
            // fmt.Println("Error adding track to playlist:", err)
            continue
        }

        // fmt.Printf("Added song: %s %s to playlist\n", songDetails[0], songDetails[1])
    }
}
