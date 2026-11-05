package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
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
	var sgID uuid.UUID
	var err error
	if sg.TelegramChatID != 0 {
		lookupErr := r.db.QueryRow(ctx, "SELECT sg_id FROM system_groups WHERE telegram_chat_id=$1 LIMIT 1", sg.TelegramChatID).Scan(&sgID)
		if lookupErr == nil {
			_, err = r.db.Exec(ctx, "UPDATE system_groups SET dedup_key=CASE WHEN NOT EXISTS (SELECT 1 FROM system_groups other WHERE other.dedup_key='c:'||$2 AND other.sg_id<>$1) THEN 'c:'||$2 ELSE dedup_key END, title=CASE WHEN COALESCE(title,'')='' OR title=tme_link THEN $3 ELSE title END, username=COALESCE(NULLIF(username,''),$4), member_count=GREATEST(member_count,$5), gtype=CASE WHEN gtype='unknown' THEN $6 ELSE gtype END, resolution_status=CASE WHEN $3<>'' OR $5>0 THEN 'resolved' ELSE resolution_status END, member_count_updated_at=CASE WHEN $5>0 THEN NOW() ELSE member_count_updated_at END, first_contributor_tenant_id=COALESCE(first_contributor_tenant_id,NULLIF($7::text,'00000000-0000-0000-0000-000000000000')::uuid), first_contributor_tenant_name=CASE WHEN first_contributor_tenant_id IS NULL THEN COALESCE((SELECT name FROM tenants WHERE tenant_id=NULLIF($7::text,'00000000-0000-0000-0000-000000000000')::uuid),'') ELSE first_contributor_tenant_name END, last_seen_at=NOW() WHERE sg_id=$1", sgID, sg.TelegramChatID, sg.Title, sg.Username, sg.MemberCount, sg.GType, sg.ContributorTenantID)
			if err != nil { return err }
		} else {
			sgID = uuid.Nil
		}
	}
	if sgID == uuid.Nil {
		_, err = r.db.Exec(ctx, "INSERT INTO system_groups (dedup_key,source,title,username,tme_link,gtype,member_count,status,ref_count,last_seen_at,telegram_chat_id,resolution_status,member_count_updated_at,first_contributor_tenant_id,first_contributor_tenant_name) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,1,NOW(),NULLIF($9,0),CASE WHEN COALESCE($3,'')<>'' OR $7>0 THEN 'resolved' ELSE 'pending' END,CASE WHEN $7>0 THEN NOW() ELSE NULL END,NULLIF($10::text,'00000000-0000-0000-0000-000000000000')::uuid,COALESCE((SELECT name FROM tenants WHERE tenant_id=NULLIF($10::text,'00000000-0000-0000-0000-000000000000')::uuid),'')) ON CONFLICT (dedup_key) DO UPDATE SET ref_count=system_groups.ref_count+1, telegram_chat_id=COALESCE(system_groups.telegram_chat_id,NULLIF(EXCLUDED.telegram_chat_id,0)), resolution_status=CASE WHEN EXCLUDED.resolution_status='resolved' THEN 'resolved' ELSE system_groups.resolution_status END, first_contributor_tenant_id=COALESCE(system_groups.first_contributor_tenant_id,EXCLUDED.first_contributor_tenant_id), first_contributor_tenant_name=CASE WHEN system_groups.first_contributor_tenant_id IS NULL THEN EXCLUDED.first_contributor_tenant_name ELSE system_groups.first_contributor_tenant_name END, member_count_updated_at=CASE WHEN EXCLUDED.member_count>0 THEN NOW() ELSE system_groups.member_count_updated_at END, member_count=GREATEST(system_groups.member_count,EXCLUDED.member_count), last_seen_at=NOW(), status=CASE WHEN system_groups.status='joined' THEN 'joined' ELSE EXCLUDED.status END, title=CASE WHEN COALESCE(system_groups.title,'')='' OR system_groups.title=system_groups.tme_link THEN EXCLUDED.title ELSE system_groups.title END, username=COALESCE(NULLIF(system_groups.username,''),EXCLUDED.username), gtype=CASE WHEN system_groups.gtype='unknown' THEN EXCLUDED.gtype ELSE system_groups.gtype END", sg.DedupKey(), sg.Source, sg.Title, sg.Username, sg.TmeLink, sg.GType, sg.MemberCount, sg.Status, sg.TelegramChatID, sg.ContributorTenantID)
		if err != nil { return err }
		if sg.TelegramChatID != 0 {
			_ = r.db.QueryRow(ctx, "SELECT sg_id FROM system_groups WHERE telegram_chat_id=$1 LIMIT 1", sg.TelegramChatID).Scan(&sgID)
		} else {
			_ = r.db.QueryRow(ctx, "SELECT sg_id FROM system_groups WHERE dedup_key=$1", sg.DedupKey()).Scan(&sgID)
		}
	}
	if sgID != uuid.Nil && sg.ContributorTenantID != uuid.Nil {
		link := normalizeSystemGroupLink(sg.TmeLink)
		if link == "" && sg.Username != "" { link = normalizeSystemGroupLink("https://t.me/" + strings.TrimPrefix(sg.Username, "@")) }
		if link != "" {
			linkType := "public"
			if strings.Contains(link, "/+") || strings.Contains(link, "/joinchat/") { linkType = "private_invite" }
			_, err = r.db.Exec(ctx, "INSERT INTO system_group_links (sg_id,original_link,normalized_link,link_type,validity_status,source) VALUES ($1,$2,$3,$4,'pending',$5) ON CONFLICT (normalized_link) DO NOTHING", sgID, sg.TmeLink, link, linkType, sg.Source)
			if err != nil { return err }
			_, err = r.db.Exec(ctx, "INSERT INTO system_group_contributions (sg_id,tenant_id,tenant_name_snapshot,submitted_link,source) SELECT $1,tenant_id,name,$2,$3 FROM tenants WHERE tenant_id=$4 ON CONFLICT (sg_id,tenant_id,submitted_link) DO NOTHING", sgID, sg.TmeLink, sg.Source, sg.ContributorTenantID)
			if err != nil { return err }
		}
	}
	return nil
}

