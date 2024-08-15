package main

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/joho/godotenv"
)

func loadEnv() {
    err := godotenv.Load(".env")
    if err != nil {
        fmt.Println("Error loading .env file")
    }
}

func generateToken() (string, error) {
    teamID := os.Getenv("TEAM_ID")
	keyID := os.Getenv("KEY_ID")

	privateKeyData, err := os.ReadFile("AuthKey_GXVS6H2456.p8")
	if err != nil {
		return "", err
	}

	block, _ := pem.Decode(privateKeyData)
	if block == nil {
		return "", fmt.Errorf("failed to parse PEM block containing the key")
	}

	parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %v", err)
	}

	ecdsaKey, ok := parsedKey.(*ecdsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("not ECDSA private key")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"iss": teamID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	token.Header["kid"] = keyID

	tokenString, err := token.SignedString(ecdsaKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}