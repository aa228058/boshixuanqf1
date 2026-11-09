package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CallCooldownRepository atomically enforces per-tenant/account/callee cooldowns.
type CallCooldownRepository struct {
	db *pgxpool.Pool
}

func NewCallCooldownRepository(db *pgxpool.Pool) *CallCooldownRepository {
	return &CallCooldownRepository{db: db}
}

// Claim returns true only when this caller reserves the ten-minute window.
// PostgreSQL's unique key and conditional upsert serialize concurrent instances.
func (r *CallCooldownRepository) Claim(ctx context.Context, tenantID, accountID uuid.UUID, calleeID int64) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		INSERT INTO monitor_call_cooldowns (tenant_id, account_id, callee_id, last_called_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (tenant_id, account_id, callee_id)
		DO UPDATE SET last_called_at = NOW(), updated_at = NOW()
		WHERE monitor_call_cooldowns.last_called_at <= NOW() - INTERVAL '10 minutes'
	`, tenantID, accountID, calleeID)
	if err != nil {
		return false, fmt.Errorf("claim monitor call cooldown: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}


// PruneExpired removes cooldown rows that can no longer block a call.
// Unlike DM action idempotency keys, expired cooldown rows are safe to delete.
func (r *CallCooldownRepository) PruneExpired(ctx context.Context) (int64, error) {
	tag, err := r.db.Exec(ctx, `
		DELETE FROM monitor_call_cooldowns
		WHERE last_called_at < NOW() - INTERVAL '10 minutes'
	`)
	if err != nil {
		return 0, fmt.Errorf("prune expired monitor call cooldowns: %w", err)
	}
	return tag.RowsAffected(), nil
}
