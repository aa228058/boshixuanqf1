package grouplib

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/mtgo-labs/mtgo/telegram"
)

type importAccountEngineStub struct {
	getCalls     int
	releaseCalls int
}

func (s *importAccountEngineStub) GetClient(context.Context, uuid.UUID, uuid.UUID) (*telegram.Client, error) {
	s.getCalls++
	return nil, errors.New("unexpected MTProto acquisition")
}

func (s *importAccountEngineStub) ReleaseClient(uuid.UUID, uuid.UUID, *telegram.Client) {
	s.releaseCalls++
}

func TestImportByLinksBulkModeDoesNotAcquireMTProtoSession(t *testing.T) {
	acc := &importAccountEngineStub{}
	engine := &Engine{acc: acc}

	lines := make([]string, 201)
	for i := range lines {
		lines[i] = fmt.Sprintf("https://t.me/group%05d", i)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := engine.ImportByLinks(ctx, uuid.New(), uuid.New(), lines)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ImportByLinks error = %v, want context.Canceled", err)
	}
	if result == nil {
		t.Fatal("expected partial import result")
	}
	if acc.getCalls != 0 {
		t.Fatalf("GetClient calls = %d, want 0 for bulk import", acc.getCalls)
	}
	if acc.releaseCalls != 0 {
		t.Fatalf("ReleaseClient calls = %d, want 0 for bulk import", acc.releaseCalls)
	}
}
