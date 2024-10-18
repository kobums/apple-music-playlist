package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/kobums/playlist/controllers/rest"
	"github.com/kobums/playlist/models"
)

func SetRouter(app *fiber.App) {
	app.Get("/api/token", func(ctx *fiber.Ctx) error {
		var controller rest.PlaylistController
		return ctx.JSON(controller.GetDeveloperToken())
	})

	app.Post("/api/playlist", func(ctx *fiber.Ctx) error {
		item_ := &models.Playlist{}
		ctx.BodyParser(item_)
		var controller rest.PlaylistController
		controller.Init(ctx)
		controller.HandlePlaylist(item_)
		return ctx.JSON(controller.Result)
	})
}
