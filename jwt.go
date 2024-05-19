package main

import (
	"fmt"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/joho/godotenv"
)

func loadEnv() {
    err := godotenv.Load()
    if err != nil {
        fmt.Println("Error loading .env file")
    }
}

func generateToken() (string, error) {
    teamID := os.Getenv("TEAM_ID")
    keyID := os.Getenv("KEY_ID")
    privateKey := os.Getenv("PRIVATE_KEY")

    token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
        "iss": teamID,
        "iat": time.Now().Unix(),
        "exp": time.Now().Add(time.Hour).Unix(),
    })

    token.Header["kid"] = keyID

    ecdsaKey, err := jwt.ParseECPrivateKeyFromPEM([]byte(privateKey))
    if err != nil {
        return "", err
    }

    tokenString, err := token.SignedString(ecdsaKey)
    if err != nil {
        return "", err
    }

    return tokenString, nil
}
