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
    // jwtToken, err := generateToken()
    // if err != nil {
    //     fmt.Println("Error generating token:", err)
    //     return
    // }
    // fmt.Println(jwtToken)

    jwtToken := "eyJhbGciOiJFUzI1NiIsImtpZCI6IjY5TTRYQkRENlIiLCJ0eXAiOiJKV1QifQ.eyJleHAiOjE3MTY2NTIzOTMsImlhdCI6MTcxNjY0ODc5MywiaXNzIjoiRzZCODJCSFZBWCJ9._KPfuOG5rCr_rM52AKWG8SdTmE8Dw14XL5eIrJrs4Nc7rWxIaAUexLERlbdHiz1rNAErL_RruWGe4CDeizHxuQ"

    loadEnv()

    filename := "playlist.txt" // 입력 파일 이름
    playlistName, songs, err := parseFile(filename)
    fmt.Print(playlistName)

    if err != nil {
        fmt.Println("Error reading file:", err)
        return
    }

    playlistID, err := createPlaylist(jwtToken, playlistName)
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
            fmt.Println("Error adding track to playlist:", err)
            continue
        }

        fmt.Printf("Added song: %s %s to playlist\n", songDetails[0], songDetails[1])
    }
}
