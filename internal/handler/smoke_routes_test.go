package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"tg-sender/internal/domain"
	"tg-sender/internal/domain/account"
	"tg-sender/internal/middleware"
	"tg-sender/internal/probe"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// newSmokeEcho 装配一个带租户上下文的 echo，并对 panic 做兜底，
// 这样"某条路由打到 nil 依赖"会体现为一个失败用例而不是整包崩掉。
func newSmokeEcho(register func(e *echo.Echo)) *echo.Echo {
	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(middleware.TenantContextKey, uuid.New().String())
			return next(c)
		}
	})
	e.Use(echo.MiddlewareFunc(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = echo.NewHTTPError(http.StatusInternalServerError, "panic")
				}
			}()
			return next(c)
		}
	}))
	register(e)
	return e
}

func hit(e *echo.Echo, method, path, body string) int {
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, nil)
		_ = body
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec.Code
}

// smokeAccountSvc 在零值假对象之上补几个"必须返回非 nil 结构"的方法，
// 否则 handler 解引用空指针会被 recover 成 500，掩盖主干是否走通。
type smokeAccountSvc struct{ zeroAccountSvc }

func (s *smokeAccountSvc) ExportTransferableGroups(context.Context, uuid.UUID, uuid.UUID) (*account.TransferGroupsExport, error) {
	return &account.TransferGroupsExport{}, nil
}

func (s *smokeAccountSvc) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.Account, error) {
	return &domain.Account{AccountID: id, TenantID: tenantID, Status: "online"}, nil
}

func (s *smokeAccountSvc) GetProbeResult(context.Context, uuid.UUID, uuid.UUID) (*probe.Result, error) {
	return &probe.Result{}, nil
}

func (s *smokeAccountSvc) Probe(context.Context, uuid.UUID, uuid.UUID) (*probe.Result, error) {
	return &probe.Result{}, nil
}

type smokeTaskSvc struct{ zeroTaskSvc }

func (s *smokeTaskSvc) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.Task, error) {
	return &domain.Task{TaskID: id, TenantID: tenantID, Status: "queued"}, nil
}

func (s *smokeTaskSvc) CreateJoinGroupsWithInterval(_ context.Context, tenantID, accountID uuid.UUID, links []string, intervalMin, intervalMax int) (*domain.Task, error) {
	return &domain.Task{TaskID: uuid.New(), TenantID: tenantID, Status: "queued"}, nil
}

func (s *smokeTaskSvc) CreateBroadcast(_ context.Context, tenantID uuid.UUID, _ []uuid.UUID, _ []string, _ string, _ *time.Time, _, _ int, _ map[string]any) (*domain.Task, error) {
	return &domain.Task{TaskID: uuid.New(), TenantID: tenantID, Status: "queued"}, nil
}

func (s *smokeTaskSvc) CreateBroadcastWithMapping(_ context.Context, tenantID uuid.UUID, _ []domain.AccountTargetMapping, _ []uuid.UUID, _ []string, _ string, _ *time.Time, _, _ int, _ map[string]any) (*domain.Task, error) {
	return &domain.Task{TaskID: uuid.New(), TenantID: tenantID, Status: "queued"}, nil
}

func (s *smokeTaskSvc) CreateScrapeMembers(_ context.Context, tenantID, _, _ uuid.UUID, _ map[string]any) (*domain.Task, error) {
	return &domain.Task{TaskID: uuid.New(), TenantID: tenantID, Status: "queued"}, nil
}

func (s *smokeTaskSvc) CreateScrapeLinks(_ context.Context, tenantID, _ uuid.UUID, _ []string, _ int) (*domain.Task, error) {
	return &domain.Task{TaskID: uuid.New(), TenantID: tenantID, Status: "queued"}, nil
}

func (s *smokeTaskSvc) CreateJoinGroups(_ context.Context, tenantID, _ uuid.UUID, _ []string) (*domain.Task, error) {
	return &domain.Task{TaskID: uuid.New(), TenantID: tenantID, Status: "queued"}, nil
}

func (s *smokeTaskSvc) ListScrapedLinks(context.Context, uuid.UUID, uuid.UUID, time.Time, int, int) ([]*domain.ScrapedLink, int, error) {
	return []*domain.ScrapedLink{}, 0, nil
}

