package rest

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

// The 401/403 → "sign in again" mapping decides whether a user is told to
// re-authorise or the failure is reported as a server problem. Getting it wrong
// in either direction is bad: blaming the user for our own expired developer
// token, or making them retry a request that can never succeed.
func TestAppleAPIErrorUnwrap(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		status        int
		wantUserToken bool
	}{
		{
			name:          "403 on a library endpoint is the user's session",
			path:          "/v1/me/library/playlists?limit=100",
			status:        http.StatusForbidden,
			wantUserToken: true,
		},
		{
			name:          "401 on a library endpoint is the user's session",
			path:          "/v1/me/library/playlists",
			status:        http.StatusUnauthorized,
			wantUserToken: true,
		},
		{
			name:          "401 on catalog is our developer token, not the user",
			path:          "/v1/catalog/kr/search?term=x&types=songs",
			status:        http.StatusUnauthorized,
			wantUserToken: false,
		},
		{
			name:          "403 on catalog is our developer token, not the user",
			path:          "/v1/catalog/kr/search",
			status:        http.StatusForbidden,
			wantUserToken: false,
		},
		{
			name:          "429 is rate limiting, not an auth problem",
			path:          "/v1/me/library/playlists",
			status:        http.StatusTooManyRequests,
			wantUserToken: false,
		},
		{
			name:          "500 is Apple having a bad day",
			path:          "/v1/me/library/playlists",
			status:        http.StatusInternalServerError,
			wantUserToken: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &appleAPIError{
				Method:     http.MethodGet,
				Path:       tt.path,
				StatusCode: tt.status,
				Body:       `{"errors":[]}`,
			}

			if got := errors.Is(err, ErrUserTokenInvalid); got != tt.wantUserToken {
				t.Errorf("errors.Is(err, ErrUserTokenInvalid) = %v, want %v", got, tt.wantUserToken)
			}
		})
	}
}

// The raw Apple response body must stay in the logs. The router answers with a
// fixed message instead, so an upstream payload never reaches the browser.
func TestAppleAPIErrorKeepsDetailForLogs(t *testing.T) {
	err := &appleAPIError{
		Method:     http.MethodGet,
		Path:       "/v1/me/library/playlists",
		StatusCode: http.StatusForbidden,
		Body:       `{"errors":[{"id":"ABC","title":"Forbidden"}]}`,
	}

	message := err.Error()
	for _, want := range []string{"GET", "/v1/me/library/playlists", "403", "Forbidden"} {
		if !strings.Contains(message, want) {
			t.Errorf("Error() = %q, missing %q", message, want)
		}
	}
}
