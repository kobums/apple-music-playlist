package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const appleMusicBaseURL = "https://api.music.apple.com"

// defaultStorefront is only a fallback. The storefront used to be hard-coded to
// "us", so Korean-language searches ran against the US catalog and frequently
// returned nothing; the real storefront is now read from the user's account.
const defaultStorefront = "kr"

// maxResponseBytes caps how much of an Apple Music response we buffer, so a
// malformed or huge reply cannot exhaust memory.
const maxResponseBytes = 8 << 20

// ErrUserTokenInvalid means Apple rejected the Music User Token, so the browser
// has to authorise again. This is what every client sees after the MusicKit key
// moves to a different Apple Developer team: user tokens are bound to the team
// that issued them, so every previously stored token stops working at once.
var ErrUserTokenInvalid = errors.New("Apple Music 인증이 만료되었습니다. 다시 로그인해 주세요")

// appleAPIError carries the HTTP status of a failed Apple Music call so callers
// can tell "this user must sign in again" apart from "Apple is having a bad
// day". Previously every non-2xx collapsed into one opaque string and the API
// answered 502 for all of them, leaving users to retry a request that could
// never succeed.
type appleAPIError struct {
	Method     string
	Path       string
	StatusCode int
	Body       string
}

func (e *appleAPIError) Error() string {
	return fmt.Sprintf("apple music 오류 (%s %s): %d %s", e.Method, e.Path, e.StatusCode, e.Body)
}

// Unwrap reports ErrUserTokenInvalid for the failures a re-login actually fixes.
//
// Apple returns 401 when the developer token is bad and 403 when the Music User
// Token is, but the split is not clean in practice. The reliable signal is the
// endpoint: /v1/me/ requests are the only ones that need a user token, so a
// rejection there is the user's session — a rejection on a catalog endpoint is
// the server's own credentials and must not be blamed on the user.
func (e *appleAPIError) Unwrap() error {
	if e.StatusCode != http.StatusUnauthorized && e.StatusCode != http.StatusForbidden {
		return nil
	}
	if !strings.HasPrefix(e.Path, "/v1/me/") {
		return nil
	}
	return ErrUserTokenInvalid
}

// appleMusicClient wraps the Apple Music API calls this service needs.
//
// Every request carries a timeout — the previous code used a bare
// &http.Client{}, which waits forever if Apple stops responding.
type appleMusicClient struct {
	developerToken string
	userToken      string
	httpClient     *http.Client
}

func newAppleMusicClient(developerToken, userToken string) *appleMusicClient {
	return &appleMusicClient{
		developerToken: developerToken,
		userToken:      userToken,
		httpClient:     &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *appleMusicClient) do(ctx context.Context, method, path string, body []byte, out any) error {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, appleMusicBaseURL+path, reader)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.developerToken)
	req.Header.Set("Music-User-Token", c.userToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("apple music 요청 실패 (%s %s): %w", method, path, err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return fmt.Errorf("apple music 응답 읽기 실패 (%s %s): %w", method, path, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &appleAPIError{
			Method:     method,
			Path:       path,
			StatusCode: resp.StatusCode,
			Body:       truncate(string(payload), 300),
		}
	}

	if out == nil || len(payload) == 0 {
		return nil
	}

	if err := json.Unmarshal(payload, out); err != nil {
		return fmt.Errorf("apple music 응답 파싱 실패 (%s %s): %w", method, path, err)
	}
	return nil
}

func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "…"
}

type storefrontResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// Storefront returns the storefront id (e.g. "kr") tied to the signed-in
// account, falling back to defaultStorefront when it cannot be determined.
func (c *appleMusicClient) Storefront(ctx context.Context) string {
	var result storefrontResponse
	if err := c.do(ctx, http.MethodGet, "/v1/me/storefront", nil, &result); err != nil {
		return defaultStorefront
	}
	if len(result.Data) == 0 || result.Data[0].ID == "" {
		return defaultStorefront
	}
	return result.Data[0].ID
}

