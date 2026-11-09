//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"tg-sender/internal/domain"
)

func TestSystemGroupUpsertReturnsCanonicalIDOnDedupConflict(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	tenantID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO tenants (tenant_id, username, password_hash, name, api_key_hash, plan_id)
		VALUES ($1, 'systemgroupupsert', 'x', 'System Group Upsert', 'system-group-upsert-key', 'free')
	`, tenantID)
	if err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	repo := NewSystemGroupRepository(pool)
	first := &domain.SystemGroup{
		Source: "scraped", Title: "Example Group", Username: "same_group_name",
		TmeLink: "https://t.me/same_group_name", GType: "supergroup",
		MemberCount: 100, Status: "active", TelegramChatID: 101,
		ContributorTenantID: tenantID,
	}
	if err := repo.Upsert(ctx, first); err != nil {
		t.Fatalf("insert initial system group: %v", err)
	}

	// A second observation with the same dedup key but a different Telegram ID
	// conflicts with the canonical group row. Upsert must still retrieve the
	// canonical sg_id and persist the newly observed link/contribution.
	second := &domain.SystemGroup{
		Source: "bio", Title: "Example Group", Username: "same_group_name",
		TmeLink: "https://t.me/same_group_name_alias", GType: "supergroup",
		MemberCount: 120, Status: "active", TelegramChatID: 202,
		ContributorTenantID: tenantID,
	}
	if err := repo.Upsert(ctx, second); err != nil {
		t.Fatalf("upsert deduplicated system group: %v", err)
	}

	var groups, links, contributions int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM system_groups WHERE dedup_key=$1`, second.DedupKey()).Scan(&groups); err != nil {
		t.Fatalf("count canonical groups: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM system_group_links WHERE normalized_link IN ($1,$2)`,
		"t.me/same_group_name", "t.me/same_group_name_alias").Scan(&links); err != nil {
		t.Fatalf("count group links: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM system_group_contributions WHERE sg_id=(SELECT sg_id FROM system_groups WHERE dedup_key=$1)`,
		second.DedupKey()).Scan(&contributions); err != nil {
		t.Fatalf("count group contributions: %v", err)
	}
	if groups != 1 || links != 2 || contributions != 1 {
		t.Fatalf("canonical rows/groups=%d links=%d contributions=%d, want 1/2/1", groups, links, contributions)
	}
}
