package handlers

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Check godoc
// @Summary Check health status
// @Description Get the health status of the API service
// @Tags health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /healthz [get]
func (h *HealthHandler) Check(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "healthy",
	})
}
