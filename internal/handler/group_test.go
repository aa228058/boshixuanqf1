package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"tg-sender/internal/config"
	"tg-sender/internal/domain"
	"tg-sender/internal/middleware"
)

type fakeGroupService struct {
	groupService // 嵌入接口：未覆写的方法在被调用时会 panic，正好提示测试覆盖不足
	lastPage     int
	lastPageSize int
	lastAccount  uuid.UUID
	listErr      error
	getErr       error
	updateErr    error
	countErr     error
	members      []*domain.Member
}

func (f *fakeGroupService) List(_ context.Context, _ uuid.UUID, page, pageSize int) ([]*domain.Group, error) {
	f.lastPage, f.lastPageSize = page, pageSize
	if f.listErr != nil {
		return nil, f.listErr
	}
	return []*domain.Group{{Title: "g1"}}, nil
}

func (f *fakeGroupService) ListByAccount(_ context.Context, _, accountID uuid.UUID, page, pageSize int) ([]*domain.Group, error) {
	f.lastAccount, f.lastPage, f.lastPageSize = accountID, page, pageSize
	return []*domain.Group{{Title: "g2"}}, nil
}

func (f *fakeGroupService) CountByAccount(context.Context, uuid.UUID, uuid.UUID) (int, error) {
	if f.countErr != nil {
		return 0, f.countErr
	}
	return 7, nil
}

func (f *fakeGroupService) GetByID(context.Context, uuid.UUID, uuid.UUID) (*domain.Group, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return &domain.Group{Title: "g1"}, nil
}

func (f *fakeGroupService) Update(context.Context, uuid.UUID, uuid.UUID, *string, *string) error {
	return f.updateErr
}

func (f *fakeGroupService) GetMembers(context.Context, uuid.UUID, uuid.UUID, int, int) ([]*domain.Member, int, error) {
	return f.members, len(f.members), nil
}

func (f *fakeGroupService) ListGlobalMembers(context.Context, uuid.UUID, uuid.UUID) ([]map[string]any, error) {
	return []map[string]any{}, nil
}

func groupHarness(fake *fakeGroupService, withTenant bool) *echo.Echo {
	e := echo.New()
	if withTenant {
		e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				c.Set(middleware.TenantContextKey, uuid.New().String())
				return next(c)
			}
		})
	}
	h := &GroupHandler{cfg: &config.Config{GroupDefaultPageSize: 100}, svc: fake}
	e.GET("/groups", h.List)
	e.GET("/groups/:id", h.GetByID)
	e.PUT("/groups/:id", h.Update)
	e.GET("/groups/:id/members", h.GetMembers)
	e.GET("/members", h.GetMembersGlobal)
	return e
}

func doGroup(e *echo.Echo, method, path, body string) *httptest.ResponseRecorder {
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

func TestGroupEndpointsRequireTenant(t *testing.T) {
	e := groupHarness(&fakeGroupService{}, false)
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/groups"},
		{http.MethodGet, "/groups/" + uuid.New().String()},
		{http.MethodPut, "/groups/" + uuid.New().String()},
		{http.MethodGet, "/members?account_id=" + uuid.New().String()},
	} {
		if rec := doGroup(e, tc.method, tc.path, ""); rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: status = %d, want 401", tc.method, tc.path, rec.Code)
		}
	}
}

func TestGroupListPaginationValidation(t *testing.T) {
	e := groupHarness(&fakeGroupService{}, true)
	if rec := doGroup(e, http.MethodGet, "/groups?page=0", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("page=0: %d", rec.Code)
	}
	if rec := doGroup(e, http.MethodGet, "/groups?page=abc", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("page=abc: %d", rec.Code)
	}
	if rec := doGroup(e, http.MethodGet, "/groups?page_size=0", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("page_size=0: %d", rec.Code)
	}
	if rec := doGroup(e, http.MethodGet, "/groups?account_id=not-a-uuid", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad account_id: %d", rec.Code)
	}
}

func TestGroupListDefaultsAndAccountOverride(t *testing.T) {
	fake := &fakeGroupService{}
	if rec := doGroup(groupHarness(fake, true), http.MethodGet, "/groups", ""); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if fake.lastPage != 1 || fake.lastPageSize != 100 {
		t.Fatalf("page=%d size=%d, want 1/100", fake.lastPage, fake.lastPageSize)
	}

	// 带 account_id 且未传 page_size：走"返回全部"分支（10000）
	acc := uuid.New()
	fake2 := &fakeGroupService{}
	if rec := doGroup(groupHarness(fake2, true), http.MethodGet, "/groups?account_id="+acc.String(), ""); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if fake2.lastPageSize != 10000 || fake2.lastAccount != acc {
		t.Fatalf("size=%d account=%s", fake2.lastPageSize, fake2.lastAccount)
	}
}

func TestGroupListClampsPageSize(t *testing.T) {
	fake := &fakeGroupService{}
	if rec := doGroup(groupHarness(fake, true), http.MethodGet, "/groups?page_size=50000", ""); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if fake.lastPageSize != maxPageSize {
		t.Fatalf("size = %d, want %d", fake.lastPageSize, maxPageSize)
	}
}

func TestGroupGetByIDNotFoundAndErrors(t *testing.T) {
	e := groupHarness(&fakeGroupService{getErr: pgx.ErrNoRows}, true)
	if rec := doGroup(e, http.MethodGet, "/groups/"+uuid.New().String(), ""); rec.Code != http.StatusNotFound {
		t.Fatalf("ErrNoRows status = %d, want 404", rec.Code)
	}
	e2 := groupHarness(&fakeGroupService{getErr: errors.New("db down")}, true)
	if rec := doGroup(e2, http.MethodGet, "/groups/"+uuid.New().String(), ""); rec.Code != http.StatusInternalServerError {
		t.Fatalf("generic error status = %d, want 500", rec.Code)
	}
	e3 := groupHarness(&fakeGroupService{}, true)
	if rec := doGroup(e3, http.MethodGet, "/groups/not-a-uuid", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad id status = %d, want 400", rec.Code)
	}
	if rec := doGroup(groupHarness(&fakeGroupService{}, true), http.MethodGet, "/groups/"+uuid.New().String(), ""); rec.Code != http.StatusOK {
		t.Fatalf("ok status = %d", rec.Code)
	}
}

func TestGroupUpdate(t *testing.T) {
	id := uuid.New().String()
	if rec := doGroup(groupHarness(&fakeGroupService{}, true), http.MethodPut, "/groups/"+id, `{"title":"x"}`); rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if rec := doGroup(groupHarness(&fakeGroupService{}, true), http.MethodPut, "/groups/bad", `{}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad id status = %d", rec.Code)
	}
	if rec := doGroup(groupHarness(&fakeGroupService{}, true), http.MethodPut, "/groups/"+id, `{bad json`); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad body status = %d", rec.Code)
	}
	if rec := doGroup(groupHarness(&fakeGroupService{updateErr: errors.New("x")}, true), http.MethodPut, "/groups/"+id, `{}`); rec.Code != http.StatusInternalServerError {
		t.Fatalf("update error status = %d", rec.Code)
	}
}

func TestGroupMembersGlobalRequiresAccountID(t *testing.T) {
	if rec := doGroup(groupHarness(&fakeGroupService{}, true), http.MethodGet, "/members", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