// 账号相关路由：用零值假 service 走通"解析参数 → 调 service → 组织响应"的主干。
func TestAccountRoutesSmoke(t *testing.T) {
	id := uuid.New().String()
	h := &AccountHandler{svc: &smokeAccountSvc{}, taskSvc: &smokeTaskSvc{}}
	e := newSmokeEcho(func(e *echo.Echo) {
		e.GET("/accounts", h.List)
		e.GET("/accounts/stats", h.Stats)
		e.GET("/accounts/:id", h.GetByID)
		e.PUT("/accounts/:id", h.Update)
		e.POST("/accounts/:id/connect", h.Connect)
		e.POST("/accounts/:id/disconnect", h.Disconnect)
		e.POST("/accounts/:id/reconnect", h.Reconnect)
		e.POST("/accounts/:id/reauthenticate", h.Reauthenticate)
		e.POST("/accounts/:id/code", h.SubmitCode)
		e.POST("/accounts/:id/password", h.Submit2FA)
		e.PUT("/accounts/:id/2fa", h.Update2FA)
		e.POST("/accounts/:id/avatar/refresh", h.RefreshAvatar)
		e.PUT("/accounts/:id/pool", h.SetPool)
		e.PUT("/accounts/:id/proxy", h.SetProxy)
		e.POST("/accounts/:id/proxy/test", h.TestProxy)
		e.POST("/accounts/:id/migrate", h.MigrateDC)
		e.POST("/accounts/:id/probe", h.RunProbe)
		e.GET("/accounts/:id/probe", h.GetProbe)
		e.GET("/accounts/:id/quotas", h.GetQuotas)
		e.GET("/accounts/:id/telegram-dialogs", h.GetTelegramDialogs)
		e.GET("/accounts/:id/telegram-contacts", h.GetTelegramContacts)
		e.POST("/accounts/:id/resolve-peer", h.ResolvePeer)
		e.POST("/accounts/:id/transfer-groups", h.TransferGroups)
		e.GET("/accounts/:id/auth-state", h.GetAuthState)
		e.PUT("/accounts/:id/raffle", h.SetRaffle)
	})

	cases := []struct{ method, path string }{
		{http.MethodGet, "/accounts"},
		{http.MethodGet, "/accounts/stats"},
		{http.MethodGet, "/accounts/" + id},
		{http.MethodPut, "/accounts/" + id},
		{http.MethodPost, "/accounts/" + id + "/connect"},
		{http.MethodPost, "/accounts/" + id + "/disconnect"},
		{http.MethodPost, "/accounts/" + id + "/reconnect"},
		{http.MethodPost, "/accounts/" + id + "/reauthenticate"},
		{http.MethodPost, "/accounts/" + id + "/code"},
		{http.MethodPost, "/accounts/" + id + "/password"},
		{http.MethodPut, "/accounts/" + id + "/2fa"},
		{http.MethodPost, "/accounts/" + id + "/avatar/refresh"},
		{http.MethodPut, "/accounts/" + id + "/pool"},
		{http.MethodPut, "/accounts/" + id + "/proxy"},
		{http.MethodPost, "/accounts/" + id + "/proxy/test"},
		{http.MethodPost, "/accounts/" + id + "/migrate"},
		{http.MethodPost, "/accounts/" + id + "/probe"},
		{http.MethodGet, "/accounts/" + id + "/probe"},
		{http.MethodGet, "/accounts/" + id + "/quotas"},
		{http.MethodGet, "/accounts/" + id + "/telegram-dialogs"},
		{http.MethodGet, "/accounts/" + id + "/telegram-contacts"},
		{http.MethodPost, "/accounts/" + id + "/resolve-peer"},
		{http.MethodPost, "/accounts/" + id + "/transfer-groups"},
		{http.MethodGet, "/accounts/" + id + "/auth-state"},
		{http.MethodPut, "/accounts/" + id + "/raffle"},
	}
	for _, tc := range cases {
		if code := hit(e, tc.method, tc.path, ""); code >= http.StatusInternalServerError {
			t.Fatalf("%s %s: status %d（主干路径不应 5xx）", tc.method, tc.path, code)
		}
	}
}

