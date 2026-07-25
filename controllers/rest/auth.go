package rest

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
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
	// defaultPrivateKeyPath is used when PRIVATE_KEY_PATH is unset.
	defaultPrivateKeyPath = "AuthKey_GXVS6H2456.p8"
)

var (
	tokenMu     sync.Mutex
	cachedToken string
	cachedUntil time.Time
)

func privateKeyPath() string {
	if path := os.Getenv("PRIVATE_KEY_PATH"); path != "" {
		return path
	}
	return defaultPrivateKeyPath
}

func loadPrivateKey() (*ecdsa.PrivateKey, error) {
	path := privateKeyPath()

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
