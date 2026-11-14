package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func errorBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	return body
}

// 5xx 不能把底层错误原文回给客户端：SQL 报错带表名/约束名，连接错误可能带 DSN 片段。
func TestAPIErrorHandlerHidesInternalErrorDetails(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = APIErrorHandler
	e.GET("/boom", func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusInternalServerError,
			`failed to query accounts: ERROR: relation "accounts" does not exist (SQLSTATE 42P01)`)
	})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/boom", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	body := errorBody(t, rec)
	if msg, _ := body["message"].(string); msg != "internal server error" {
		t.Fatalf("message = %q, 内部细节泄漏", body["message"])
	}
	if code, _ := body["code"].(float64); int(code) != 500 {
		t.Fatalf("code = %v, want 500", body["code"])
	}
}

// 4xx 的业务文案必须原样保留：前端直接展示这些中文提示。
func TestAPIErrorHandlerKeepsClientErrorMessages(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = APIErrorHandler
	e.GET("/bad", func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusBadRequest, "请求参数错误")
	})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/bad", nil))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if msg, _ := errorBody(t, rec)["message"].(string); msg != "请求参数错误" {
		t.Fatalf("message = %q, want 请求参数错误", msg)
	}
}

// 非 echo.HTTPError 的裸错误一律按 500 处理，且不回显原文。
func TestAPIErrorHandlerTreatsPlainErrorAsInternal(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = APIErrorHandler
	e.GET("/plain", func(c echo.Context) error {
		return errors.New("pq: password authentication failed for user \"postgres\"")
	})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/plain", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if msg, _ := errorBody(t, rec)["message"].(string); msg != "internal server error" {
		t.Fatalf("message = %q, 内部细节泄漏", msg)
	}
}

func TestClampPageSize(t *testing.T) {
	cases := map[int]int{
		0:               defaultPageSize,
		-5:              defaultPageSize,
		1:               1,
		50:              50,
		maxPageSize:     maxPageSize,
		maxPageSize + 1: maxPageSize,
		1000000:         maxPageSize,
	}
	for in, want := range cases {
		if got := clampPageSize(in); got != want {
			t.Fatalf("clampPageSize(%d) = %d, want %d", in, got, want)
		}
	}
}
