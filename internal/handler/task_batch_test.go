package handler

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestParseBatchTaskIDs(t *testing.T) {
	t.Run("rejects empty list", func(t *testing.T) {
		if _, err := parseBatchTaskIDs(nil); err == nil {
			t.Fatal("expected empty list to be rejected")
		}
	})

	t.Run("rejects oversized list", func(t *testing.T) {
		ids := make([]string, 501)
		for i := range ids {
			ids[i] = uuid.NewString()
		}
		if _, err := parseBatchTaskIDs(ids); err == nil {
			t.Fatal("expected more than 500 IDs to be rejected")
		}
	})

	t.Run("rejects malformed ID", func(t *testing.T) {
		if _, err := parseBatchTaskIDs([]string{"not-a-uuid"}); err == nil || !strings.Contains(err.Error(), "invalid task id") {
			t.Fatalf("error = %v, want invalid task ID error", err)
		}
	})

	t.Run("accepts maximum batch size", func(t *testing.T) {
		ids := make([]string, 500)
		for i := range ids {
			ids[i] = uuid.NewString()
		}
		parsed, err := parseBatchTaskIDs(ids)
		if err != nil {
			t.Fatalf("parse maximum batch: %v", err)
		}
		if len(parsed) != 500 {
			t.Fatalf("parsed %d IDs, want 500", len(parsed))
		}
	})
}
