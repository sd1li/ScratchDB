package api

import (
	"fmt"

	"github.com/gofiber/contrib/fiberzerolog"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func (a *API) InitializeAPIServer() error {
	app := fiber.New()
	a.app = app

	app.Use(fiberzerolog.New(fiberzerolog.Config{
		Logger: &log.Logger,
		Levels: []zerolog.Level{zerolog.ErrorLevel, zerolog.WarnLevel, zerolog.TraceLevel},
	}))

	a.app.Get("/query", a.AuthMiddleware, a.Query)
	a.app.Post("/query", a.AuthMiddleware, a.Query)
	a.app.Post("/data", a.AuthMiddleware, a.Insert)

	err := app.Listen(fmt.Sprintf(":%d", a.config.Port))
	if err != nil {
		return err
	}

	return nil
}
