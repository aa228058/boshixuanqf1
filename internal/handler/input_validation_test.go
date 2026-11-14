package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tg-sender/internal/domain"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// 坏输入必须被挡在 handler 层（4xx），不能穿透到 service 变成 5xx。
func assertClientError(t *testing.T, e *echo.Echo, method, path, body string) {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code >= http.StatusInternalServerError {
		t.Fatalf("%s %s body=%q: status %d，坏输入不应落到 5xx", method, path, body, rec.Code)
	}
}

// safeAdmin 在 fakeAdmin 之上补齐写接口用例需要的方法（否则 nil 接口会 panic）。
type safeAdmin struct{ fakeAdmin }

func (s *safeAdmin) GenerateCodes(context.Context, uuid.UUID, string, int, string, *int, string) ([]domain.ActivationCode, error) {
	return []domain.ActivationCode{}, nil
}

func (s *safeAdmin) CreateTenant(context.Context, string, string, string, string) (*domain.Tenant, string, error) {
	return &domain.Tenant{TenantID: uuid.New()}, "key", nil
}

func (s *safeAdmin) UpdatePlan(context.Context, string, map[string]any) error { return nil }
func (s *safeAdmin) BatchRevokeCodes(context.Context, []uuid.UUID) error      { return nil }
func (s *safeAdmin) RevokeCode(context.Context, uuid.UUID) error              { return nil }

func TestAccountWriteEndpointsRejectBadBodies(t *testing.T) {
	h := &AccountHandler{svc: &smokeAccountSvc{}, taskSvc: &smokeTaskSvc{}}
	e := newSmokeEcho(func(e *echo.Echo) {
		e.POST("/accounts", h.Create)
		e.PUT("/accounts/:id", h.Update)
		e.POST("/accounts/pool-batch", h.BatchSetPool)
		e.POST("/accounts/profile-batch", h.BatchUpdateProfile)
		e.POST("/accounts/clone-profile", h.CloneProfile)
		e.POST("/accounts/:id/code", h.SubmitCode)
		e.PUT("/accounts/:id/2fa", h.Update2FA)
		e.POST("/accounts/:id/resolve-peer", h.ResolvePeer)
	})
	id := uuid.New().String()
	cases := []struct{ method, path, body string }{
		{http.MethodPost, "/accounts", `{`},
		{http.MethodPut, "/accounts/" + id, `{`},
		{http.MethodPost, "/accounts/pool-batch", `{`},
		{http.MethodPost, "/accounts/profile-batch", `{`},
		{http.MethodPost, "/accounts/clone-profile", `{`},
		{http.MethodPost, "/accounts/" + id + "/code", `{`},
		{http.MethodPut, "/accounts/" + id + "/2fa", `{`},
		{http.MethodPost, "/accounts/" + id + "/resolve-peer", `{`},
		{http.MethodPut, "/accounts/bad-id", `{}`},
	}
	for _, tc := range cases {
		assertClientError(t, e, tc.method, tc.path, tc.body)
	}
}

func TestTaskQueryEndpointsTolerateBadParams(t *testing.T) {
	h := &TaskHandler{svc: &smokeTaskSvc{}}
	e := newSmokeEcho(func(e *echo.Echo) {
		e.GET("/tasks", h.List)
		e.GET("/tasks/:id/logs", h.GetLogs)
		e.GET("/tasks/logs", h.GetAccountLogs)
		e.GET("/tasks/:id/stats-by-account", h.GetStatsByAccount)
		e.DELETE("/tasks/:id/logs", h.DeleteTaskLogs)
		e.GET("/scrape-links", h.ListScrapedLinks)
	})
	id := uuid.New().String()
	cases := []struct{ method, path string }{
		{http.MethodGet, "/tasks?page=abc&page_size=xyz&status=bogus"},
		{http.MethodGet, "/tasks/" + id + "/logs?page=0"},
		{http.MethodGet, "/tasks/logs?account_id=not-a-uuid"},
		{http.MethodGet, "/tasks/" + id + "/stats-by-account"},
		{http.MethodDelete, "/tasks/bad-id/logs"},
		{http.MethodGet, "/scrape-links?account_id=not-a-uuid"},
	}
	for _, tc := range cases {
		assertClientError(t, e, tc.method, tc.path, "")
	}
}

