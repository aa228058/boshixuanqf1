package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"tg-sender/internal/domain"
	"tg-sender/internal/middleware"
)

type fakeMonitorSvc struct {
	monitorRuleService // 嵌入接口：未覆写的方法被调用会 panic
	rules              []*domain.MonitorRule
	listErr            error
	getErr             error
	deleteErr          error
	deleted            uuid.UUID
}

func (f *fakeMonitorSvc) List(context.Context, uuid.UUID) ([]*domain.MonitorRule, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.rules, nil
}

func (f *fakeMonitorSvc) GetByID(context.Context, uuid.UUID, uuid.UUID) (*domain.MonitorRule, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return &domain.MonitorRule{Name: "r1"}, nil
}

func (f *fakeMonitorSvc) Delete(_ context.Context, _, id uuid.UUID) error {
	f.deleted = id
	return f.deleteErr
}

func monitorHarness(fake *fakeMonitorSvc, withTenant bool) *echo.Echo {
	e := echo.New()
	if withTenant {
		e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				c.Set(middleware.TenantContextKey, uuid.New().String())
				return next(c)
			}
		})
	}
	h := &MonitorHandler{svc: fake}
	e.GET("/monitor/rules", h.List)
	e.GET("/monitor/rules/:id", h.GetByID)
	e.DELETE("/monitor/rules/:id", h.Delete)
	return e
}

func doMonitor(e *echo.Echo, method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestMonitorEndpointsRequireTenant(t *testing.T) {
	e := monitorHarness(&fakeMonitorSvc{}, false)
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/monitor/rules"},
		{http.MethodGet, "/monitor/rules/" + uuid.New().String()},
		{http.MethodDelete, "/monitor/rules/" + uuid.New().String()},
	} {
		if rec := doMonitor(e, tc.method, tc.path); rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: status = %d", tc.method, tc.path, rec.Code)
		}
	}
}

func TestMonitorListAndError(t *testing.T) {
	fake := &fakeMonitorSvc{rules: []*domain.MonitorRule{{Name: "r1"}, {Name: "r2"}}}
	if rec := doMonitor(monitorHarness(fake, true), http.MethodGet, "/monitor/rules"); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec := doMonitor(monitorHarness(&fakeMonitorSvc{listErr: errors.New("x")}, true), http.MethodGet, "/monitor/rules"); rec.Code != http.StatusInternalServerError {
		t.Fatalf("error status = %d, want 500", rec.Code)
	}
}

func TestMonitorGetByID(t *testing.T) {
	id := uuid.New().String()
	if rec := doMonitor(monitorHarness(&fakeMonitorSvc{}, true), http.MethodGet, "/monitor/rules/"+id); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec := doMonitor(monitorHarness(&fakeMonitorSvc{}, true), http.MethodGet, "/monitor/rules/bad"); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad id status = %d", rec.Code)
	}
	if rec := doMonitor(monitorHarness(&fakeMonitorSvc{getErr: pgx.ErrNoRows}, true), http.MethodGet, "/monitor/rules/"+id); rec.Code != http.StatusNotFound {
		t.Fatalf("not found status = %d, want 404", rec.Code)
	}
	if rec := doMonitor(monitorHarness(&fakeMonitorSvc{getErr: errors.New("x")}, true), http.MethodGet, "/monitor/rules/"+id); rec.Code != http.StatusInternalServerError {
		t.Fatalf("error status = %d, want 500", rec.Code)
	}
}

func TestMonitorDelete(t *testing.T) {
	fake := &fakeMonitorSvc{}
	id := uuid.New()
	if rec := doMonitor(monitorHarness(fake, true), http.MethodDelete, "/monitor/rules/"+id.String()); rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if fake.deleted != id {
		t.Fatalf("删除的规则 id = %s, want %s", fake.deleted, id)
	}
	if rec := doMonitor(monitorHarness(&fakeMonitorSvc{}, true), http.MethodDelete, "/monitor/rules/bad"); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad id status = %d", rec.Code)
	}
	if rec := doMonitor(monitorHarness(&fakeMonitorSvc{deleteErr: errors.New("x")}, true), http.MethodDelete, "/monitor/rules/"+id.String()); rec.Code != http.StatusInternalServerError {
		t.Fatalf("error status = %d, want 500", rec.Code)
	}
}
