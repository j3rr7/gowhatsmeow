package main

import (
	"gowhatsmeow/internal/api"
	"gowhatsmeow/internal/api/handlers"
	"gowhatsmeow/internal/config"
	"gowhatsmeow/internal/db"
	"gowhatsmeow/internal/logger"
	"gowhatsmeow/internal/whatsapp"

	"go.uber.org/fx"
)

// @title Whatsapp API
// @version 1.0
// @description Whatsapp API
// @servers http://localhost:8080
func main() {
	app := fx.New(
		fx.Provide(
			config.LoadConfig,

			logger.NewZapLogger,

			logger.NewZapWAAdapter,

			db.NewSQLContainer,

			whatsapp.NewWhatsAppManager,
			whatsapp.NewWhatsAppService,

			api.NewEchoServer,
			handlers.NewHealthHandler,
			handlers.NewWhatsAppHandler,
		),

		fx.Invoke(
			api.RegisterRoutes,

			func(mgr *whatsapp.WhatsAppManager) {},
		),
	)

	app.Run()
}
