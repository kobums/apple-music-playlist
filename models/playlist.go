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
