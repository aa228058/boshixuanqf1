package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"tg-sender/internal/domain"
)

type fakeAdmin struct {
	adminPlaneService // 嵌入接口：未覆写的方法被调用会 panic，提示测试缺口
	tenants           []*domain.Tenant
	total             int
	lastPage          int
	lastPageSize      int
	lastSearch        string
	listErr           error
	getErr            error
	statsErr          error
	lastRetention     int
}

func (f *fakeAdmin) ListTenants(_ context.Context, page, pageSize int, _, _, search string) ([]*domain.Tenant, int, error) {
	f.lastPage, f.lastPageSize, f.lastSearch = page, pageSize, search
	if f.listErr != nil {
		return nil, 0, f.listErr
	}
	return f.tenants, f.total, nil
}

func (f *fakeAdmin) GetTenant(_ context.Context, id uuid.UUID) (*domain.Tenant, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return &domain.Tenant{TenantID: id, PlanID: "basic"}, nil
}

func (f *fakeAdmin) GetStats(context.Context) (map[string]any, error) {
	if f.statsErr != nil {
		return nil, f.statsErr
	}
	return map[string]any{"tenants": 3}, nil
}

func (f *fakeAdmin) GetSystemMetrics(context.Context) (map[string]any, error) {
	return map[string]any{"goroutines": 42}, nil
}

func (f *fakeAdmin) ListPlans(context.Context) ([]domain.Plan, error) {
	return []domain.Plan{{PlanID: "basic"}}, nil
}

func (f *fakeAdmin) ListCodes(_ context.Context, _, _ string, page, pageSize int) ([]domain.ActivationCode, int, error) {
	f.lastPage, f.lastPageSize = page, pageSize
	return []domain.ActivationCode{}, 0, nil
}

func (f *fakeAdmin) ListAnnouncements(context.Context) ([]domain.Announcement, error) {
	return []domain.Announcement{{Title: "a"}}, nil
}

func (f *fakeAdmin) CreateAnnouncement(context.Context, *domain.Announcement) error { return nil }

func (f *fakeAdmin) GetAuditLogs(_ context.Context, page, pageSize int, _, _ string) ([]*domain.AuditLog, int, error) {
	f.lastPage, f.lastPageSize = page, pageSize
	return []*domain.AuditLog{}, 0, nil
}

func (f *fakeAdmin) GetSupport(context.Context) (*domain.SupportConfig, error) {
	return &domain.SupportConfig{}, nil
}

func (f *fakeAdmin) PurgeExpiredTenants(_ context.Context, retentionDays int) (int, error) {
	f.lastRetention = retentionDays
	return 2, nil
}

func adminHarness(fake *fakeAdmin) *echo.Echo {
	e := echo.New()
	h := &AdminHandler{adminService: fake}
	e.GET("/admin/stats", h.GetStats)
	e.GET("/admin/system", h.GetSystemMetrics)
	e.GET("/admin/tenants", h.ListTenants)
	e.GET("/admin/tenants/:id", h.GetTenant)
	e.GET("/admin/plans", h.ListPlans)
	e.GET("/admin/codes", h.ListCodes)
	e.GET("/admin/announcements", h.ListAnnouncements)
	e.POST("/admin/announcements", h.CreateAnnouncement)
	e.GET("/admin/audit-logs", h.GetAuditLogs)
	e.GET("/admin/support", h.GetSupport)
	e.POST("/admin/tenants/purge-expired", h.PurgeExpiredTenants)
	return e
}

func doAdmin(e *echo.Echo, method, path, body string) *httptest.ResponseRecorder {
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestAdminListTenantsDefaultsAndFilters(t *testing.T) {
	fake := &fakeAdmin{total: 3}
	rec := doAdmin(adminHarness(fake), http.MethodGet, "/admin/tenants?search=%20acme%20", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if fake.lastPage != 1 || fake.lastPageSize != defaultPageSize {
		t.Fatalf("page=%d size=%d, want 1/%d", fake.lastPage, fake.lastPageSize, defaultPageSize)
	}
	if fake.lastSearch != "acme" {
		t.Fatalf("search 未去空格: %q", fake.lastSearch)
	}
}

func TestAdminListTenantsClampsPageSize(t *testing.T) {
	fake := &fakeAdmin{}
	if rec := doAdmin(adminHarness(fake), http.MethodGet, "/admin/tenants?page_size=999999", ""); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if fake.lastPageSize != maxPageSize {
		t.Fatalf("pageSize = %d, want %d", fake.lastPageSize, maxPageSize)
	}
}

func TestAdminListTenantsError(t *testing.T) {
	rec := doAdmin(adminHarness(&fakeAdmin{listErr: errors.New("db")}), http.MethodGet, "/admin/tenants", "")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestAdminGetTenant(t *testing.T) {
	id := uuid.New()
	if rec := doAdmin(adminHarness(&fakeAdmin{}), http.MethodGet, "/admin/tenants/"+id.String(), ""); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec := doAdmin(adminHarness(&fakeAdmin{}), http.MethodGet, "/admin/tenants/bad", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad id status = %d", rec.Code)
	}
	if rec := doAdmin(adminHarness(&fakeAdmin{getErr: errors.New("not found")}), http.MethodGet, "/admin/tenants/"+id.String(), ""); rec.Code != http.StatusNotFound {
		t.Fatalf("missing tenant status = %d, want 404", rec.Code)
	}
}

func TestAdminStatsError(t *testing.T) {
	if rec := doAdmin(adminHarness(&fakeAdmin{}), http.MethodGet, "/admin/stats", ""); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec := doAdmin(adminHarness(&fakeAdmin{statsErr: errors.New("x")}), http.MethodGet, "/admin/stats", ""); rec.Code != http.StatusInternalServerError {
		t.Fatalf("error status = %d, want 500", rec.Code)
	}
}

func TestAdminReadOnlyEndpoints(t *testing.T) {
	e := adminHarness(&fakeAdmin{})
	for _, p := range []string{"/admin/system", "/admin/plans", "/admin/codes", "/admin/announcements", "/admin/audit-logs", "/admin/support"} {
		if rec := doAdmin(e, http.MethodGet, p, ""); rec.Code != http.StatusOK {
			t.Fatalf("%s: status = %d", p, rec.Code)
		}
	}
}

func TestAdminPurgeExpiredPassesRetention(t *testing.T) {
	fake := &fakeAdmin{}
	if rec := doAdmin(adminHarness(fake), http.MethodPost, "/admin/tenants/purge-expired", `{"retention_days":30}`); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if fake.lastRetention != 30 {
		t.Fatalf("retention = %d, want 30", fake.lastRetention)
	}
}

func TestAdminCreateAnnouncementValidation(t *testing.T) {
	if rec := doAdmin(adminHarness(&fakeAdmin{}), http.MethodPost, "/admin/announcements", `{bad`); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad body status = %d, want 400", rec.Code)
	}
	if rec := doAdmin(adminHarness(&fakeAdmin{}), http.MethodPost, "/admin/announcements", `{"title":"t","content":"c"}`); rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d", rec.Code)
	}
}
