package task

import (
	"context"
	"testing"

	"tg-sender/internal/domain"
)

type nilAccountLister struct{}

func (nilAccountLister) ListAll(context.Context) ([]*domain.Account, error) {
	return []*domain.Account{nil, nil}, nil
}

func TestLinkScraperSyncOnlineAccountsSkipsNilRows(t *testing.T) {
	ls := NewLinkScraper(nil, nil, nil, nil, nilAccountLister{})
	ls.syncOnlineAccounts(context.Background())
	if got := ls.GetStats()["handlers_active"]; got != 0 {
		t.Fatalf("handlers_active = %v, want 0", got)
	}
}
