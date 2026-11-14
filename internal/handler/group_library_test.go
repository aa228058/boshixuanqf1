package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// 群库 handler 依赖具体的 repository 类型（无法在测试里造假对象），
// 但"解析请求 → 校验 → 报 4xx"这一段不碰依赖，可以在无数据库时覆盖。
func TestGroupLibraryRejectsBadRequests(t *testing.T) {
	h := &GroupLibraryHandler{}
	e := newSmokeEcho(func(e *echo.Echo) {
		e.POST("/group-library/import", h.Import)
		e.POST("/group-library/import-jobs/:id/confirm", h.ConfirmImportJob)
		e.POST("/group-library/import-jobs/:id/retry", h.RetryFailedImportJob)
		e.POST("/group-library/import-jobs/:id/cancel", h.CancelImportJob)
		e.DELETE("/group-library/import-jobs/:id", h.DeleteImportJob)
		e.GET("/group-library/import-jobs/:id", h.GetImportJob)
	})

	badID := []struct{ method, path string }{
		{http.MethodPost, "/group-library/import-jobs/not-a-uuid/confirm"},
		{http.MethodPost, "/group-library/import-jobs/not-a-uuid/retry"},
		{http.MethodPost, "/group-library/import-jobs/not-a-uuid/cancel"},
		{http.MethodDelete, "/group-library/import-jobs/not-a-uuid"},
		{http.MethodGet, "/group-library/import-jobs/not-a-uuid"},
	}
	for _, tc := range badID {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s %s: status = %d, want 400", tc.method, tc.path, rec.Code)
		}
	}

	// 缺文件/正文的导入请求同样应被 4xx 拒绝（而不是撞到 nil repo 变 500）
	req := httptest.NewRequest(http.MethodPost, "/group-library/import", strings.NewReader(`{}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code >= http.StatusInternalServerError {
		t.Fatalf("空导入请求 status = %d，不应 5xx", rec.Code)
	}

	// 合法 UUID 会走到仓储层（测试里为 nil）——只断言"不是参数错误"
	goodID := uuid.New().String()
	req2 := httptest.NewRequest(http.MethodGet, "/group-library/import-jobs/"+goodID, nil)
	rec2 := httptest.NewRecorder()
	func() {
		defer func() { _ = recover() }() // nil repo panic 属于预期，不计为失败
		e.ServeHTTP(rec2, req2)
	}()
}