// 任务相关路由：同上。
func TestTaskRoutesSmoke(t *testing.T) {
	id := uuid.New().String()
	h := &TaskHandler{svc: &smokeTaskSvc{}}
	e := newSmokeEcho(func(e *echo.Echo) {
		e.GET("/tasks", h.List)
		e.GET("/tasks/:id", h.GetByID)
		e.PUT("/tasks/:id", h.Update)
		e.POST("/tasks/:id/pause", h.Pause)
		e.POST("/tasks/:id/resume", h.Resume)
		e.POST("/tasks/:id/cancel", h.Cancel)
		e.POST("/tasks/:id/retry", h.Retry)
		e.DELETE("/tasks/:id", h.Delete)
		e.GET("/tasks/:id/logs", h.GetLogs)
		e.DELETE("/tasks/:id/logs", h.DeleteTaskLogs)
		e.GET("/tasks/:id/stats-by-account", h.GetStatsByAccount)
		e.GET("/tasks/logs", h.GetAccountLogs)
		e.POST("/tasks/batch/cancel", h.BatchCancel)
		e.POST("/tasks/batch/delete", h.BatchDelete)
		e.GET("/scrape-links", h.ListScrapedLinks)
		e.POST("/scrape-links/toggle", h.ToggleLinkScraping)
		e.GET("/scrape-links/status", h.LinkScrapingStatus)
		e.GET("/raffle/results", h.RaffleResults)
		e.DELETE("/raffle/results", h.RaffleResultsClear)
		e.POST("/tasks/validate-targets", h.ValidateTargets)
	})

	cases := []struct{ method, path string }{
		{http.MethodGet, "/tasks"},
		{http.MethodGet, "/tasks/" + id},
		{http.MethodPut, "/tasks/" + id},
		{http.MethodPost, "/tasks/" + id + "/pause"},
		{http.MethodPost, "/tasks/" + id + "/resume"},
		{http.MethodPost, "/tasks/" + id + "/cancel"},
		{http.MethodPost, "/tasks/" + id + "/retry"},
		{http.MethodDelete, "/tasks/" + id},
		{http.MethodGet, "/tasks/" + id + "/logs"},
		{http.MethodDelete, "/tasks/" + id + "/logs"},
		{http.MethodGet, "/tasks/" + id + "/stats-by-account"},
		{http.MethodGet, "/tasks/logs"},
		{http.MethodPost, "/tasks/batch/cancel"},
		{http.MethodPost, "/tasks/batch/delete"},
		{http.MethodGet, "/scrape-links"},
		{http.MethodPost, "/scrape-links/toggle"},
		{http.MethodGet, "/scrape-links/status"},
		{http.MethodGet, "/raffle/results"},
		{http.MethodDelete, "/raffle/results"},
		{http.MethodPost, "/tasks/validate-targets"},
	}
	for _, tc := range cases {
		if code := hit(e, tc.method, tc.path, ""); code >= http.StatusInternalServerError {
			t.Fatalf("%s %s: status %d（主干路径不应 5xx）", tc.method, tc.path, code)
		}
	}
}

// Create 是任务创建的总入口：按 type 分发到各子 handler，并在分发前做套餐门禁、
// 审计与单/复数账号归一化。这里用一组最小合法请求覆盖每条分支。
func TestTaskCreateDispatch(t *testing.T) {
	acc := uuid.New().String()
	target := uuid.New().String()
	h := &TaskHandler{svc: &smokeTaskSvc{}}
	e := newSmokeEcho(func(e *echo.Echo) { e.POST("/tasks", h.Create) })

	bodies := map[string]string{
		"broadcast":            `{"type":"broadcast","account_ids":["` + acc + `"],"target_ids":["@somegroup"],"message":"hi"}`,
		"scrape_members":       `{"type":"scrape_members","account_id":"` + acc + `","target_ids":["` + target + `"]}`,
		"scrape_links":         `{"type":"scrape_links","account_id":"` + acc + `"}`,
		"join_groups":          `{"type":"join_groups","account_id":"` + acc + `","links":["https://t.me/somegroup"]}`,
		"scrape_members no tg": `{"type":"scrape_members","account_id":"` + acc + `"}`,
		"unknown type":         `{"type":"nonsense"}`,
		"bad json":             `{`,
	}
	for name, body := range bodies {
		req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code >= http.StatusInternalServerError {
			t.Fatalf("%s: status %d（不应 5xx）", name, rec.Code)
		}
	}
}
