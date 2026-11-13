package handler

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TestParseOptionalUUIDParam(t *testing.T) {
	t.Run("empty value leaves filter unset", func(t *testing.T) {
		got, err := parseOptionalUUIDParam("", "account_id")
		if err != nil || got != nil {
			t.Fatalf("got (%v, %v), want (nil, nil)", got, err)
		}
	})

	t.Run("valid UUID is parsed", func(t *testing.T) {
		want := uuid.New()
		got, err := parseOptionalUUIDParam(want.String(), "account_id")
		if err != nil || got == nil || *got != want {
			t.Fatalf("got (%v, %v), want (%s, nil)", got, err, want)
		}
	})

	t.Run("invalid UUID is rejected instead of dropping filter", func(t *testing.T) {
		got, err := parseOptionalUUIDParam("not-a-uuid", "account_id")
		if got != nil || err == nil {
			t.Fatalf("got (%v, %v), want (nil, bad request)", got, err)
		}
		httpErr, ok := err.(*echo.HTTPError)
		if !ok || httpErr.Code != http.StatusBadRequest {
			t.Fatalf("got error %T (%v), want HTTP 400", err, err)
		}
	})
}
