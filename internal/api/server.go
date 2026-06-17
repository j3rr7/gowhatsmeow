package api

import (
	"context"
	"fmt"
	apiMiddleware "gowhatsmeow/internal/api/middleware"
	"gowhatsmeow/internal/config"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewEchoServer(lc fx.Lifecycle, cfg *config.Config, zapLog *zap.Logger) *echo.Echo {
	e := echo.New()

	e.HideBanner = true

	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(apiMiddleware.ZapLogger(zapLog))

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
			zapLog.Info("Starting Echo server", zap.String("address", addr))

			go func() {
				if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
					zapLog.Fatal("Echo server failed to start", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			zapLog.Info("Shutting down Echo server...")
			return e.Shutdown(ctx)
		},
	})

	return e
}
