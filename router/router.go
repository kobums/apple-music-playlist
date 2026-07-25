package router

import (
	"errors"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/kobums/playlist/controllers/rest"
	"github.com/kobums/playlist/models"
)

// SetRouter wires the HTTP routes.
//
// Both handlers used to ignore every error: BodyParser's return value was
// discarded and HandlePlaylist could not report failure at all, so the client
// always received 200 with a partial or empty result.
func SetRouter(app *fiber.App) {
	app.Get("/api/token", func(ctx *fiber.Ctx) error {
		var controller rest.PlaylistController

		result, err := controller.GetDeveloperToken()
		if err != nil {
			log.Printf("developer token failed: %v", err)
			return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"code":  "error",
				"error": "개발자 토큰을 발급하지 못했습니다. 서버 설정을 확인해 주세요.",
			})
		}

		return ctx.JSON(result)
	})

	app.Post("/api/playlist", func(ctx *fiber.Ctx) error {
		item := &models.Playlist{}
		if err := ctx.BodyParser(item); err != nil {
			return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{
				"code":  "error",
				"error": "요청 형식이 올바르지 않습니다.",
			})
		}

		var controller rest.PlaylistController
		controller.Init(ctx)

		if err := controller.HandlePlaylist(ctx.UserContext(), item); err != nil {
			log.Printf("handle playlist failed: %v", err)

			status := http.StatusBadGateway
			code := "error"
			// Never surface err.Error() directly — it carries the raw Apple
			// response body.
			message := "Apple Music에 곡을 추가하지 못했습니다. 잠시 후 다시 시도해 주세요."

			switch {
			case errors.Is(err, rest.ErrMissingUserToken), errors.Is(err, rest.ErrUserTokenInvalid):
				// "unauthorized" tells the browser to drop its stored tokens and
				// send the user back through sign-in, instead of retrying a
				// request that cannot succeed.
				status = http.StatusUnauthorized
				code = "unauthorized"
				message = rest.ErrUserTokenInvalid.Error()
			case errors.Is(err, rest.ErrMissingTitle):
				status = http.StatusBadRequest
				message = rest.ErrMissingTitle.Error()
			}

			return ctx.Status(status).JSON(fiber.Map{
				"code":  code,
				"error": message,
			})
		}

		return ctx.JSON(controller.Result)
	})
}