func normalizeSystemGroupLink(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "https://")
	value = strings.TrimPrefix(value, "http://")
	value = strings.TrimPrefix(value, "www.")
	value = strings.TrimRight(value, "/")
	if value == "" { return "" }
	if !strings.HasPrefix(value, "t.me/") && !strings.HasPrefix(value, "telegram.me/") && !strings.HasPrefix(value, "telegram.dog/") { return "" }
	return value
}

// UpdateResolvedInfo 后台解析器回写: 私密邀请链接经 checkChatInvite 补全群名/成员数/类型。
func (r *SystemGroupRepository) UpdateResolvedInfo(ctx context.Context, sgID uuid.UUID, title string, memberCount int, gtype string, chatID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE system_groups SET title=$2, member_count=GREATEST(member_count, $3), gtype=$4, telegram_chat_id=COALESCE(telegram_chat_id,NULLIF($5,0)), resolution_status='resolved', member_count_updated_at=CASE WHEN $3>0 THEN NOW() ELSE member_count_updated_at END, last_seen_at=NOW()
		WHERE sg_id=$1`, sgID, title, memberCount, gtype, chatID)
	return err
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
		       gtype, member_count, status, ref_count, last_seen_at, COALESCE(first_contributor_tenant_name,'')
		FROM system_groups WHERE sg_id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.SystemGroup
	for rows.Next() {
		var g domain.SystemGroup
		if err := rows.Scan(&g.SGID, &g.Source, &g.Title, &g.Username, &g.TmeLink, &g.GType, &g.MemberCount, &g.Status, &g.RefCount, &g.LastSeenAt, &g.FirstContributorTenantName); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// List 分页查询
func (r *SystemGroupRepository) List(ctx context.Context, f SystemGroupFilter) ([]domain.SystemGroup, int64, error) {
	var where []string
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
		       gtype, member_count, status, ref_count, last_seen_at, COALESCE(first_contributor_tenant_name,'')
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
		if err := rows.Scan(&g.SGID, &g.Source, &g.Title, &g.Username, &g.TmeLink, &g.GType, &g.MemberCount, &g.Status, &g.RefCount, &g.LastSeenAt, &g.FirstContributorTenantName); err != nil {
			return nil, 0, err
		}
		out = append(out, g)
	}
	return out, total, rows.Err()
}
