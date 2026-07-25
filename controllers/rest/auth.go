package rest

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// developerTokenTTL is how long a signed token stays valid. Apple allows up
	// to six months; twelve hours keeps the blast radius small while still
	// avoiding a signature on every request.
	developerTokenTTL = 12 * time.Hour
	// developerTokenSkew forces a refresh this long before actual expiry.
	developerTokenSkew = 30 * time.Minute
	// privateKeyGlob matches Apple's naming for a downloaded MusicKit key.
	privateKeyGlob = "AuthKey_*.p8"
)

var (
	tokenMu     sync.Mutex
	cachedToken string
	cachedUntil time.Time
)

// resolvePrivateKeyPath finds the MusicKit key to sign with.
//
// There is no sensible fixed default: Apple names the file after the Key ID, so
// it changes whenever the key does. This used to fall back to one hard-coded
// filename, which after a key rotation pointed at a key belonging to a
// different Apple Developer team and produced a confusing "file not found".
//
// PRIVATE_KEY_PATH wins when set. Otherwise exactly one AuthKey_*.p8 in dir is
// used — matching how the Dockerfile copies the key in. Zero or several matches
// is ambiguous, so it is an error that says what to do.
func resolvePrivateKeyPath(dir string) (string, error) {
	if path := os.Getenv("PRIVATE_KEY_PATH"); path != "" {
		return path, nil
	}

	matches, err := filepath.Glob(filepath.Join(dir, privateKeyGlob))
	if err != nil {
		return "", fmt.Errorf("개인 키를 찾지 못했습니다: %w", err)
	}

	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return "", fmt.Errorf(
			"MusicKit 개인 키(%s)가 없습니다. .p8 파일을 두거나 PRIVATE_KEY_PATH 를 설정해 주세요",
			privateKeyGlob,
		)
	default:
		return "", fmt.Errorf(
			"MusicKit 개인 키가 여러 개입니다 (%s). PRIVATE_KEY_PATH 로 하나를 지정해 주세요",
			strings.Join(matches, ", "),
		)
	}
}

func loadPrivateKey() (*ecdsa.PrivateKey, error) {
	path, err := resolvePrivateKeyPath(".")
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("개인 키를 읽지 못했습니다 (%s): %w", path, err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("개인 키가 PEM 형식이 아닙니다 (%s)", path)
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("개인 키를 파싱하지 못했습니다: %w", err)
	}

	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("개인 키가 ECDSA 키가 아닙니다")
	}
	return key, nil
}

// DeveloperToken returns a signed MusicKit developer token.
//
// The token is cached until it approaches expiry. The previous code re-read the
// .p8 file and signed a fresh token on every single request, and the identical
// implementation was duplicated across AuthController and PlaylistController.
func DeveloperToken() (string, error) {
	tokenMu.Lock()
	defer tokenMu.Unlock()

	if cachedToken != "" && time.Now().Before(cachedUntil.Add(-developerTokenSkew)) {
		return cachedToken, nil
	}

	teamID := os.Getenv("TEAM_ID")
	keyID := os.Getenv("KEY_ID")
	if teamID == "" || keyID == "" {
		return "", errors.New("TEAM_ID 또는 KEY_ID 환경변수가 설정되지 않았습니다")
	}

	key, err := loadPrivateKey()
	if err != nil {
		return "", err
	}

	now := time.Now()
	expiresAt := now.Add(developerTokenTTL)

	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"iss": teamID,
		"iat": now.Unix(),
		"exp": expiresAt.Unix(),
	})
	token.Header["kid"] = keyID

	signed, err := token.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("개발자 토큰 서명에 실패했습니다: %w", err)
	}

	cachedToken = signed
	cachedUntil = expiresAt
	return signed, nil
}
