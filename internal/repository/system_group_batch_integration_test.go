//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"tg-sender/internal/domain"
)

func TestSystemGroupUpsertBatchPersistsLinksAndContributions(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	tenantID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO tenants (tenant_id, username, password_hash, name, api_key_hash, plan_id)
		VALUES ($1, $2, 'x', 'Group Batch Test', $3, 'free')
	`, tenantID, "groupbatch_"+tenantID.String()[:8], "groupbatch-key-"+tenantID.String())
	if err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	repo := NewSystemGroupRepository(pool)
	groups := []*domain.SystemGroup{
		{Source: "import", Title: "Batch Group One", Username: "batchgroupone", TmeLink: "https://t.me/batchgroupone", GType: "group", Status: "active", ContributorTenantID: tenantID},
		{Source: "import", Title: "Batch Group Two", Username: "batchgrouptwo", TmeLink: "https://t.me/batchgrouptwo", GType: "channel", Status: "active", ContributorTenantID: tenantID},
	}
	if err := repo.UpsertBatch(ctx, groups); err != nil {
		t.Fatalf("upsert batch: %v", err)
	}
	// A URL spelling variant must normalize to the same contribution identity.
	groups[0].TmeLink = "https://www.t.me/batchgroupone/?start=campaign"
	// Replaying the same import chunk must not inflate the number of distinct links.
	if err := repo.UpsertBatch(ctx, groups); err != nil {
		t.Fatalf("replay batch: %v", err)
	}

	for _, group := range groups {
		var count, refCount, contributionCount int
		if err := pool.QueryRow(ctx, `
			SELECT sg.ref_count,
			       (SELECT COUNT(*) FROM system_group_links l WHERE l.sg_id=sg.sg_id),
			       (SELECT COUNT(*) FROM system_group_contributions c WHERE c.sg_id=sg.sg_id AND c.tenant_id=$2)
			FROM system_groups sg WHERE sg.dedup_key=$1
		`, group.DedupKey(), tenantID).Scan(&refCount, &count, &contributionCount); err != nil {
			t.Fatalf("read persisted group %q: %v", group.Username, err)
		}
		if refCount != 1 || count != 1 || contributionCount != 1 {
			t.Fatalf("group %q counts: ref_count=%d links=%d contributions=%d; want 1 each", group.Username, refCount, count, contributionCount)
		}
	}
}
