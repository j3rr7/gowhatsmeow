package api

import (
	"gowhatsmeow/internal/api/handlers"

	"github.com/labstack/echo/v4"
)

func RegisterRoutes(e *echo.Echo, health *handlers.HealthHandler, wa *handlers.WhatsAppHandler) {
	e.GET("/healthz", health.Check)

	v1 := e.Group("/api/v1")
	{
		whatsapp := v1.Group("/whatsapp")
		{
			whatsapp.GET("/status", wa.GetStatus)
			whatsapp.GET("/qr", wa.GetQR)
			whatsapp.POST("/send", wa.SendMessage)
			whatsapp.POST("/send-file", wa.SendFile)
			whatsapp.POST("/logout", wa.Logout)
		}
	}
}