func TestAdminWriteEndpointsRejectBadBodies(t *testing.T) {
	h := &AdminHandler{adminService: &safeAdmin{}}
	e := echo.New()
	e.POST("/admin/codes", h.GenerateCodes)
	e.POST("/admin/tenants", h.CreateTenant)
	e.PUT("/admin/plans/:id", h.UpdatePlan)
	e.POST("/admin/codes/batch-revoke", h.BatchRevokeCodes)
	e.DELETE("/admin/codes/:id", h.RevokeCode)
	cases := []struct{ method, path, body string }{
		{http.MethodPost, "/admin/codes", `{`},
		{http.MethodPost, "/admin/tenants", `{`},
		{http.MethodPut, "/admin/plans/basic", `{`},
		{http.MethodPost, "/admin/codes/batch-revoke", `{`},
		{http.MethodDelete, "/admin/codes/bad-id", ``},
	}
	for _, tc := range cases {
		assertClientError(t, e, tc.method, tc.path, tc.body)
	}
}

// 任务查询参数的各种组合：覆盖分页/时间窗/状态过滤的解析分支。
func TestTaskQueryVariants(t *testing.T) {
	h := &TaskHandler{svc: &smokeTaskSvc{}}
	e := newSmokeEcho(func(e *echo.Echo) {
		e.GET("/tasks", h.List)
		e.GET("/tasks/:id/logs", h.GetLogs)
		e.GET("/tasks/logs", h.GetAccountLogs)
		e.GET("/scrape-links", h.ListScrapedLinks)
		e.GET("/scrape-links/status", h.LinkScrapingStatus)
		e.POST("/scrape-links/toggle", h.ToggleLinkScraping)
	})
	acc := uuid.New().String()
	tid := uuid.New().String()
	cases := []struct{ method, path, body string }{
		{http.MethodGet, "/tasks?status=completed&type=broadcast&hide_loops=1&page=2&page_size=10", ""},
		{http.MethodGet, "/tasks?since=2026-01-01T00:00:00Z&account_id=" + acc, ""},
		{http.MethodGet, "/tasks/" + tid + "/logs?page=2&page_size=50", ""},
		{http.MethodGet, "/tasks/" + tid + "/logs?from=2026-01-01T00:00:00Z&to=2026-01-02T00:00:00Z", ""},
		{http.MethodGet, "/tasks/logs?account_id=" + acc + "&page=1&page_size=20", ""},
		{http.MethodGet, "/scrape-links?account_id=" + acc + "&page=1&page_size=10", ""},
		{http.MethodGet, "/scrape-links/status?account_id=" + acc, ""},
		{http.MethodPost, "/scrape-links/toggle", `{"account_id":"` + acc + `","enabled":true}`},
		{http.MethodPost, "/scrape-links/toggle", `{"account_id":"` + acc + `","mode":"bio","enabled":false}`},
	}
	for _, tc := range cases {
		assertClientError(t, e, tc.method, tc.path, tc.body)
	}
}

// 带 Authorization 头但内容畸形的登出/刷新，必须走拒绝分支而不是崩。
func TestAuthEndpointsTolerateMalformedToken(t *testing.T) {
	e, _ := authHarness()
	for _, hdr := range []string{"Bearer", "Bearer not-a-jwt", "Basic abc", "bearer " + strings.Repeat("x", 40)} {
		req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
		req.Header.Set(echo.HeaderAuthorization, hdr)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code >= http.StatusInternalServerError {
			t.Fatalf("logout header=%q: status %d", hdr, rec.Code)
		}
	}
	if rec := postJSON(e, "/auth/refresh", `{"refresh_token":"eyJhbGciOiJIUzI1NiJ9.bad.payload"}`); rec.Code >= http.StatusInternalServerError {
		t.Fatalf("refresh 畸形 token: status %d", rec.Code)
	}
}
