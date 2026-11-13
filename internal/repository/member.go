package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"tg-sender/internal/domain"
)

// MemberRepository handles member data access.
type MemberRepository struct {
	db *pgxpool.Pool
}

// NewMemberRepository creates a new member repository.
func NewMemberRepository(db *pgxpool.Pool) *MemberRepository {
	return &MemberRepository{db: db}
}

// ListByGroup retrieves members for a group.
func (r *MemberRepository) ListByGroup(ctx context.Context, tenantID, groupID uuid.UUID, limit, offset int) ([]*domain.Member, error) {
	rows, err := r.db.Query(ctx, `
		SELECT member_id, tenant_id, group_id, tg_user_id, username, first_name, last_name, phone, bio, last_seen, scraped_at, access_hash
		FROM members WHERE tenant_id = $1 AND group_id = $2 ORDER BY scraped_at DESC LIMIT $3 OFFSET $4`,
		tenantID, groupID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	var members []*domain.Member
	for rows.Next() {
		var m domain.Member
		// P0-2 审计修复: 原 SELECT/Scan 漏 access_hash——邀请入群时所有成员 AccessHash=0 全跳过
		err := rows.Scan(&m.MemberID, &m.TenantID, &m.GroupID, &m.TGUserID, &m.Username, &m.FirstName,
			&m.LastName, &m.Phone, &m.Bio, &m.LastSeen, &m.ScrapedAt, &m.AccessHash)
		if err != nil {
			return nil, err
		}
		members = append(members, &m)
	}
	return members, rows.Err()
}

// ListByTenant retrieves all members across all groups for a tenant.
func (r *MemberRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]*domain.Member, error) {
	rows, err := r.db.Query(ctx, `
		SELECT member_id, tenant_id, group_id, tg_user_id, username, first_name, last_name, phone, bio, last_seen, scraped_at, access_hash
		FROM members WHERE tenant_id = $1 ORDER BY scraped_at DESC LIMIT $2`,
		tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("list members by tenant: %w", err)
	}
	defer rows.Close()

	var members []*domain.Member
	for rows.Next() {
		var m domain.Member
		// P0-2 审计修复: 原 SELECT/Scan 漏 access_hash——邀请入群时所有成员 AccessHash=0 全跳过
		err := rows.Scan(&m.MemberID, &m.TenantID, &m.GroupID, &m.TGUserID, &m.Username, &m.FirstName,
			&m.LastName, &m.Phone, &m.Bio, &m.LastSeen, &m.ScrapedAt, &m.AccessHash)
		if err != nil {
			return nil, err
		}
		members = append(members, &m)
	}
	return members, rows.Err()
}

