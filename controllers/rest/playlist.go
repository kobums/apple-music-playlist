package rest

import (
	"context"
	"errors"
	"log"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/kobums/playlist/controllers"
	"github.com/kobums/playlist/models"
)

// addTracksChunkSize bounds how many songs go into one add-tracks request.
const addTracksChunkSize = 100

// ErrMissingUserToken is returned when the request carries no Music User Token.
var ErrMissingUserToken = errors.New("Apple Music 사용자 토큰이 없습니다. 다시 로그인해 주세요")

// ErrMissingTitle is returned when the playlist title is blank.
var ErrMissingTitle = errors.New("플레이리스트 제목이 없습니다")

// ErrMissingQuery is returned when a search request carries no search term.
var ErrMissingQuery = errors.New("검색어가 없습니다")

// ErrMissingPlaylistID is returned when a track-add request names no playlist.
var ErrMissingPlaylistID = errors.New("플레이리스트를 찾을 수 없습니다. 처음부터 다시 시도해 주세요")

type PlaylistController struct {
	controllers.Controller
}

// TrackResult is the outcome for one input line.
//
// Status replaced a plain bool: "added" and "not found" were the only two
// outcomes the old API could express, so a low-confidence guess had to be
// reported as a success. "review" is the third case — we found something
// plausible but will not add it without a person agreeing.
type TrackResult struct {
	Song       string      `json:"song"`
	Status     MatchStatus `json:"status"`
	Score      float64     `json:"score,omitempty"`
	Track      *Track      `json:"track,omitempty"`
	Candidates []Track     `json:"candidates,omitempty"`
}

// GetDeveloperToken returns a signed MusicKit developer token.
//
// Previously a signing failure was swallowed and the literal string
// "errorerror" was handed to the browser as if it were a token.
func (c *PlaylistController) GetDeveloperToken() (fiber.Map, error) {
	token, err := DeveloperToken()
	if err != nil {
		return nil, err
	}

	return fiber.Map{
		"code":           "ok",
		"developerToken": token,
	}, nil
}

var trailingTimestamp = regexp.MustCompile(`\s*\d{1,2}:\d{2}(?::\d{2})?`)

// parsePlaylist splits the pasted text into one entry per song.
//
// Blank lines used to survive as empty entries, so a trailing newline in the
// textarea produced a phantom "failed" row in the results.
func (c *PlaylistController) parsePlaylist(playlist string) []string {
	var songs []string

	for _, line := range strings.Split(playlist, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "playlistname:") {
			continue
		}

		song := strings.TrimSpace(trailingTimestamp.ReplaceAllString(line, ""))
		if song == "" {
			continue
		}
		songs = append(songs, song)
	}

	return songs
}

// HandlePlaylist resolves (or creates) the target playlist and adds every song.
//
// Errors used to be printed and then ignored, so a failed lookup or a failed
// creation left playlistID empty and the handler carried on adding tracks to
// nothing — the user saw every song reported as "failed" with no reason.
func (c *PlaylistController) HandlePlaylist(ctx context.Context, item *models.Playlist) error {
	title := strings.TrimSpace(item.Title)
	if title == "" {
		return ErrMissingTitle
	}
	if strings.TrimSpace(item.UserToken) == "" {
		return ErrMissingUserToken
	}

	// Never trust a developer token supplied by the client; sign a fresh one.
	developerToken, err := DeveloperToken()
	if err != nil {
		return err
	}

	client := newAppleMusicClient(developerToken, item.UserToken)
	songs := c.parsePlaylist(item.List)

	playlists, err := client.LibraryPlaylists(ctx)
	if err != nil {
		return err
	}

	playlistID, found := findPlaylistByName(playlists, title)
	if !found {
		playlistID, err = client.CreatePlaylist(ctx, title)
		if err != nil {
			return err
		}
	}

	storefront := client.Storefront(ctx)

	results := make([]TrackResult, len(songs))
	// Index of each confidently matched song, so a failed add can be mapped back
	// to the right rows.
	var pendingIndexes []int
	var pendingIDs []string

	for index, song := range songs {
		normalised := transformText(song)
		results[index] = TrackResult{Song: normalised, Status: MatchMissing}

		artist, songTitle, ok := splitArtistTitle(normalised)
		if !ok {
			continue
		}

		candidates, err := client.SearchSongs(ctx, storefront, artist+" "+songTitle)
		if err != nil {
			log.Printf("search failed for %q: %v", normalised, err)
			continue
		}

		match := BestMatch(artist, songTitle, candidates)
		results[index].Status = match.Status
		results[index].Score = match.Score
		results[index].Track = match.Track
		results[index].Candidates = match.Candidates

		if match.Status == MatchAdded {
			pendingIndexes = append(pendingIndexes, index)
			pendingIDs = append(pendingIDs, match.Track.ID)
		}
	}

	for start := 0; start < len(pendingIDs); start += addTracksChunkSize {
		end := min(start+addTracksChunkSize, len(pendingIDs))

		if err := client.AddTracks(ctx, playlistID, pendingIDs[start:end]); err != nil {
			log.Printf("add tracks failed (%d-%d): %v", start, end, err)
			// The track was identified but never made it in. "review" keeps the
			// match on screen with a way to add it, which "missing" would throw
			// away and "added" would lie about.
			for _, index := range pendingIndexes[start:end] {
				results[index].Status = MatchReview
			}
			continue
		}
	}

	// The playlist id lets the browser add confirmed picks afterwards without
	// re-resolving the playlist by name.
	c.Set("playlistId", playlistID)
	c.Set("result", results)
	return nil
}

