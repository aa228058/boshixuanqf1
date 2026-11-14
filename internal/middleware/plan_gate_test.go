package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

// featureHarness 直接把 plan_features 塞进上下文，隔离掉 PlanContext 的 DB 依赖，
// 专测"功能关 → 403 / 功能开 → 放行"这条门禁规则。
func featureHarness(features any) *echo.Echo {
	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("plan_features", features)
			return next(c)
		}
	})
	pc := &PlanChecker{}
	e.GET("/monitor", func(c echo.Context) error { return c.NoContent(http.StatusOK) },
		pc.RequirePlanFeature(FeatureMonitor))
	return e
}

func TestRequirePlanFeatureAllowsEnabledFeature(t *testing.T) {
	rec := httptest.NewRecorder()
	featureHarness(map[string]any{FeatureMonitor: true}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/monitor", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestRequirePlanFeatureBlocksDisabledFeature(t *testing.T) {
	cases := map[string]any{
		"false":        map[string]any{FeatureMonitor: false},
		"missing":      map[string]any{},
		"nil features": nil,
		"string false": map[string]any{FeatureMonitor: "false"},
		"zero int":     map[string]any{FeatureMonitor: 0},
	}
	for name, features := range cases {
		rec := httptest.NewRecorder()
		featureHarness(features).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/monitor", nil))
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s: status = %d, want 403", name, rec.Code)
		}
	}
}

// 老数据里 features 可能是字符串数组（迁移前的形态），仍需兼容放行。
func TestRequirePlanFeatureSupportsLegacySlice(t *testing.T) {
	rec := httptest.NewRecorder()
	featureHarness([]any{FeatureMonitor}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/monitor", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
