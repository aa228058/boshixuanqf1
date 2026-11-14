package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TestTenantIsolationRejectsMissingTenant(t *testing.T) {
	e := echo.New()
	e.Use(TenantIsolation())
	e.GET("/x", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestTenantIsolationAcceptsValidTenant(t *testing.T) {
	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(TenantContextKey, uuid.New().String())
			return next(c)
		}
	})
	e.Use(TenantIsolation())
	e.GET("/x", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

// 上下文里放了非字符串（例如误塞了 uuid.UUID）不能当作有效租户。
func TestGetTenantIDRejectsWrongType(t *testing.T) {
	e := echo.New()
	e.GET("/x", func(c echo.Context) error {
		c.Set(TenantContextKey, uuid.New()) // 故意放 uuid.UUID 而不是 string
		if _, err := GetTenantID(c); err == nil {
			t.Fatal("非字符串租户 ID 应报错")
		}
		return c.NoContent(http.StatusOK)
	})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
}

func TestGetTenantIDRejectsInvalidUUID(t *testing.T) {
	e := echo.New()
	e.GET("/x", func(c echo.Context) error {
		c.Set(TenantContextKey, "not-a-uuid")
		if _, err := GetTenantID(c); err == nil {
			t.Fatal("非法 UUID 应报错")
		}
		return c.NoContent(http.StatusOK)
	})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
}
