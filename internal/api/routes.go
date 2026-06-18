package api

import (
	"gowhatsmeow/internal/api/handlers"

	"github.com/labstack/echo/v4"
)

// @title Whatsapp API
// @version 1.0
// @description Whatsapp API

// @host localhost:8080
// @BasePath /
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
			whatsapp.POST("/check-number", wa.CheckNumber)
			whatsapp.GET("/webhook", wa.GetWebhook)
			whatsapp.POST("/webhook", wa.SetWebhook)
		}
	}
}
