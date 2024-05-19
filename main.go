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

    filename := "playlist.txt" // 입력 파일 이름
    playlistName, songs, err := parseFile(filename)
    if err != nil {
        fmt.Println("Error reading file:", err)
        return
    }

    jwtToken, err := generateToken()
    if err != nil {
        fmt.Println("Error generating token:", err)
        return
    }

    playlistID, err := createPlaylist(jwtToken, playlistName)
    if err != nil {
        fmt.Println("Error creating playlist:", err)
        return
    }

    for _, song := range songs {
        songDetails := strings.Split(" - ", song)
        if len(songDetails) != 2 {
            fmt.Println("Invalid song format:", song)
            continue
        }

        searchTerm := fmt.Sprintf("%s %s", songDetails[0], songDetails[1])
        songID, err := searchSong(jwtToken, searchTerm)
        if err != nil {
            fmt.Println("Error searching song:", err)
            continue
        }

        err = addTrackToPlaylist(jwtToken, playlistID, songID)
        if err != nil {
            fmt.Println("Error adding track to playlist:", err)
            continue
        }

        fmt.Printf("Added song: %s by %s to playlist\n", songDetails[0], songDetails[1])
    }
}
