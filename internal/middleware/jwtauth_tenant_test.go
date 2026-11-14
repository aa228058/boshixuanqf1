package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"tg-sender/internal/config"
	"tg-sender/internal/domain"
)

// fakeTenantRepo 只实现 GetByID；其余方法不应在鉴权路径上被调用。
type fakeTenantRepo struct {
	tenants map[uuid.UUID]*domain.Tenant
	err     error
}

func (f *fakeTenantRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Tenant, error) {
	if f.err != nil {
		return nil, f.err
	}
	if t, ok := f.tenants[id]; ok {
		return t, nil
	}
	return nil, nil
}

func (f *fakeTenantRepo) GetByUsername(context.Context, string) (*domain.Tenant, error) {
	panic("GetByUsername must not be called by JWTAuth")
}

func (f *fakeTenantRepo) Create(context.Context, *domain.Tenant, string) error {
	panic("Create must not be called by JWTAuth")
}

func (f *fakeTenantRepo) Update(context.Context, uuid.UUID, map[string]any) error {
	panic("Update must not be called by JWTAuth")
}

func (f *fakeTenantRepo) List(context.Context, int, int, string, string, string) ([]*domain.Tenant, int, error) {
	panic("List must not be called by JWTAuth")
}

func testConfig(secret string) *config.Config {
	return &config.Config{JWTSecret: secret, JWTExpiry: time.Hour}
}

// authHarness 组装一个只挂 JWTAuth 的 echo 实例，受保护路由把当前租户回显出来，
// 便于断言"这个请求被认成了哪个租户"。
func authHarness(t *testing.T, cfg *config.Config, repo domain.TenantRepository) *echo.Echo {
	t.Helper()
	e := echo.New()
	e.Use(JWTAuth(cfg, repo, nil))
	e.GET("/api/v1/protected", func(c echo.Context) error {
		tenantID, err := GetTenantID(c)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, map[string]string{"tenant_id": tenantID.String()})
	})
	return e
}

