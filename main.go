package main

import (
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/kobums/playlist/router"
)

func main() {
	app := fiber.New()
	app.Use(logger.New(logger.Config{
		Format:     "[${time}] | ${status} | ${latency} | ${ip}:${port} | ${method} | ${url}\n",
		TimeFormat: time.DateTime,
	}))

	app.Use(cors.New(cors.Config{
		AllowHeaders:     "Origin, Content-Type, Authorization, Accept",
		AllowCredentials: true,
		AllowOrigins:     "http://localhost:9002, http://140.82.12.99:9002, http://www.gowoobro.com:9002, http://playlist.gowoobro.com, https://140.82.12.99:9002, https://www.gowoobro.com:9002, https://playlist.gowoobro.com",
		AllowMethods: strings.Join([]string{
			fiber.MethodGet,
			fiber.MethodPost,
			fiber.MethodPut,
			fiber.MethodDelete,
			// fiber.MethodHead,
			// fiber.MethodPatch,
		}, ","),
	}))

	router.SetRouter(app)

	log.Fatal(app.Listen(":8002"))
}