type libraryPlaylist struct {
	ID         string `json:"id"`
	Attributes struct {
		Name string `json:"name"`
	} `json:"attributes"`
}

type libraryPlaylistsResponse struct {
	Data []libraryPlaylist `json:"data"`
	Next string            `json:"next"`
}

// LibraryPlaylists lists every playlist in the user's library.
//
// The old implementation stopped at the first 100, so a user with more than
// that would get a duplicate playlist created instead of the existing one being
// reused. This follows the `next` cursor to the end.
func (c *appleMusicClient) LibraryPlaylists(ctx context.Context) ([]libraryPlaylist, error) {
	var all []libraryPlaylist
	path := "/v1/me/library/playlists?limit=100"

	// Bounded so a misbehaving cursor cannot loop forever.
	for page := 0; page < 50 && path != ""; page++ {
		var result libraryPlaylistsResponse
		if err := c.do(ctx, http.MethodGet, path, nil, &result); err != nil {
			return nil, err
		}
		all = append(all, result.Data...)
		path = result.Next
	}

	return all, nil
}

type createPlaylistRequest struct {
	Attributes struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"attributes"`
}

// CreatePlaylist creates a library playlist and returns its id.
func (c *appleMusicClient) CreatePlaylist(ctx context.Context, name string) (string, error) {
	// The request body used to be built with fmt.Sprintf, so a playlist name
	// containing a quote or backslash produced invalid JSON.
	var payload createPlaylistRequest
	payload.Attributes.Name = name
	payload.Attributes.Description = "Generated by MusicKit API"

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	var result libraryPlaylistsResponse
	if err := c.do(ctx, http.MethodPost, "/v1/me/library/playlists", body, &result); err != nil {
		return "", err
	}
	if len(result.Data) == 0 || result.Data[0].ID == "" {
		return "", fmt.Errorf("플레이리스트 생성 응답에 id가 없습니다")
	}
	return result.Data[0].ID, nil
}

type searchResponse struct {
	Results struct {
		Songs struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		} `json:"songs"`
	} `json:"results"`
}

// SearchSong returns the catalog id of the best match for term.
//
// The old version reached into the decoded map with a chain of type assertions
// and relied on `recover()` to survive a missing key. Decoding into a struct
// makes a missing field an ordinary empty value instead.
func (c *appleMusicClient) SearchSong(ctx context.Context, storefront, term string) (string, error) {
	path := fmt.Sprintf(
		"/v1/catalog/%s/search?term=%s&types=songs&limit=1",
		url.PathEscape(storefront),
		url.QueryEscape(term),
	)

	var result searchResponse
	if err := c.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return "", err
	}

	songs := result.Results.Songs.Data
	if len(songs) == 0 || songs[0].ID == "" {
		return "", fmt.Errorf("검색 결과가 없습니다: %s", term)
	}
	return songs[0].ID, nil
}

type trackRef struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type addTracksRequest struct {
	Data []trackRef `json:"data"`
}

// AddTracks appends catalog songs to a library playlist in a single request.
func (c *appleMusicClient) AddTracks(ctx context.Context, playlistID string, songIDs []string) error {
	if len(songIDs) == 0 {
		return nil
	}

	payload := addTracksRequest{Data: make([]trackRef, 0, len(songIDs))}
	for _, id := range songIDs {
		payload.Data = append(payload.Data, trackRef{ID: id, Type: "songs"})
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/v1/me/library/playlists/%s/tracks", url.PathEscape(playlistID))
	return c.do(ctx, http.MethodPost, path, body, nil)
}

// findPlaylistByName looks up an existing playlist by exact name, ignoring
// surrounding whitespace.
func findPlaylistByName(playlists []libraryPlaylist, name string) (string, bool) {
	target := strings.TrimSpace(name)
	for _, playlist := range playlists {
		if playlist.ID == "" {
			continue
		}
		if strings.TrimSpace(playlist.Attributes.Name) == target {
			return playlist.ID, true
		}
	}
	return "", false
}
