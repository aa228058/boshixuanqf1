package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func roleHarness(t *testing.T, role string) *echo.Echo {
	t.Helper()
	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if role != "" {
				c.Set("admin_role", role)
			}
			return next(c)
		}
	})
	e.DELETE("/admin/tenants/1", func(c echo.Context) error { return c.NoContent(http.StatusNoContent) },
		RequireRole("super"))
	return e
}

func TestRequireRoleAllowsMatchingRole(t *testing.T) {
	rec := httptest.NewRecorder()
	roleHarness(t, "super").ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/admin/tenants/1", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

func TestRequireRoleBlocksOtherRole(t *testing.T) {
	rec := httptest.NewRecorder()
	roleHarness(t, "operator").ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/admin/tenants/1", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

// 没有 admin_role 时按拒绝处理（漏配 AdminAuth 不能变成放行）。
func TestRequireRoleBlocksMissingRole(t *testing.T) {
	rec := httptest.NewRecorder()
	roleHarness(t, "").ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/admin/tenants/1", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}
