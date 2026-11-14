package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestMediaUploadRejectsMissingFile(t *testing.T) {
	h := &MediaHandler{}
	e := echo.New()
	e.POST("/media", h.Upload)
	req := httptest.NewRequest(http.MethodPost, "/media", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEMultipartForm)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestContactListValidation(t *testing.T) {
	h := &ContactHandler{}
	e := newSmokeEcho(func(e *echo.Echo) { e.GET("/contacts", h.List) })
	if rec := hitRec(e, http.MethodGet, "/contacts"); rec.Code != http.StatusBadRequest {
		t.Fatalf("缺 account_id status = %d, want 400", rec.Code)
	}
	if rec := hitRec(e, http.MethodGet, "/contacts?account_id=not-a-uuid"); rec.Code != http.StatusBadRequest {
		t.Fatalf("非法 account_id status = %d, want 400", rec.Code)
	}
}

func hitRec(e *echo.Echo, method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}
