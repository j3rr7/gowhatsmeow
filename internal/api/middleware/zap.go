package middleware

import (
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func ZapLogger(log *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)
			if err != nil {
				c.Error(err)
			}

			req := c.Request()
			res := c.Response()

			fields := []zap.Field{
				zap.Int("status", res.Status),
				zap.String("method", req.Method),
				zap.String("path", req.URL.Path),
				zap.String("query", req.URL.RawQuery),
				zap.String("ip", c.RealIP()),
				zap.String("user_agent", req.UserAgent()),
				zap.Duration("latency", time.Since(start)),
			}

			if err != nil {
				fields = append(fields, zap.Error(err))
			}

			n := res.Status
			switch {
			case n >= 500:
				log.Error("Server Error", fields...)
			case n >= 400:
				log.Warn("Client Error", fields...)
			default:
				log.Info("Request handled", fields...)
			}

			return nil
		}
	}
}
