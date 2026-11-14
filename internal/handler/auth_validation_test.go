package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"tg-sender/internal/config"
)

// 这些用例只覆盖"进入仓储之前"的校验分支：AuthHandler 的 repo 依赖是接口，
// 但这里刻意全部留空——任何用例只要越过校验就会 panic，从而暴露测试写错。
func authHarness() (*echo.Echo, *AuthHandler) {
	h := &AuthHandler{cfg: &config.Config{JWTSecret: strings.Repeat("k", 40)}}
	e := echo.New()
	e.POST("/auth/register", h.Register)
	e.POST("/auth/login", h.Login)
	e.POST("/auth/refresh", h.Refresh)
	e.POST("/auth/logout", h.Logout)
	e.GET("/auth/me", h.Me)
	return e, h
}

func postJSON(e *echo.Echo, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestRegisterValidation(t *testing.T) {
	e, _ := authHarness()
	cases := map[string]struct {
		body string
		want int
	}{
		"bad json":          {`{`, http.StatusBadRequest},
		"short username":    {`{"username":"ab","password":"secret1","name":"n","code":"c"}`, http.StatusBadRequest},
		"long username":     {`{"username":"` + strings.Repeat("a", 51) + `","password":"secret1","name":"n","code":"c"}`, http.StatusBadRequest},
		"short password":    {`{"username":"alice","password":"123","name":"n","code":"c"}`, http.StatusBadRequest},
		"oversize password": {`{"username":"alice","password":"` + strings.Repeat("p", 73) + `","name":"n","code":"c"}`, http.StatusBadRequest},
		"empty name":        {`{"username":"alice","password":"secret1","name":"","code":"c"}`, http.StatusBadRequest},
		"empty code":        {`{"username":"alice","password":"secret1","name":"n","code":""}`, http.StatusBadRequest},
	}
	for name, tc := range cases {
		if rec := postJSON(e, "/auth/register", tc.body); rec.Code != tc.want {
			t.Fatalf("%s: status = %d, want %d", name, rec.Code, tc.want)
		}
	}
}

func TestLoginValidation(t *testing.T) {
	e, _ := authHarness()
	if rec := postJSON(e, "/auth/login", `{`); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad json status = %d", rec.Code)
	}
	if rec := postJSON(e, "/auth/login", `{"username":"","password":""}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("empty creds status = %d", rec.Code)
	}
}

func TestRefreshRequiresToken(t *testing.T) {
	e, _ := authHarness()
	if rec := postJSON(e, "/auth/refresh", `{}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("missing token status = %d, want 400", rec.Code)
	}
	// 非 JWT 的 token 必须被拒
	if rec := postJSON(e, "/auth/refresh", `{"refresh_token":"not-a-jwt"}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("garbage token status = %d, want 401", rec.Code)
	}
}

func TestMeRequiresTenantContext(t *testing.T) {
	e, _ := authHarness()
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/auth/me", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestLogoutWithoutTokenIsSafe(t *testing.T) {
	e, _ := authHarness()
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/auth/logout", nil))
	if rec.Code >= http.StatusInternalServerError {
		t.Fatalf("logout status = %d, 无 token 时不应 5xx", rec.Code)
	}
	// 必须下发清除 refresh cookie 的指令
	if !strings.Contains(rec.Header().Get("Set-Cookie"), "refresh_token=") {
		t.Fatalf("logout 未清理 refresh cookie: %q", rec.Header().Get("Set-Cookie"))
	}
}
