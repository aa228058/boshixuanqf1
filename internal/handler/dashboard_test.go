package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"tg-sender/internal/domain"
	"tg-sender/internal/middleware"
)

type fakeDashboard struct {
	statsCalled     bool
	statsForAccount bool
	lastAccountID   uuid.UUID
	lastLimit       int
	lastOffset      int
	lastMonths      int
	statsErr        error
	calendarErr     error
	riskErr         error
	auditErr        error
}

func (f *fakeDashboard) Stats(context.Context, uuid.UUID) (map[string]any, error) {
	f.statsCalled = true
	if f.statsErr != nil {
		return nil, f.statsErr
	}
	return map[string]any{"accounts": 3}, nil
}

func (f *fakeDashboard) StatsForAccount(_ context.Context, _, accountID uuid.UUID) (map[string]any, error) {
	f.statsForAccount = true
	f.lastAccountID = accountID
	return map[string]any{"accounts": 1}, nil
}

func (f *fakeDashboard) SendCalendar(_ context.Context, _ uuid.UUID, months int) (map[string]int, error) {
	f.lastMonths = months
	if f.calendarErr != nil {
		return nil, f.calendarErr
	}
	return map[string]int{"2026-01-01": 5}, nil
}

func (f *fakeDashboard) AuditLogs(_ context.Context, _ uuid.UUID, limit, offset int, _, _ string) ([]*domain.AuditLog, error) {
	f.lastLimit, f.lastOffset = limit, offset
	if f.auditErr != nil {
		return nil, f.auditErr
	}
	return []*domain.AuditLog{{Action: "x"}}, nil
}

func (f *fakeDashboard) RiskEvents(_ context.Context, _ uuid.UUID, limit, offset int, _, _ string) ([]*domain.RiskEvent, error) {
	f.lastLimit, f.lastOffset = limit, offset
	if f.riskErr != nil {
		return nil, f.riskErr
	}
	return []*domain.RiskEvent{}, nil
}

func dashHarness(fake *fakeDashboard, withTenant bool) *echo.Echo {
	e := echo.New()
	if withTenant {
		e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				c.Set(middleware.TenantContextKey, uuid.New().String())
				return next(c)
			}
		})
	}
	h := &DashboardHandler{svc: fake}
	e.GET("/dashboard/stats", h.Stats)
	e.GET("/dashboard/send-calendar", h.SendCalendar)
	e.GET("/dashboard/risk-events", h.RiskEvents)
	e.GET("/dashboard/activities", h.AuditLogs)
	return e
}

func callDash(e *echo.Echo, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestDashboardRequiresTenant(t *testing.T) {
	e := dashHarness(&fakeDashboard{}, false)
	for _, p := range []string{"/dashboard/stats", "/dashboard/send-calendar", "/dashboard/risk-events", "/dashboard/activities"} {
		if rec := callDash(e, p); rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s: status = %d, want 401", p, rec.Code)
		}
	}
}

func TestDashboardStatsAggregated(t *testing.T) {
	fake := &fakeDashboard{}
	rec := callDash(dashHarness(fake, true), "/dashboard/stats")
	if rec.Code != http.StatusOK || !fake.statsCalled || fake.statsForAccount {
		t.Fatalf("status=%d statsCalled=%v perAccount=%v", rec.Code, fake.statsCalled, fake.statsForAccount)
	}
}

func TestDashboardStatsPerAccount(t *testing.T) {
	fake := &fakeDashboard{}
	acc := uuid.New()
	rec := callDash(dashHarness(fake, true), "/dashboard/stats?account_id="+acc.String())
	if rec.Code != http.StatusOK || !fake.statsForAccount || fake.lastAccountID != acc {
		t.Fatalf("status=%d perAccount=%v id=%s", rec.Code, fake.statsForAccount, fake.lastAccountID)
	}
}

func TestDashboardStatsRejectsInvalidAccountID(t *testing.T) {
	rec := callDash(dashHarness(&fakeDashboard{}, true), "/dashboard/stats?account_id=abc")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestDashboardStatsInternalError(t *testing.T) {
	rec := callDash(dashHarness(&fakeDashboard{statsErr: errors.New("boom")}, true), "/dashboard/stats")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestDashboardSendCalendarPassesMonths(t *testing.T) {
	fake := &fakeDashboard{}
	if rec := callDash(dashHarness(fake, true), "/dashboard/send-calendar?months=6"); rec.Code != http.StatusOK || fake.lastMonths != 6 {
		t.Fatalf("status=%d months=%d", rec.Code, fake.lastMonths)
	}
	if rec := callDash(dashHarness(&fakeDashboard{calendarErr: errors.New("x")}, true), "/dashboard/send-calendar"); rec.Code != http.StatusInternalServerError {
		t.Fatalf("error case status = %d, want 500", rec.Code)
	}
}

func TestDashboardRiskEventsClampsPagination(t *testing.T) {
	fake := &fakeDashboard{}
	// limit 超上限应退回默认 50；offset 超上限夹到 100000
	rec := callDash(dashHarness(fake, true), "/dashboard/risk-events?limit=999&offset=99999999")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if fake.lastLimit != 50 || fake.lastOffset != 100000 {
		t.Fatalf("limit=%d offset=%d, want 50/100000", fake.lastLimit, fake.lastOffset)
	}
}

func TestDashboardRiskEventsInvalidTimeIs400(t *testing.T) {
	rec := callDash(dashHarness(&fakeDashboard{riskErr: errors.New("invalid time format: bad")}, true), "/dashboard/risk-events")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestDashboardAuditLogsLimitAndError(t *testing.T) {
	fake := &fakeDashboard{}
	if rec := callDash(dashHarness(fake, true), "/dashboard/activities?limit=10"); rec.Code != http.StatusOK || fake.lastLimit != 10 {
		t.Fatalf("status=%d limit=%d", rec.Code, fake.lastLimit)
	}
	rec := callDash(dashHarness(&fakeDashboard{auditErr: errors.New("invalid time format: x")}, true), "/dashboard/activities")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
