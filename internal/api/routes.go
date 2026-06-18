package api

import (
	"strings"

	"gowhatsmeow/docs"
	"gowhatsmeow/internal/api/handlers"

	"github.com/labstack/echo/v5"
	echoSwagger "github.com/swaggo/echo-swagger/v2"
)

func RegisterRoutes(e *echo.Echo, health *handlers.HealthHandler, wa *handlers.WhatsAppHandler) {
	// Swagger UI in echo-swagger/v2 expects "openapi": "3.0.0" instead of "3.1.0"
	if docs.SwaggerInfo != nil {
		docs.SwaggerInfo.SwaggerTemplate = strings.ReplaceAll(docs.SwaggerInfo.SwaggerTemplate, `"openapi": "3.1.0"`, `"openapi": "3.0.0"`)
	}

	e.GET("/swagger/*", echoSwagger.WrapHandlerV3)
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
