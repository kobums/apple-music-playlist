package router

import (
	"errors"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/kobums/playlist/controllers/rest"
	"github.com/kobums/playlist/models"
)

// fail turns an internal error into a client response.
//
// Handlers used to ignore errors entirely and always answer 200. This is the
// single place that decides status and wording, so a raw Apple response body
// never reaches the browser and "sign in again" stays consistent across routes.
func fail(ctx *fiber.Ctx, context string, err error) error {
	log.Printf("%s failed: %v", context, err)

	status := http.StatusBadGateway
	code := "error"
	message := "Apple Music 요청에 실패했습니다. 잠시 후 다시 시도해 주세요."

	switch {
	case errors.Is(err, rest.ErrMissingUserToken), errors.Is(err, rest.ErrUserTokenInvalid):
		// "unauthorized" tells the browser to drop its stored tokens and send
		// the user back through sign-in, instead of retrying a request that
		// cannot succeed.
		status = http.StatusUnauthorized
		code = "unauthorized"
		message = rest.ErrUserTokenInvalid.Error()
	case errors.Is(err, rest.ErrMissingTitle):
		status = http.StatusBadRequest
		message = rest.ErrMissingTitle.Error()
	case errors.Is(err, rest.ErrMissingQuery):
		status = http.StatusBadRequest
		message = rest.ErrMissingQuery.Error()
	case errors.Is(err, rest.ErrMissingPlaylistID):
		status = http.StatusBadRequest
		message = rest.ErrMissingPlaylistID.Error()
	}

	return ctx.Status(status).JSON(fiber.Map{"code": code, "error": message})
}

func badRequest(ctx *fiber.Ctx) error {
	return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{
		"code":  "error",
		"error": "요청 형식이 올바르지 않습니다.",
	})
}

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
			return badRequest(ctx)
		}

		var controller rest.PlaylistController
		controller.Init(ctx)

		if err := controller.HandlePlaylist(ctx.UserContext(), item); err != nil {
			return fail(ctx, "handle playlist", err)
		}

		return ctx.JSON(controller.Result)
	})

	// Playlists the user can add to, so a target is picked rather than typed.
	app.Post("/api/playlists", func(ctx *fiber.Ctx) error {
		item := &models.PlaylistsRequest{}
		if err := ctx.BodyParser(item); err != nil {
			return badRequest(ctx)
		}

		var controller rest.PlaylistController
		controller.Init(ctx)

		playlists, err := controller.ListPlaylists(ctx.UserContext(), item.UserToken)
		if err != nil {
			return fail(ctx, "list playlists", err)
		}

		controller.Set("playlists", playlists)
		return ctx.JSON(controller.Result)
	})

	// Used when someone rejects a suggested match and looks for the right track.
	app.Post("/api/search", func(ctx *fiber.Ctx) error {
		item := &models.SearchRequest{}
		if err := ctx.BodyParser(item); err != nil {
			return badRequest(ctx)
		}

		var controller rest.PlaylistController
		controller.Init(ctx)

		tracks, err := controller.SearchTracks(ctx.UserContext(), item.UserToken, item.Query)
		if err != nil {
			return fail(ctx, "search tracks", err)
		}

		controller.Set("candidates", tracks)
		return ctx.JSON(controller.Result)
	})

	// Adds the tracks a person confirmed on the results screen.
	app.Post("/api/playlist/tracks", func(ctx *fiber.Ctx) error {
		item := &models.AddTracksRequest{}
		if err := ctx.BodyParser(item); err != nil {
			return badRequest(ctx)
		}

		var controller rest.PlaylistController
		controller.Init(ctx)

		outcome, err := controller.AddConfirmedTracks(
			ctx.UserContext(), item.UserToken, item.PlaylistID, item.SongIDs,
		)
		if err != nil {
			return fail(ctx, "add confirmed tracks", err)
		}

		controller.Set("added", outcome.Added)
		controller.Set("duplicate", outcome.Duplicate)
		return ctx.JSON(controller.Result)
	})
}
