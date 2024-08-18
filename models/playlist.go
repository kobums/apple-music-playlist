package models

type Playlist struct {
	Title          string `json:"title"`
	List           string `json:"list"`
	UserToken      string `json:"usertoken"`
	DeveloperToken string `json:"developertoken"`
}
