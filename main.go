package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
	"github.com/kobums/playlist/router"
)

// defaultAllowedOrigins keeps production working out of the box while also
// allowing a local frontend — CORS used to permit the production origin only,
// so running the app on localhost was impossible without editing this file.
const defaultAllowedOrigins = "https://playlist.gowoobro.com,http://localhost:9002,http://127.0.0.1:9002"

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	// .env used to be loaded lazily inside the auth controller on every token
	// request; it belongs here, once, at startup.
	if err := godotenv.Load(".env"); err != nil {
		log.Printf(".env 파일을 읽지 못했습니다 (환경변수를 직접 사용합니다): %v", err)
	}

	app := fiber.New(fiber.Config{
		AppName: "apple-music-playlist",
		// Building a large playlist fans out to many Apple Music calls.
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 5 * time.Minute,
	})

	// Without this, a panic in a handler takes the whole process down.
	app.Use(recover.New())

	app.Use(logger.New(logger.Config{
		Format:     "[${time}] | ${status} | ${latency} | ${ip}:${port} | ${method} | ${url}\n",
		TimeFormat: time.DateTime,
	}))

	app.Use(cors.New(cors.Config{
		AllowHeaders:     "Origin, Content-Type, Authorization, Accept",
		AllowCredentials: true,
		AllowOrigins:     env("ALLOWED_ORIGINS", defaultAllowedOrigins),
		AllowMethods: strings.Join([]string{
			fiber.MethodGet,
			fiber.MethodPost,
			fiber.MethodPut,
			fiber.MethodDelete,
		}, ","),
	}))

	router.SetRouter(app)

	addr := ":" + env("PORT", "8002")
	log.Printf("listening on %s", addr)
	log.Fatal(app.Listen(addr))
}
