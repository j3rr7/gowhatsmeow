package api

import (
	"context"
	"fmt"
	apiMiddleware "gowhatsmeow/internal/api/middleware"
	"gowhatsmeow/internal/config"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewEchoServer(lc fx.Lifecycle, cfg *config.Config, zapLog *zap.Logger) *echo.Echo {
	e := echo.New()

	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
	}))
	e.Use(apiMiddleware.ZapLogger(zapLog))

	srvCtx, cancelServer := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
			zapLog.Info("Starting Echo server", zap.String("address", addr))

			go func() {
				sc := echo.StartConfig{
					Address: addr,
				}
				if err := sc.Start(srvCtx, e); err != nil && err != http.ErrServerClosed {
					zapLog.Fatal("Echo server failed to start", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			zapLog.Info("Shutting down Echo server...")
			cancelServer()
			return nil
		},
	})

	return e
}