// CountByGroup counts members for a group.
func (r *MemberRepository) CountByGroup(ctx context.Context, tenantID, groupID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM members WHERE tenant_id = $1 AND group_id = $2`, tenantID, groupID).Scan(&count)
	return count, err
}

// BatchCreate inserts multiple members using COPY FROM for high performance.
// Falls back to batch INSERT with unnest if COPY is not available.
func (r *MemberRepository) BatchCreate(ctx context.Context, members []*domain.Member) error {
	// 防御: producer 未生成主键时逐行补齐, 否则所有行同为 uuid.Nil
	// 会在 CopyFrom/INSERT 时撞 members_pkey 主键导致整批失败。
	for _, m := range members {
		if m.MemberID == uuid.Nil {
			m.MemberID = uuid.New()
		}
	}
	if len(members) == 0 {
		return nil
	}

	// Use COPY FROM for optimal bulk insert performance
	copyCount, err := r.db.CopyFrom(ctx, pgx.Identifier{"members"}, []string{
		"member_id", "tenant_id", "group_id", "tg_user_id", "username",
		"first_name", "last_name", "phone", "bio", "last_seen", "scraped_at", "access_hash",
	}, pgx.CopyFromSlice(len(members), func(i int) ([]interface{}, error) {
		m := members[i]
		return []interface{}{
			m.MemberID, m.TenantID, m.GroupID, m.TGUserID, m.Username,
			m.FirstName, m.LastName, m.Phone, m.Bio, m.LastSeen, m.ScrapedAt, m.AccessHash,
		}, nil
	}))
	if err == nil && int(copyCount) == len(members) {
		return nil
	}

	// Fallback: use batch INSERT with ON CONFLICT for partial failures
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		for _, m := range members {
			_, err := tx.Exec(ctx, `
				INSERT INTO members (member_id, tenant_id, group_id, tg_user_id, username, first_name, last_name, phone, bio, last_seen, has_photo, scraped_at, access_hash)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
				ON CONFLICT ON CONSTRAINT members_tenant_id_group_id_tg_user_id_key DO UPDATE SET
					username = EXCLUDED.username, first_name = EXCLUDED.first_name, last_name = EXCLUDED.last_name,
					phone = EXCLUDED.phone, bio = EXCLUDED.bio, last_seen = EXCLUDED.last_seen, has_photo = EXCLUDED.has_photo, scraped_at = EXCLUDED.scraped_at, access_hash = EXCLUDED.access_hash`,
				m.MemberID, m.TenantID, m.GroupID, m.TGUserID, m.Username, m.FirstName, m.LastName, m.Phone, m.Bio, m.LastSeen, m.HasPhoto, m.ScrapedAt, m.AccessHash)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// BatchCreateOwned persists monitor-scraped members and their progress only while
// the task lease is still owned by workerID. The row lock serializes both writes
// with cancellation and lease reclamation: whichever transaction locks first wins.
func (r *MemberRepository) BatchCreateOwned(ctx context.Context, tenantID, taskID uuid.UUID, workerID string, members []*domain.Member) error {
	if workerID == "" {
		return fmt.Errorf("persist owned members: empty worker ID")
	}
	if len(members) == 0 {
		return nil
	}
	for _, m := range members {
		if m == nil {
			return fmt.Errorf("persist owned members: nil member")
		}
		if m.TenantID != tenantID {
			return fmt.Errorf("persist owned members: member tenant does not match task tenant")
		}
		if m.MemberID == uuid.Nil {
			m.MemberID = uuid.New()
		}
	}

	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		var currentWorkerID string
		err := tx.QueryRow(ctx, `
			SELECT worker_id
			FROM tasks
			WHERE tenant_id = $1 AND task_id = $2 AND status = 'running'
			FOR NO KEY UPDATE
		`, tenantID, taskID).Scan(&currentWorkerID)
		if err != nil {
			if err == pgx.ErrNoRows {
				return fmt.Errorf("persist owned members: task is not running or no longer exists")
			}
			return fmt.Errorf("persist owned members: verify task lease: %w", err)
		}
		if currentWorkerID != workerID {
			return fmt.Errorf("persist owned members: task lease is owned by another worker")
		}

		for _, m := range members {
			_, err := tx.Exec(ctx, `
				INSERT INTO members (member_id, tenant_id, group_id, tg_user_id, username, first_name, last_name, phone, bio, last_seen, has_photo, scraped_at, access_hash)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
				ON CONFLICT ON CONSTRAINT members_tenant_id_group_id_tg_user_id_key DO UPDATE SET
					username = EXCLUDED.username, first_name = EXCLUDED.first_name, last_name = EXCLUDED.last_name,
					phone = EXCLUDED.phone, bio = EXCLUDED.bio, last_seen = EXCLUDED.last_seen,
					has_photo = EXCLUDED.has_photo, scraped_at = EXCLUDED.scraped_at, access_hash = EXCLUDED.access_hash
			`, m.MemberID, m.TenantID, m.GroupID, m.TGUserID, m.Username, m.FirstName, m.LastName,
				m.Phone, m.Bio, m.LastSeen, m.HasPhoto, m.ScrapedAt, m.AccessHash)
			if err != nil {
				return fmt.Errorf("persist owned member %d: %w", m.TGUserID, err)
			}
		}

		// Keep the member batch and its progress increment in one transaction.
		// The task row is already locked against cancellation/reclaim above.
		lastMember := members[len(members)-1]
		progressResult, err := tx.Exec(ctx, `
			UPDATE tasks
			SET progress = jsonb_set(
				jsonb_set(COALESCE(progress, '{}'::jsonb), '{done}',
					to_jsonb(COALESCE((progress->>'done')::int, 0) + $3)),
				'{current_target}', to_jsonb($4::text)
			)
			WHERE tenant_id = $1 AND task_id = $2 AND status = 'running' AND worker_id = $5
		`, tenantID, taskID, len(members), fmt.Sprintf("user_%d", lastMember.TGUserID), workerID)
		if err != nil {
			return fmt.Errorf("persist owned member progress: %w", err)
		}
		if progressResult.RowsAffected() != 1 {
			return fmt.Errorf("persist owned member progress: task lease lost")
		}
		return nil
	})
}

// DeleteByGroup removes all members of a group.
func (r *MemberRepository) DeleteByGroup(ctx context.Context, tenantID, groupID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM members WHERE tenant_id = $1 AND group_id = $2`, tenantID, groupID)
	return err
}

