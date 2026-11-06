package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"tg-sender/internal/domain"
)

// SystemGroupRepository 平台级群组库(找群功能)。跨租户共享, 所有查询无租户过滤。
type SystemGroupRepository struct {
	db *pgxpool.Pool
}

func NewSystemGroupRepository(db *pgxpool.Pool) *SystemGroupRepository {
	return &SystemGroupRepository{db: db}
}

// SystemGroupFilter 列表筛选
type SystemGroupFilter struct {
	Keyword  string // 标题/username/link 模糊
	GType    string // public | private | channel | unknown | ""
	Status   string // active | joined | invalid | ""
	Source   string // scraped | joined | bio | ""
	Page     int
	PageSize int
}

// Upsert 归并写入: dedup_key 冲突时热度+1、取更优标题/更大成员数、刷新最后收录时间。
func (r *SystemGroupRepository) Upsert(ctx context.Context, sg *domain.SystemGroup) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var sgID uuid.UUID
	if sg.TelegramChatID != 0 {
		lookupErr := tx.QueryRow(ctx, "SELECT sg_id FROM system_groups WHERE telegram_chat_id=$1 LIMIT 1", sg.TelegramChatID).Scan(&sgID)
		if lookupErr == nil {
			_, err = tx.Exec(ctx, "UPDATE system_groups SET dedup_key=CASE WHEN NOT EXISTS (SELECT 1 FROM system_groups other WHERE other.dedup_key='c:'||$2 AND other.sg_id<>$1) THEN 'c:'||$2 ELSE dedup_key END, title=CASE WHEN COALESCE(title,'')='' OR title=tme_link THEN $3::text ELSE title END, username=COALESCE(NULLIF(username,''),$4), member_count=CASE WHEN $5>0 THEN $5 ELSE member_count END, gtype=CASE WHEN gtype='unknown' THEN $6 ELSE gtype END, resolution_status=CASE WHEN $3::text<>'' OR $5>0 THEN 'resolved' ELSE resolution_status END, member_count_updated_at=CASE WHEN $5>0 THEN NOW() ELSE member_count_updated_at END, first_contributor_tenant_id=COALESCE(first_contributor_tenant_id,NULLIF($7::text,'00000000-0000-0000-0000-000000000000')::uuid), first_contributor_tenant_name=CASE WHEN first_contributor_tenant_id IS NULL THEN COALESCE((SELECT name FROM tenants WHERE tenant_id=NULLIF($7::text,'00000000-0000-0000-0000-000000000000')::uuid),'') ELSE first_contributor_tenant_name END, last_seen_at=NOW() WHERE sg_id=$1", sgID, sg.TelegramChatID, sg.Title, sg.Username, sg.MemberCount, sg.GType, sg.ContributorTenantID)
			if err != nil {
				return err
			}
		} else {
			sgID = uuid.Nil
		}
	}
	if sgID == uuid.Nil {
		_, err = tx.Exec(ctx, "INSERT INTO system_groups (dedup_key,source,title,username,tme_link,gtype,member_count,status,ref_count,last_seen_at,telegram_chat_id,resolution_status,member_count_updated_at,first_contributor_tenant_id,first_contributor_tenant_name) VALUES ($1,$2,$3::text,$4,$5,$6,$7,$8,1,NOW(),NULLIF($9,0),CASE WHEN $9<>0 THEN 'resolved' WHEN COALESCE($3::text,'')<>'' OR $7>0 THEN 'identity_pending' ELSE 'pending' END,CASE WHEN $7>0 THEN NOW() ELSE NULL END,NULLIF($10::text,'00000000-0000-0000-0000-000000000000')::uuid,COALESCE((SELECT name FROM tenants WHERE tenant_id=NULLIF($10::text,'00000000-0000-0000-0000-000000000000')::uuid),'')) ON CONFLICT (dedup_key) DO UPDATE SET ref_count=system_groups.ref_count+1, telegram_chat_id=COALESCE(system_groups.telegram_chat_id,NULLIF(EXCLUDED.telegram_chat_id,0)), resolution_status=CASE WHEN EXCLUDED.resolution_status='resolved' THEN 'resolved' WHEN EXCLUDED.resolution_status='identity_pending' AND system_groups.resolution_status='pending' THEN 'identity_pending' ELSE system_groups.resolution_status END, first_contributor_tenant_id=COALESCE(system_groups.first_contributor_tenant_id,EXCLUDED.first_contributor_tenant_id), first_contributor_tenant_name=CASE WHEN system_groups.first_contributor_tenant_id IS NULL THEN EXCLUDED.first_contributor_tenant_name ELSE system_groups.first_contributor_tenant_name END, member_count_updated_at=CASE WHEN EXCLUDED.member_count>0 THEN NOW() ELSE system_groups.member_count_updated_at END, member_count=CASE WHEN EXCLUDED.member_count>0 THEN EXCLUDED.member_count ELSE system_groups.member_count END, last_seen_at=NOW(), status=CASE WHEN system_groups.status='joined' THEN 'joined' ELSE EXCLUDED.status END, title=CASE WHEN COALESCE(system_groups.title,'')='' OR system_groups.title=system_groups.tme_link THEN EXCLUDED.title ELSE system_groups.title END, username=COALESCE(NULLIF(system_groups.username,''),EXCLUDED.username), gtype=CASE WHEN system_groups.gtype='unknown' THEN EXCLUDED.gtype ELSE system_groups.gtype END", sg.DedupKey(), sg.Source, sg.Title, sg.Username, sg.TmeLink, sg.GType, sg.MemberCount, sg.Status, sg.TelegramChatID, sg.ContributorTenantID)
		if err != nil {
			return err
		}
		if sg.TelegramChatID != 0 {
			_ = tx.QueryRow(ctx, "SELECT sg_id FROM system_groups WHERE telegram_chat_id=$1 LIMIT 1", sg.TelegramChatID).Scan(&sgID)
		} else {
			_ = tx.QueryRow(ctx, "SELECT sg_id FROM system_groups WHERE dedup_key=$1", sg.DedupKey()).Scan(&sgID)
		}
	}
	if sgID != uuid.Nil && sg.ContributorTenantID != uuid.Nil {
		link := normalizeSystemGroupLink(sg.TmeLink)
		if link == "" && sg.Username != "" {
			link = normalizeSystemGroupLink("https://t.me/" + strings.TrimPrefix(sg.Username, "@"))
		}
		if link != "" {
			linkType := "public"
			if strings.Contains(link, "/+") || strings.Contains(link, "/joinchat/") {
				linkType = "private_invite"
			}
			_, err = tx.Exec(ctx, "INSERT INTO system_group_links (sg_id,original_link,normalized_link,link_type,validity_status,source,last_verified_at) VALUES ($1,$2,$3,$4,CASE WHEN $6<>'' OR $7>0 OR $8<>0 THEN 'valid' ELSE 'pending' END,$5,CASE WHEN $6<>'' OR $7>0 OR $8<>0 THEN NOW() ELSE NULL END) ON CONFLICT (normalized_link) DO UPDATE SET validity_status=CASE WHEN EXCLUDED.validity_status='valid' THEN 'valid' ELSE system_group_links.validity_status END,last_verified_at=COALESCE(EXCLUDED.last_verified_at,system_group_links.last_verified_at)", sgID, sg.TmeLink, link, linkType, sg.Source, sg.Title, sg.MemberCount, sg.TelegramChatID)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, "INSERT INTO system_group_contributions (sg_id,tenant_id,tenant_name_snapshot,submitted_link,source) SELECT $1,tenant_id,name,$2,$3 FROM tenants WHERE tenant_id=$4 ON CONFLICT (sg_id,tenant_id,submitted_link) DO NOTHING", sgID, sg.TmeLink, sg.Source, sg.ContributorTenantID)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, "UPDATE system_groups SET first_contributor_tenant_id=first_contribution.tenant_id, first_contributor_tenant_name=first_contribution.tenant_name_snapshot FROM (SELECT tenant_id,tenant_name_snapshot FROM system_group_contributions WHERE sg_id=$1 AND tenant_id IS NOT NULL ORDER BY created_at ASC, contribution_id ASC LIMIT 1) AS first_contribution WHERE system_groups.sg_id=$1", sgID)
			if err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

