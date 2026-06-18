package middleware

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"go.uber.org/zap"
)

func ZapLogger(log *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			start := time.Now()

			err := next(c)
			if err != nil {
				c.Echo().HTTPErrorHandler(c, err)
			}

			req := c.Request()

			status := http.StatusOK
			rw, uErr := echo.UnwrapResponse(c.Response())
			if uErr == nil {
				status = rw.Status
			} else if err != nil {
				var httpErr *echo.HTTPError
				if errors.As(err, &httpErr) {
					status = httpErr.Code
				} else {
					status = http.StatusInternalServerError
				}
			}

			fields := []zap.Field{
				zap.Int("status", status),
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

			switch {
			case status >= 500:
				log.Error("Server Error", fields...)
			case status >= 400:
				log.Warn("Client Error", fields...)
			default:
				log.Info("Request handled", fields...)
			}

			return nil
		}
	}
}