// ===== 联系人缓存(私聊昵称反查) =====

// UpsertContact 写入或更新单个联系人(同步私聊会话时调用)
func (r *MemberRepository) UpsertContact(ctx context.Context, tenantID, accountID uuid.UUID, tgUserID int64, username, firstName, lastName string, accessHash int64) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO contacts (contact_id, tenant_id, account_id, tg_user_id, username, first_name, last_name, access_hash, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		ON CONFLICT (account_id, tg_user_id) DO UPDATE SET
			username = EXCLUDED.username, first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name, access_hash = EXCLUDED.access_hash, updated_at = NOW()`,
		tenantID, accountID, tgUserID, username, firstName, lastName, accessHash)
	return err
}

// GetAccessHashByTGUserID 返回某 TG 用户的 AccessHash(私聊发送兜底用), 未找到返回 0
func (r *MemberRepository) GetAccessHashByTGUserID(ctx context.Context, tenantID uuid.UUID, tgUserID int64) (int64, error) {
	var h int64
	err := r.db.QueryRow(ctx, `
		SELECT access_hash FROM members
		WHERE tenant_id=$1 AND tg_user_id=$2
		ORDER BY scraped_at DESC LIMIT 1`, tenantID, tgUserID).Scan(&h)
	if err != nil {
		return 0, err
	}
	return h, nil
}

// ListContactRows 返回联系人列表(供前端选择器直接展示)
func (r *MemberRepository) ListContactRows(ctx context.Context, tenantID, accountID uuid.UUID) ([]map[string]any, error) {
	rows, err := r.db.Query(ctx, `
		SELECT tg_user_id, username, first_name, last_name, access_hash FROM contacts
		WHERE tenant_id = $1 AND account_id = $2 ORDER BY updated_at DESC LIMIT 1000`, tenantID, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, ah int64
		var un, fn, ln string
		if err := rows.Scan(&id, &un, &fn, &ln, &ah); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"tg_user_id": id, "username": un, "first_name": fn, "last_name": ln, "access_hash": ah})
	}
	return out, rows.Err()
}

// ListContactsByAccount 返回账号的联系人缓存(id -> 显示名)
func (r *MemberRepository) ListContactsByAccount(ctx context.Context, tenantID, accountID uuid.UUID) (map[int64]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT tg_user_id, username, first_name FROM contacts
		WHERE tenant_id = $1 AND account_id = $2`, tenantID, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]string{}
	for rows.Next() {
		var id int64
		var un, fn string
		if err := rows.Scan(&id, &un, &fn); err != nil {
			return nil, err
		}
		if un != "" {
			out[id] = "@" + un
		} else {
			out[id] = fn
		}
	}
	return out, rows.Err()
}