// SearchTracks resolves a free-text query to catalog candidates, for the
// "이 곡이 아닌데" retry on the results screen.
func (c *PlaylistController) SearchTracks(ctx context.Context, userToken, query string) ([]Track, error) {
	if strings.TrimSpace(userToken) == "" {
		return nil, ErrMissingUserToken
	}
	if strings.TrimSpace(query) == "" {
		return nil, ErrMissingQuery
	}

	developerToken, err := DeveloperToken()
	if err != nil {
		return nil, err
	}

	client := newAppleMusicClient(developerToken, userToken)
	return client.SearchSongs(ctx, client.Storefront(ctx), strings.TrimSpace(query))
}

// AddConfirmedTracks appends tracks the user picked to an existing playlist.
func (c *PlaylistController) AddConfirmedTracks(
	ctx context.Context,
	userToken, playlistID string,
	songIDs []string,
) error {
	if strings.TrimSpace(userToken) == "" {
		return ErrMissingUserToken
	}
	if strings.TrimSpace(playlistID) == "" {
		return ErrMissingPlaylistID
	}
	if len(songIDs) == 0 {
		return nil
	}

	developerToken, err := DeveloperToken()
	if err != nil {
		return err
	}

	client := newAppleMusicClient(developerToken, userToken)

	for start := 0; start < len(songIDs); start += addTracksChunkSize {
		end := min(start+addTracksChunkSize, len(songIDs))
		if err := client.AddTracks(ctx, playlistID, songIDs[start:end]); err != nil {
			return err
		}
	}
	return nil
}

// splitArtistTitle splits "artist - title" into its two halves.
//
// The old code used strings.Split and required exactly two parts, so any title
// that itself contained " - " (remixes, subtitles) was rejected outright.
func splitArtistTitle(input string) (artist, title string, ok bool) {
	parts := strings.SplitN(input, " - ", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	artist = strings.TrimSpace(parts[0])
	title = strings.TrimSpace(parts[1])
	if artist == "" || title == "" {
		return "", "", false
	}
	return artist, title, true
}

// removeParentheses drops a trailing parenthesised segment.
func removeParentheses(s string) string {
	if index := strings.Index(s, "("); index != -1 {
		return strings.TrimSpace(s[:index])
	}
	return s
}

var (
	// "07:19 | 아티스트 ‘곡 제목’"
	quotedPattern = regexp.MustCompile(`\|\s*(.*?)\s*‘(.*?)’`)
	// "07:19 | 아티스트 - 곡 제목"
	pipedPattern = regexp.MustCompile(`\|\s*(.*?)\s*-\s*(.*)`)
	// "아티스트 - 곡 제목"
	plainPattern = regexp.MustCompile(`^(.*?)\s*-\s*(.*)$`)
)

// transformText normalises the supported input shapes to "artist - title".
func transformText(input string) string {
	var artist, title string

	switch {
	case len(quotedPattern.FindStringSubmatch(input)) >= 3:
		matches := quotedPattern.FindStringSubmatch(input)
		artist, title = matches[1], matches[2]
	case len(pipedPattern.FindStringSubmatch(input)) >= 3:
		matches := pipedPattern.FindStringSubmatch(input)
		artist, title = matches[1], matches[2]
	case len(plainPattern.FindStringSubmatch(input)) >= 3:
		matches := plainPattern.FindStringSubmatch(input)
		artist, title = matches[1], matches[2]
	default:
		return input
	}

	return removeParentheses(artist) + " - " + removeParentheses(title)
}
