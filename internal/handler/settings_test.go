package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"tg-sender/internal/domain"
	"tg-sender/internal/middleware"
)

type fakeSettings struct {
	settingsService // 嵌入接口：未覆写的方法在被调用时会 panic，提示测试未覆盖到
	profileErr      error
	plans           []domain.Plan
	templates       []*domain.MessageTemplate
	announcements   []domain.Announcement
	accountGroups   []*domain.AccountGroup
	decoration      bool
	tenant          *domain.Tenant
	lastSMTP        json.RawMessage
	markedRead      uuid.UUID
}

func (f *fakeSettings) GetProfile(context.Context, uuid.UUID) (*domain.Tenant, error) {
	if f.profileErr != nil {
		return nil, f.profileErr
	}
	if f.tenant != nil {
		return f.tenant, nil
	}
	return &domain.Tenant{TenantID: uuid.New(), PlanID: "basic", Status: "active"}, nil
}

func (f *fakeSettings) GetPlans(context.Context) ([]domain.Plan, error) {
	if f.plans != nil {
		return f.plans, nil
	}
	return []domain.Plan{{PlanID: "basic", Name: "基础版"}}, nil
}

func (f *fakeSettings) GetAnnouncements(context.Context, uuid.UUID, string) ([]domain.Announcement, error) {
	return f.announcements, nil
}

func (f *fakeSettings) MarkAnnouncementRead(_ context.Context, _, id uuid.UUID) error {
	f.markedRead = id
	return nil
}

func (f *fakeSettings) ListTemplates(context.Context, uuid.UUID) ([]*domain.MessageTemplate, error) {
	return f.templates, nil
}

func (f *fakeSettings) CreateTemplate(_ context.Context, tenantID uuid.UUID, name, content string) (*domain.MessageTemplate, error) {
	return &domain.MessageTemplate{TemplateID: uuid.New(), TenantID: tenantID, Name: name, Content: content}, nil
}

func (f *fakeSettings) UpdateTemplate(context.Context, uuid.UUID, uuid.UUID, string, string) error {
	return nil
}
func (f *fakeSettings) DeleteTemplate(context.Context, uuid.UUID, uuid.UUID) error { return nil }

func (f *fakeSettings) ListAccountGroups(context.Context, uuid.UUID) ([]*domain.AccountGroup, error) {
	return f.accountGroups, nil
}

func (f *fakeSettings) CreateAccountGroup(_ context.Context, tenantID uuid.UUID, name, color string, ids []uuid.UUID) (*domain.AccountGroup, error) {
	return &domain.AccountGroup{Name: name, Color: color}, nil
}

func (f *fakeSettings) UpdateAccountGroup(context.Context, uuid.UUID, uuid.UUID, string, string, []uuid.UUID) error {
	return nil
}
func (f *fakeSettings) DeleteAccountGroup(context.Context, uuid.UUID, uuid.UUID) error { return nil }

func (f *fakeSettings) GetMessageDecorationEnabled(context.Context, uuid.UUID) (bool, error) {
	return f.decoration, nil
}

func (f *fakeSettings) UpdateMessageDecorationEnabled(_ context.Context, _ uuid.UUID, enabled bool) error {
	f.decoration = enabled
	return nil
}

func (f *fakeSettings) GetSupport(context.Context) (*domain.SupportConfig, error) {
	return &domain.SupportConfig{}, nil
}

func (f *fakeSettings) UpdateNotify(_ context.Context, _ uuid.UUID, _ string, _, smtp json.RawMessage) error {
	f.lastSMTP = smtp
	return nil
}

func settingsHarness(fake *fakeSettings, withTenant bool) *echo.Echo {
	e := echo.New()
	if withTenant {
		e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				c.Set(middleware.TenantContextKey, uuid.New().String())
				return next(c)
			}
		})
	}
	h := &SettingsHandler{svc: fake}
	e.GET("/tenant/profile", h.GetProfile)
	e.GET("/plans", h.GetPlans)
	e.GET("/announcements", h.GetAnnouncements)
	e.POST("/announcements/:id/read", h.MarkAnnouncementRead)
	e.GET("/templates", h.ListTemplates)
	e.POST("/templates", h.CreateTemplate)
	e.PUT("/templates/:id", h.UpdateTemplate)
	e.DELETE("/templates/:id", h.DeleteTemplate)
	e.GET("/account-groups", h.ListAccountGroups)
	e.POST("/account-groups", h.CreateAccountGroup)
	e.PUT("/account-groups/:id", h.UpdateAccountGroup)
	e.DELETE("/account-groups/:id", h.DeleteAccountGroup)
	e.GET("/tenant/message-decoration", h.GetMessageDecoration)
	e.PUT("/tenant/message-decoration", h.UpdateMessageDecoration)
	e.GET("/tenant/notify", h.GetNotify)
	e.PUT("/tenant/notify", h.UpdateNotify)
	e.GET("/support", h.GetSupport)
	return e
}

