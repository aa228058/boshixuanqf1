package middleware

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"
)

// TimeoutWithSkip returns a Timeout middleware that does not apply the deadline
// to paths in skipPaths (matched by suffix, e.g. "/progress/stream").
func TimeoutWithSkip(d time.Duration, skipPaths ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path
			for _, sp := range skipPaths {
				if len(path) >= len(sp) && path[len(path)-len(sp):] == sp {
					return next(c)
				}
			}
			ctx, cancel := context.WithTimeout(c.Request().Context(), d)
			defer cancel()
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}