func normalizeSystemGroupLink(value string) string {
	value = strings.TrimSpace(value)
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "https://") {
		value = value[len("https://"):]
	} else if strings.HasPrefix(lower, "http://") {
		value = value[len("http://"):]
	}
	value = strings.SplitN(value, "?", 2)[0]
	value = strings.SplitN(value, "#", 2)[0]
	value = strings.TrimRight(value, "/")
	parts := strings.SplitN(value, "/", 2)
	if len(parts) != 2 {
		return ""
	}
	host := strings.ToLower(strings.TrimPrefix(parts[0], "www."))
	if host != "t.me" && host != "telegram.me" && host != "telegram.dog" {
		return ""
	}
	path := parts[1]
	if strings.HasPrefix(path, "+") || strings.HasPrefix(path, "joinchat/") {
		// Invite hashes are case-sensitive; normalize only the host, never the hash.
		return "t.me/" + path
	}
	return "t.me/" + strings.ToLower(path)
}

// UpdateResolvedInfo 后台解析器回写: 私密邀请链接经 checkChatInvite 补全群名/成员数/类型。
func (r *SystemGroupRepository) UpdateResolvedInfo(ctx context.Context, sgID uuid.UUID, title string, memberCount int, gtype string, chatID int64) error {
	if chatID == 0 {
		_, err := r.db.Exec(ctx, "UPDATE system_groups SET title=CASE WHEN $2<>'' THEN $2 ELSE title END, member_count=CASE WHEN $3>0 THEN $3 ELSE member_count END, gtype=$4, resolution_status='identity_pending', member_count_updated_at=CASE WHEN $3>0 THEN NOW() ELSE member_count_updated_at END, last_seen_at=NOW() WHERE sg_id=$1", sgID, title, memberCount, gtype)
		return err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var canonicalID uuid.UUID
	lookupErr := tx.QueryRow(ctx, "SELECT sg_id FROM system_groups WHERE sg_id<>$1 AND (telegram_chat_id=$2 OR dedup_key=$3) ORDER BY created_at LIMIT 1 FOR UPDATE", sgID, chatID, fmt.Sprintf("c:%d", chatID)).Scan(&canonicalID)
	if lookupErr == nil {
		_, err = tx.Exec(ctx, "UPDATE system_groups SET telegram_chat_id=$2, dedup_key=$3, title=CASE WHEN $4<>'' THEN $4 ELSE title END, member_count=CASE WHEN $5>0 THEN $5 ELSE member_count END, gtype=$6, resolution_status='resolved', member_count_updated_at=CASE WHEN $5>0 THEN NOW() ELSE member_count_updated_at END, last_seen_at=NOW() WHERE sg_id=$1", canonicalID, chatID, fmt.Sprintf("c:%d", chatID), title, memberCount, gtype)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, "INSERT INTO system_group_links (sg_id,original_link,normalized_link,link_type,invite_hash,validity_status,source,last_verified_at,created_at) SELECT $1,original_link,normalized_link,link_type,invite_hash,validity_status,source,last_verified_at,created_at FROM system_group_links WHERE sg_id=$2 ON CONFLICT (normalized_link) DO NOTHING", canonicalID, sgID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, "INSERT INTO system_group_contributions (sg_id,tenant_id,tenant_name_snapshot,submitted_link,source,created_at) SELECT $1,tenant_id,tenant_name_snapshot,submitted_link,source,created_at FROM system_group_contributions WHERE sg_id=$2 ON CONFLICT (sg_id,tenant_id,submitted_link) DO NOTHING", canonicalID, sgID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, "DELETE FROM system_groups WHERE sg_id=$1", sgID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, "UPDATE system_groups SET first_contributor_tenant_id=first_contribution.tenant_id, first_contributor_tenant_name=first_contribution.tenant_name_snapshot FROM (SELECT tenant_id,tenant_name_snapshot FROM system_group_contributions WHERE sg_id=$1 AND tenant_id IS NOT NULL ORDER BY created_at,contribution_id LIMIT 1) AS first_contribution WHERE system_groups.sg_id=$1", canonicalID)
		if err != nil {
			return err
		}
		return tx.Commit(ctx)
	}

	_, err = tx.Exec(ctx, "UPDATE system_groups SET dedup_key=$2, telegram_chat_id=$3, title=CASE WHEN $4<>'' THEN $4 ELSE title END, member_count=GREATEST(member_count,$5), gtype=$6, resolution_status='resolved', member_count_updated_at=CASE WHEN $5>0 THEN NOW() ELSE member_count_updated_at END, last_seen_at=NOW() WHERE sg_id=$1", sgID, fmt.Sprintf("c:%d", chatID), chatID, title, memberCount, gtype)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// MarkInvalidByDedupKey R87-M10: 加入失败等场景把群标记为失效("已失效"筛选的数据源)。
// 已加入的群不降级。
func (r *SystemGroupRepository) MarkInvalidByDedupKey(ctx context.Context, key string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE system_groups SET status='invalid' WHERE dedup_key=$1 AND status != 'joined'`, key)
	return err
}

// GetByIDs 按主键批量取(批量加入用), 保持传入顺序无要求。
func (r *SystemGroupRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.SystemGroup, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT sg_id, source, COALESCE(title,''), COALESCE(username,''), COALESCE(tme_link,''),
		       gtype, member_count, COALESCE(free_speech_status,'unknown'), status, ref_count, last_seen_at, COALESCE(first_contributor_tenant_name,'')
		FROM system_groups WHERE sg_id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.SystemGroup
	for rows.Next() {
		var g domain.SystemGroup
		if err := rows.Scan(&g.SGID, &g.Source, &g.Title, &g.Username, &g.TmeLink, &g.GType, &g.MemberCount, &g.FreeSpeechStatus, &g.Status, &g.RefCount, &g.LastSeenAt, &g.FirstContributorTenantName); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := r.attachLinks(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

// List 分页查询
func (r *SystemGroupRepository) List(ctx context.Context, f SystemGroupFilter) ([]domain.SystemGroup, int64, error) {
	// Exclude Telegram bot usernames, including legacy rows imported before bot filtering.
	var where []string = []string{"(COALESCE(username,'') = '' OR username NOT ILIKE '%bot')"}
	var args []interface{}
	n := 0
	// 关键字三字段模糊(同一参数复用)
	if f.Keyword != "" {
		n++
		kw := "%" + f.Keyword + "%"
		where = append(where, fmt.Sprintf("(title ILIKE $%d OR username ILIKE $%d OR tme_link ILIKE $%d)", n, n, n))
		args = append(args, kw)
	}
	if f.GType != "" {
		n++
		where = append(where, fmt.Sprintf("gtype = $%d", n))
		args = append(args, f.GType)
	}
	if f.Status != "" {
		n++
		where = append(where, fmt.Sprintf("status = $%d", n))
		args = append(args, f.Status)
	}
	if f.Source != "" {
		n++
		where = append(where, fmt.Sprintf("source = $%d", n))
		args = append(args, f.Source)
	}
	w := ""
	if len(where) > 0 {
		w = "WHERE " + strings.Join(where, " AND ")
	}
	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM system_groups "+w, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	page, size := f.Page, f.PageSize
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	n++
	args = append(args, size)
	n++
	args = append(args, (page-1)*size)
	rows, err := r.db.Query(ctx, `
		SELECT sg_id, source, COALESCE(title,''), COALESCE(username,''), COALESCE(tme_link,''),
		       gtype, member_count, COALESCE(free_speech_status,'unknown'), status, ref_count, last_seen_at, COALESCE(first_contributor_tenant_name,'')
		FROM system_groups `+w+`
		ORDER BY last_seen_at DESC, sg_id DESC  -- 唯一tiebreaker: 防同值行间顺序不定导致翻页重复/遗漏
		LIMIT $`+fmt.Sprint(n-1)+` OFFSET $`+fmt.Sprint(n), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.SystemGroup
	for rows.Next() {
		var g domain.SystemGroup
		if err := rows.Scan(&g.SGID, &g.Source, &g.Title, &g.Username, &g.TmeLink, &g.GType, &g.MemberCount, &g.FreeSpeechStatus, &g.Status, &g.RefCount, &g.LastSeenAt, &g.FirstContributorTenantName); err != nil {
			return nil, 0, err
		}
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if err := r.attachLinks(ctx, out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// UpdateFreeSpeechStatusByChatID stores the shared default speaking rule for a
// Telegram group. It deliberately does not store account-specific mute state.
func (r *SystemGroupRepository) UpdateFreeSpeechStatusByChatID(ctx context.Context, chatID int64, status string) error {
	if chatID == 0 {
		return fmt.Errorf("telegram chat id is required")
	}
	if status != "allowed" && status != "restricted" && status != "unknown" {
		return fmt.Errorf("invalid free-speech status %q", status)
	}
	_, err := r.db.Exec(ctx, `
		UPDATE system_groups
		SET free_speech_status=$2, last_seen_at=NOW()
		WHERE telegram_chat_id=$1
	`, chatID, status)
	return err
}

// UpsertJoinedGroup merges a confirmed membership into the canonical shared
// group record, including a previously imported username/invite placeholder.
func (r *SystemGroupRepository) UpsertJoinedGroup(ctx context.Context, sg *domain.SystemGroup, freeSpeechStatus string) error {
	if sg == nil || sg.TelegramChatID == 0 {
		return fmt.Errorf("joined group identity is incomplete")
	}
	identity := domain.SystemGroup{Username: sg.Username, TmeLink: normalizeSystemGroupLink(sg.TmeLink)}
	dedupKey := identity.DedupKey()
	var existingID uuid.UUID
	err := r.db.QueryRow(ctx, `
		SELECT sg_id
		FROM system_groups
		WHERE telegram_chat_id=$1 OR dedup_key=$2
		ORDER BY CASE WHEN telegram_chat_id=$1 THEN 1 ELSE 0 END
		LIMIT 1
	`, sg.TelegramChatID, dedupKey).Scan(&existingID)
	if err == nil {
		if err := r.UpdateResolvedInfo(ctx, existingID, sg.Title, sg.MemberCount, sg.GType, sg.TelegramChatID); err != nil {
			return err
		}
		_, err = r.db.Exec(ctx, "UPDATE system_groups SET status='active' WHERE telegram_chat_id=$1", sg.TelegramChatID)
		if err != nil {
			return err
		}
	} else if err == pgx.ErrNoRows {
		if err := r.Upsert(ctx, sg); err != nil {
			return err
		}
	} else {
		return err
	}
	if freeSpeechStatus != "" && freeSpeechStatus != "unknown" {
		return r.UpdateFreeSpeechStatusByChatID(ctx, sg.TelegramChatID, freeSpeechStatus)
	}
	return nil
}