func doRequest(e *echo.Echo, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func tenantIDFromBody(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		TenantID string `json:"tenant_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	return body.TenantID
}

func signedToken(t *testing.T, secret, issuer, audience, tenantID, tokenType string, expiresAt time.Time) string {
	t.Helper()
	claims := JWTClaims{
		TenantID:  tenantID,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{audience},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func TestJWTAuthRejectsMissingOrMalformedHeader(t *testing.T) {
	cfg := testConfig(strings.Repeat("a", 40))
	e := authHarness(t, cfg, nil)

	for _, header := range []string{"", "Token abc", "Bearer", "Basic dXNlcjpwYXNz", "Bearer not-a-jwt"} {
		rec := doRequest(e, header)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("header %q: status = %d, want 401", header, rec.Code)
		}
	}
}

func TestJWTAuthAcceptsValidAccessToken(t *testing.T) {
	cfg := testConfig(strings.Repeat("b", 40))
	tenantID := uuid.New()
	e := authHarness(t, cfg, nil)

	token, err := GenerateToken(cfg, tenantID.String())
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	rec := doRequest(e, "Bearer "+token)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := tenantIDFromBody(t, rec); got != tenantID.String() {
		t.Fatalf("tenant = %s, want %s", got, tenantID)
	}
}

// 刷新令牌不能当访问令牌用：它的有效期长得多，放进来等于绕过短期凭证。
func TestJWTAuthRejectsRefreshTokenAsAccess(t *testing.T) {
	cfg := testConfig(strings.Repeat("c", 40))
	token, _, err := GenerateRefreshToken(cfg, uuid.New().String())
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}
	rec := doRequest(authHarness(t, cfg, nil), "Bearer "+token)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestJWTAuthRejectsBadSignatureIssuerAudienceAndExpiry(t *testing.T) {
	cfg := testConfig(strings.Repeat("d", 40))
	tenantID := uuid.New().String()
	e := authHarness(t, cfg, nil)

	cases := map[string]string{
		"wrong secret":   signedToken(t, strings.Repeat("x", 40), TokenIssuer, TokenAudience, tenantID, "access", time.Now().Add(time.Hour)),
		"wrong issuer":   signedToken(t, cfg.JWTSecret, "someone-else", TokenAudience, tenantID, "access", time.Now().Add(time.Hour)),
		"wrong audience": signedToken(t, cfg.JWTSecret, TokenIssuer, "other-api", tenantID, "access", time.Now().Add(time.Hour)),
		"expired":        signedToken(t, cfg.JWTSecret, TokenIssuer, TokenAudience, tenantID, "access", time.Now().Add(-time.Minute)),
	}
	for name, token := range cases {
		if rec := doRequest(e, "Bearer "+token); rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s: status = %d, want 401", name, rec.Code)
		}
	}
}

func TestJWTAuthEnforcesTenantStatus(t *testing.T) {
	cfg := testConfig(strings.Repeat("e", 40))
	active := uuid.New()
	suspended := uuid.New()
	expired := uuid.New()
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	repo := &fakeTenantRepo{tenants: map[uuid.UUID]*domain.Tenant{
		active:    {TenantID: active, Status: "active", ExpiresAt: &future},
		suspended: {TenantID: suspended, Status: "suspended", ExpiresAt: &future},
		expired:   {TenantID: expired, Status: "active", ExpiresAt: &past},
	}}
	e := authHarness(t, cfg, repo)

	tokenFor := func(id uuid.UUID) string {
		token, err := GenerateToken(cfg, id.String())
		if err != nil {
			t.Fatalf("generate token: %v", err)
		}
		return token
	}

	if rec := doRequest(e, "Bearer "+tokenFor(active)); rec.Code != http.StatusOK {
		t.Fatalf("active tenant: status = %d, want 200", rec.Code)
	}
	if rec := doRequest(e, "Bearer "+tokenFor(suspended)); rec.Code != http.StatusForbidden {
		t.Fatalf("suspended tenant: status = %d, want 403", rec.Code)
	}
	if rec := doRequest(e, "Bearer "+tokenFor(expired)); rec.Code != http.StatusForbidden {
		t.Fatalf("expired tenant: status = %d, want 403", rec.Code)
	}
	// 库里查不到的租户不能被当成有效身份。
	if rec := doRequest(e, "Bearer "+tokenFor(uuid.New())); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unknown tenant: status = %d, want 401", rec.Code)
	}
}

// 隔离性：并发/连续请求之间不能残留上一个请求的租户身份。
func TestJWTAuthDoesNotLeakTenantBetweenRequests(t *testing.T) {
	cfg := testConfig(strings.Repeat("f", 40))
	tenantA, tenantB := uuid.New(), uuid.New()
	future := time.Now().Add(time.Hour)
	repo := &fakeTenantRepo{tenants: map[uuid.UUID]*domain.Tenant{
		tenantA: {TenantID: tenantA, Status: "active", ExpiresAt: &future},
		tenantB: {TenantID: tenantB, Status: "active", ExpiresAt: &future},
	}}
	e := authHarness(t, cfg, repo)

	tokenA, _ := GenerateToken(cfg, tenantA.String())
	tokenB, _ := GenerateToken(cfg, tenantB.String())

	if got := tenantIDFromBody(t, doRequest(e, "Bearer "+tokenA)); got != tenantA.String() {
		t.Fatalf("first request tenant = %s, want %s", got, tenantA)
	}
	if got := tenantIDFromBody(t, doRequest(e, "Bearer "+tokenB)); got != tenantB.String() {
		t.Fatalf("second request tenant = %s, want %s (串号)", got, tenantB)
	}
	// 无凭证的请求不能继承上一次的身份。
	if rec := doRequest(e, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated request status = %d, want 401", rec.Code)
	}
}
