package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// parseOptionalUUIDParam parses an optional UUID query parameter without
// silently dropping an invalid filter and broadening the requested result set.
func parseOptionalUUIDParam(value, field string) (*uuid.UUID, error) {
	if value == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, field+"格式错误")
	}
	return &id, nil
}
