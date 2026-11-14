package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"tg-sender/internal/middleware"
)

// paramHarness 装配一个带租户上下文的路由，用于验证"参数非法时必须在碰 service 之前拒绝"。
// 传入的 handler 是零值结构（service 为 nil）：如果校验顺序写错，测试会直接 panic。
func paramHarness(method, path, paramValue string, h echo.HandlerFunc) *echo.Echo {
	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(middleware.TenantContextKey, uuid.New().String())
			return next(c)
		}
	})
	e.Add(method, path, h)
	return e
}

func doParam(e *echo.Echo, method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestTaskHandlerGetByIDRejectsMalformedUUID(t *testing.T) {
	h := &TaskHandler{}
	e := paramHarness(http.MethodGet, "/tasks/:id", "", h.GetByID)
	if rec := doParam(e, http.MethodGet, "/tasks/not-a-uuid"); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestAccountHandlerGetByIDRejectsMalformedUUID(t *testing.T) {
	h := &AccountHandler{}
	e := paramHarness(http.MethodGet, "/accounts/:id", "", h.GetByID)
	if rec := doParam(e, http.MethodGet, "/accounts/12345"); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestGroupHandlerGetByIDRejectsMalformedUUID(t *testing.T) {
	h := &GroupHandler{}
	e := paramHarness(http.MethodGet, "/groups/:id", "", h.GetByID)
	if rec := doParam(e, http.MethodGet, "/groups/../etc/passwd"); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestMarkAnnouncementReadRejectsMalformedUUID(t *testing.T) {
	h := &SettingsHandler{}
	e := paramHarness(http.MethodPost, "/announcements/:id/read", "", h.MarkAnnouncementRead)
	if rec := doParam(e, http.MethodPost, "/announcements/not-a-uuid/read"); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// 没有租户上下文时必须 401，而不是继续往下走到 service。
func TestHandlersRequireTenantContext(t *testing.T) {
	e := echo.New()
	e.GET("/tasks/:id", (&TaskHandler{}).GetByID)
	e.GET("/accounts/:id", (&AccountHandler{}).GetByID)
	e.GET("/groups/:id", (&GroupHandler{}).GetByID)
	e.POST("/announcements/:id/read", (&SettingsHandler{}).MarkAnnouncementRead)

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/tasks/" + uuid.New().String()},
		{http.MethodGet, "/accounts/" + uuid.New().String()},
		{http.MethodGet, "/groups/" + uuid.New().String()},
		{http.MethodPost, "/announcements/" + uuid.New().String() + "/read"},
	} {
		rec := doParam(e, tc.method, tc.path)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: status = %d, want 401", tc.method, tc.path, rec.Code)
		}
	}
}
