package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
)

// handleServiceError统一处理service层错误，区分NoRows和内部错误
func handleServiceError(c echo.Context, err error, notFoundMsg string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return echo.NewHTTPError(http.StatusNotFound, notFoundMsg)
	}
	log.Printf("[handler] 服务错误: %v", err)
	return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
}

// handleBindError统一处理请求体bind错误
func handleBindError(c echo.Context, err error) error {
	log.Printf("[handler] 请求参数错误: %v", err)
	return echo.NewHTTPError(http.StatusBadRequest, "请求参数错误")
}


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