func doSettings(e *echo.Echo, method, path, body string) *httptest.ResponseRecorder {
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

func TestSettingsReadEndpointsRequireTenant(t *testing.T) {
	e := settingsHarness(&fakeSettings{}, false)
	// 注意：/plans 是套餐价目表，按设计不依赖租户上下文（鉴权由路由中间件负责），
	// 所以这里不列入"缺租户必须 401"的清单。
	for _, p := range []string{"/tenant/profile", "/announcements", "/templates", "/account-groups", "/tenant/message-decoration", "/tenant/notify"} {
		if rec := doSettings(e, http.MethodGet, p, ""); rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s: status = %d, want 401", p, rec.Code)
		}
	}
}

func TestSettingsProfileAndPlans(t *testing.T) {
	e := settingsHarness(&fakeSettings{}, true)
	if rec := doSettings(e, http.MethodGet, "/tenant/profile", ""); rec.Code != http.StatusOK {
		t.Fatalf("profile status = %d", rec.Code)
	}
	if rec := doSettings(e, http.MethodGet, "/plans", ""); rec.Code != http.StatusOK {
		t.Fatalf("plans status = %d", rec.Code)
	}
	e2 := settingsHarness(&fakeSettings{profileErr: errors.New("db down")}, true)
	if rec := doSettings(e2, http.MethodGet, "/tenant/profile", ""); rec.Code != http.StatusInternalServerError {
		t.Fatalf("profile error status = %d, want 500", rec.Code)
	}
}

func TestAnnouncementsListAndRead(t *testing.T) {
	fake := &fakeSettings{announcements: []domain.Announcement{{Title: "a1"}}}
	e := settingsHarness(fake, true)
	if rec := doSettings(e, http.MethodGet, "/announcements", ""); rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}
	id := uuid.New()
	if rec := doSettings(e, http.MethodPost, "/announcements/"+id.String()+"/read", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("read status = %d, want 204", rec.Code)
	}
	if fake.markedRead != id {
		t.Fatalf("未把已读写回仓储: %s", fake.markedRead)
	}
	if rec := doSettings(e, http.MethodPost, "/announcements/not-a-uuid/read", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad id status = %d, want 400", rec.Code)
	}
}

func TestTemplatesCRUD(t *testing.T) {
	e := settingsHarness(&fakeSettings{}, true)
	if rec := doSettings(e, http.MethodGet, "/templates", ""); rec.Code != http.StatusOK {
		t.Fatalf("list = %d", rec.Code)
	}
	if rec := doSettings(e, http.MethodPost, "/templates", `{"name":"t1","content":"hi"}`); rec.Code != http.StatusOK {
		t.Fatalf("create = %d", rec.Code)
	}
	if rec := doSettings(e, http.MethodPost, "/templates", `{bad`); rec.Code != http.StatusBadRequest {
		t.Fatalf("create bad body = %d, want 400", rec.Code)
	}
	id := uuid.New().String()
	if rec := doSettings(e, http.MethodPut, "/templates/"+id, `{"name":"t2","content":"yo"}`); rec.Code != http.StatusNoContent {
		t.Fatalf("update = %d", rec.Code)
	}
	if rec := doSettings(e, http.MethodPut, "/templates/bad", `{}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("update bad id = %d", rec.Code)
	}
	if rec := doSettings(e, http.MethodDelete, "/templates/"+id, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete = %d", rec.Code)
	}
	if rec := doSettings(e, http.MethodDelete, "/templates/bad", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("delete bad id = %d", rec.Code)
	}
}

func TestAccountGroupsCRUD(t *testing.T) {
	e := settingsHarness(&fakeSettings{}, true)
	if rec := doSettings(e, http.MethodGet, "/account-groups", ""); rec.Code != http.StatusOK {
		t.Fatalf("list = %d", rec.Code)
	}
	if rec := doSettings(e, http.MethodPost, "/account-groups", `{"name":"g","color":"#fff","account_ids":[]}`); rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Fatalf("create = %d", rec.Code)
	}
	if rec := doSettings(e, http.MethodPost, "/account-groups", `{bad`); rec.Code != http.StatusBadRequest {
		t.Fatalf("create bad body = %d", rec.Code)
	}
	id := uuid.New().String()
	if rec := doSettings(e, http.MethodDelete, "/account-groups/"+id, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete = %d", rec.Code)
	}
}

func TestMessageDecorationToggle(t *testing.T) {
	fake := &fakeSettings{}
	e := settingsHarness(fake, true)
	if rec := doSettings(e, http.MethodGet, "/tenant/message-decoration", ""); rec.Code != http.StatusOK {
		t.Fatalf("get = %d", rec.Code)
	}
	if rec := doSettings(e, http.MethodPut, "/tenant/message-decoration", `{"enabled":true}`); rec.Code != http.StatusOK || !fake.decoration {
		t.Fatalf("put = %d enabled=%v", rec.Code, fake.decoration)
	}
	if rec := doSettings(e, http.MethodPut, "/tenant/message-decoration", `{}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("put without enabled = %d, want 400", rec.Code)
	}
}

func TestNotifyRequiresEmail(t *testing.T) {
	e := settingsHarness(&fakeSettings{tenant: &domain.Tenant{TenantID: uuid.New()}}, true)
	if rec := doSettings(e, http.MethodGet, "/tenant/notify", ""); rec.Code != http.StatusOK {
		t.Fatalf("get notify = %d", rec.Code)
	}
	if rec := doSettings(e, http.MethodPut, "/tenant/notify", `{"settings":{}}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("put without email = %d, want 400", rec.Code)
	}
}
