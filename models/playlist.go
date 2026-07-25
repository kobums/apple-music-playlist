package models

// Playlist is the request body for POST /api/playlist.
//
// The JSON tags used to be lowercased ("usertoken"), which only worked because
// encoding/json falls back to a case-insensitive match against the camelCase
// keys the frontend actually sends. They now match the wire format exactly.
type Playlist struct {
	Title string `json:"title"`
	List  string `json:"list"`
	// UserToken is the Music User Token obtained by MusicKit in the browser.
	UserToken string `json:"userToken"`
	// DeveloperToken is accepted for backwards compatibility but ignored — the
	// server signs its own token rather than trusting a client-supplied one.
	DeveloperToken string `json:"developerToken"`
}

// SearchRequest is the body for POST /api/search, used when someone rejects a
// suggested match and searches for the right track themselves.
type SearchRequest struct {
	UserToken string `json:"userToken"`
	Query     string `json:"query"`
}

// AddTracksRequest is the body for POST /api/playlist/tracks, used to add the
// tracks a person confirmed on the results screen.
type AddTracksRequest struct {
	UserToken  string   `json:"userToken"`
	PlaylistID string   `json:"playlistId"`
	SongIDs    []string `json:"songIds"`
}
