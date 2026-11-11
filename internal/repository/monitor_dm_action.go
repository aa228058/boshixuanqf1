package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const DMActionStaleAfter = 5 * time.Minute

// DMActionRepository persists one-time DM action claims across processes and restarts.
type DMActionRepository struct {
	db *pgxpool.Pool
}

func NewDMActionRepository(db *pgxpool.Pool) *DMActionRepository {
	return &DMActionRepository{db: db}
}

// Claim atomically creates an in_progress row. A conflict means another delivery
// already claimed or completed this exact action; it must never be resent.
// A claim left in_progress for five minutes is finalized as outcome_unknown,
// never reclaimed: Telegram may have accepted the external action before a crash.
func (r *DMActionRepository) Claim(ctx context.Context, tenantID, accountID, ruleID uuid.UUID, targetID, sourceChatID int64, sourceMessageID int, action string) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		INSERT INTO monitor_dm_actions
			(tenant_id, account_id, rule_id, target_id, source_chat_id, source_message_id, action, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,'in_progress')
		ON CONFLICT (tenant_id, account_id, rule_id, target_id, source_chat_id, source_message_id, action)
		DO NOTHING
	`, tenantID, accountID, ruleID, targetID, sourceChatID, sourceMessageID, action)
	if err != nil {
		return false, fmt.Errorf("claim monitor DM action: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return true, nil
	}

	// A duplicate delivery is also an opportunity to close a stale claim for this
	// exact key. It remains non-claimable, so an uncertain send is never repeated.
	_, err = r.db.Exec(ctx, `
		UPDATE monitor_dm_actions
		SET status='outcome_unknown',
		    last_error=COALESCE(NULLIF(last_error,''), 'claim remained in_progress for more than five minutes; external outcome unknown'),
		    updated_at=NOW()
		WHERE tenant_id=$1 AND account_id=$2 AND rule_id=$3 AND target_id=$4
		  AND source_chat_id=$5 AND source_message_id=$6 AND action=$7
		  AND status='in_progress' AND updated_at < NOW() - ($8::bigint * INTERVAL '1 second')
	`, tenantID, accountID, ruleID, targetID, sourceChatID, sourceMessageID, action, int64(DMActionStaleAfter/time.Second))
	if err != nil {
		return false, fmt.Errorf("resolve stale monitor DM action claim: %w", err)
	}
	return false, nil
}

// ResolveStale finalizes claims left behind by a process crash. It deliberately
// does not make them claimable again because the external Telegram result is unknown.
// Idempotency-key rows are retained as tombstones to prevent old updates replaying.
func (r *DMActionRepository) ResolveStale(ctx context.Context, olderThan time.Duration) (int64, error) {
	if olderThan < time.Second {
		return 0, fmt.Errorf("stale DM action age must be at least one second")
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE monitor_dm_actions
		SET status='outcome_unknown',
		    last_error=COALESCE(NULLIF(last_error,''), 'process ended while action was in_progress; external outcome unknown'),
		    updated_at=NOW()
		WHERE status='in_progress'
		  AND updated_at < NOW() - ($1::bigint * INTERVAL '1 second')
	`, int64(olderThan/time.Second))
	if err != nil {
		return 0, fmt.Errorf("resolve stale monitor DM action claims: %w", err)
	}
	return tag.RowsAffected(), nil
}

// Finish only transitions a still-running claim; late updates cannot overwrite a terminal result.
func (r *DMActionRepository) Finish(ctx context.Context, tenantID, accountID, ruleID uuid.UUID, targetID, sourceChatID int64, sourceMessageID int, action, status, lastError string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE monitor_dm_actions
		SET status=$8, last_error=NULLIF($9,''), updated_at=NOW()
		WHERE tenant_id=$1 AND account_id=$2 AND rule_id=$3 AND target_id=$4
		  AND source_chat_id=$5 AND source_message_id=$6 AND action=$7
		  AND status='in_progress'
	`, tenantID, accountID, ruleID, targetID, sourceChatID, sourceMessageID, action, status, lastError)
	if err != nil {
		return fmt.Errorf("finish monitor DM action: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("monitor DM action claim missing or no longer in_progress")
	}
	return nil
}
