// Package grouplib 平台级群组库聚合器(找群功能的后端)。
// 三路来源, 每轮增量归并进 system_groups:
//  1. scraped_links —— 所有租户链接采集到的私密/公开群链接
//  2. groups        —— 所有租户已加入且带 username 的公开群
//  3. members.bio   —— 群成员简介里挖出的 t.me 链接/@用户名(简介采集)
package grouplib

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"tg-sender/internal/domain"
	"tg-sender/internal/repository"
	"tg-sender/internal/safego"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// 简介挖掘正则
var (
	tmeRe = regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?(?:t\.me|telegram\.me|telegram\.dog)/(joinchat/|\+)?([A-Za-z0-9_\-]{4,64})`)
	// V28 审计修复: 原来会匹配邮箱(@example.com 的 @example)当 TG 用户名采集。
	// 排除逻辑在使用处做(emailTLDs 黑名单)——RE2 引擎不支持负向断言 (?!...),
	// 上一版直接写进正则导致 init() 时 MustCompile panic, 容器起不来(2026-10-23 运行时实证)。
	atRe = regexp.MustCompile(`(?m)@([A-Za-z][A-Za-z0-9_]{3,31})`)
)

// emailTLDs: @用户名 后跟这些域名段的视为邮箱地址, 不当 TG 用户名采集
var emailTLDs = []string{".com", ".net", ".org", ".cn", ".io", ".me", ".edu", ".gov", ".info", ".biz", ".cc", ".xyz"}

type aggregationCursor struct {
	at time.Time
	id uuid.UUID
}

type Engine struct {
	repo     *repository.SystemGroupRepository
	pool     *pgxpool.Pool
	acc      accountEngine // 借账号做关键词搜群(由 container 注入)
	interval time.Duration
	stopCh   chan struct{}
	wg       sync.WaitGroup

	lifecycleMu sync.Mutex
	started     bool
	stopped     bool

	mu sync.Mutex

	resolveCache   map[string]cachedImportResolution
	resolveFlights map[string]*importResolveCall

	// workerImportOverride is used by deterministic PostgreSQL worker tests.
	workerImportOverride func(context.Context, uuid.UUID, uuid.UUID, []string) (*ImportResult, error)

	slW aggregationCursor // scraped_links 复合游标
	gW  aggregationCursor // groups 复合游标
	mW  aggregationCursor // members 复合游标

	// kickCh 由写入路径(链接采集落库)触发，让聚合不必傻等 2 分钟轮询。
	// 带去抖：两次触发之间至少隔 mergeKickMinGap，避免高频落库把扫描打爆。
	kickCh  chan struct{}
	lastRun atomic.Int64 // unix nano，供去抖判断
}

// mergeKickMinGap 是"被唤醒"后的最小间隔。落库可能每秒一批，而一次聚合要扫三路来源，
// 15 秒足够把突发批量合并成一轮，又不会明显拖慢新链接可见时间。
const mergeKickMinGap = 15 * time.Second

// TriggerMerge 请求尽快跑一轮聚合(非阻塞，已排队时直接丢弃)。
// 调用方：链接采集批量落库成功后。聚合循环本身仍在跑 2 分钟兜底轮询，
// 所以即使这里丢了事件也只是晚一点，不会漏。
func (e *Engine) TriggerMerge() {
	select {
	case e.kickCh <- struct{}{}:
	default:
	}
}

func (e *Engine) loadAggregationCursor(ctx context.Context, source string, local aggregationCursor) (aggregationCursor, error) {
	var stored aggregationCursor
	err := e.pool.QueryRow(ctx, `
		SELECT cursor_at, cursor_id
		FROM group_library_aggregation_cursors
		WHERE source=$1`, source).Scan(&stored.at, &stored.id)
	if errors.Is(err, pgx.ErrNoRows) {
		return local, nil
	}
	if err != nil {
		return aggregationCursor{}, err
	}
	if stored.at.After(local.at) || (stored.at.Equal(local.at) && stored.id.String() > local.id.String()) {
		return stored, nil
	}
	return local, nil
}

func (e *Engine) saveAggregationCursor(ctx context.Context, source string, cursor aggregationCursor) error {
	if cursor.id == uuid.Nil {
		return nil
	}
	_, err := e.pool.Exec(ctx, `
		INSERT INTO group_library_aggregation_cursors (source, cursor_at, cursor_id, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (source) DO UPDATE
		SET cursor_at=EXCLUDED.cursor_at, cursor_id=EXCLUDED.cursor_id, updated_at=NOW()
		WHERE (group_library_aggregation_cursors.cursor_at, group_library_aggregation_cursors.cursor_id)
		    < (EXCLUDED.cursor_at, EXCLUDED.cursor_id)`, source, cursor.at, cursor.id)
	return err
}

const groupLibraryAggregationLockKey int64 = 0x544753454E444552

// acquireAggregationLock serializes durable source scans across all application
// replicas. The cursor is shared in PostgreSQL, so concurrent scanners must not
// advance it past another replica's failed batch.
func (e *Engine) acquireAggregationLock(ctx context.Context) (func(), bool, error) {
	conn, err := e.pool.Acquire(ctx)
	if err != nil {
		return nil, false, err
	}
	// A transaction-scoped lock is automatically released if the connection or
	// process is interrupted; a failed explicit unlock cannot poison the pool.
	tx, err := conn.Begin(ctx)
	if err != nil {
		conn.Release()
		return nil, false, err
	}
	var acquired bool
	if err := tx.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock($1)`, groupLibraryAggregationLockKey).Scan(&acquired); err != nil {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = tx.Rollback(rollbackCtx)
		cancel()
		conn.Release()
		return nil, false, err
	}
	if !acquired {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = tx.Rollback(rollbackCtx)
		cancel()
		conn.Release()
		return nil, false, nil
	}
	unlock := func() {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tx.Rollback(rollbackCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			log.Printf("[grouplib] release aggregation advisory transaction: %v", err)
			// Do not return a potentially lock-holding connection to the pool.
			closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = conn.Hijack().Close(closeCtx)
			closeCancel()
			return
		}
		conn.Release()
	}
	return unlock, true, nil
}
func NewEngine(pool *pgxpool.Pool, repo *repository.SystemGroupRepository, acc accountEngine) *Engine {
	return &Engine{repo: repo, pool: pool, acc: acc, interval: 2 * time.Minute,
		stopCh: make(chan struct{}), kickCh: make(chan struct{}, 1)}
}

func (e *Engine) Start() {
	e.lifecycleMu.Lock()
	if e.started || e.stopped {
		e.lifecycleMu.Unlock()
		return
	}
	// Register all background goroutines while holding the lifecycle lock so
	// Stop cannot begin Wait before these Add calls are complete.
	e.started = true
	e.wg.Add(2)
	e.lifecycleMu.Unlock()

	// 聚合任务保持低频运行，避免影响主业务数据库。
	safego.Go(func() {
		defer e.wg.Done()
		t := time.NewTicker(e.interval)
		defer t.Stop()
		e.runOnce()
		for {
			select {
			case <-e.stopCh:
				return
			case <-e.kickCh:
				if e.mergeKickAllowed() {
					e.runOnce()
				}
			case <-t.C:
				e.runOnce()
			}
		}
	})

	// 用户确认的导入任务使用独立调度周期，不再等待两分钟一次的全量聚合周期。
	safego.Go(func() {
		defer e.wg.Done()
		ctx, cancel := context.WithCancel(context.Background())
		stopWatchDone := make(chan struct{})
		safego.Go(func() {
			select {
			case <-e.stopCh:
				cancel()
			case <-stopWatchDone:
			}
		})
		defer func() {
			close(stopWatchDone)
			cancel()
		}()

		e.processImportJobs(ctx)
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-e.stopCh:
				return
			case <-ctx.Done():
				return
			case <-t.C:
				e.processImportJobs(ctx)
			}
		}
	})
}

func (e *Engine) Stop() {
	e.lifecycleMu.Lock()
	if !e.stopped {
		e.stopped = true
		close(e.stopCh)
	}
	e.lifecycleMu.Unlock()
	e.wg.Wait()
}

// mergeKickAllowed 判断被唤醒的这一轮是否值得马上跑（去抖）。
func (e *Engine) mergeKickAllowed() bool {
	last := e.lastRun.Load()
	if last == 0 {
		return true
	}
	return time.Since(time.Unix(0, last)) >= mergeKickMinGap
}

func (e *Engine) runOnce() {
	e.lastRun.Store(time.Now().UnixNano())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	done := make(chan struct{})
	safego.Go(func() {
		select {
		case <-e.stopCh:
			cancel()
		case <-done:
		}
	})
	defer close(done)
	defer cancel()
	unlock, acquired, lockErr := e.acquireAggregationLock(ctx)
	if lockErr != nil {
		log.Printf("[grouplib] acquire aggregation advisory lock: %v", lockErr)
		return
	}
	if !acquired {
		// Another replica owns the shared cursor scan; it will persist progress.
		return
	}
	defer unlock()

	var errs []error
	if err := e.aggregateScrapedLinks(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := e.aggregateJoinedGroups(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := e.aggregateMemberBios(ctx); err != nil {
		errs = append(errs, err)
	}
	// 批量导入的私密链接在此低速补群名(每轮最多 resolveBatchSize 条)
	if err := e.resolvePendingTitles(ctx); err != nil {
		errs = append(errs, err)
	}
	for _, err := range errs {
		log.Printf("[grouplib] 聚合失败: %v", err)
	}
}

// aggregateScrapedLinks 增量聚合链接采集结果
func (e *Engine) aggregateScrapedLinks(ctx context.Context) error {
	e.mu.Lock()
	cursor := e.slW
	e.mu.Unlock()
	var cursorErr error
	cursor, cursorErr = e.loadAggregationCursor(ctx, "scraped_links", cursor)
	if cursorErr != nil {
		return fmt.Errorf("load scraped_links aggregation cursor: %w", cursorErr)
	}
	// 复合游标以 link_id 作为同时间戳的唯一排序键，避免 LIMIT 截断后重复首批。
	rows, err := e.pool.Query(ctx, `
		SELECT link_id, tenant_id, full_url, link_type, COALESCE(NULLIF(resolved_title,''), NULLIF(source_chat_title,''), ''),
		       member_count, status, created_at, COALESCE(resolved_chat_id,0)
		FROM scraped_links WHERE (created_at, link_id) > ($1, $2)
		ORDER BY created_at ASC, link_id ASC LIMIT 5000`, cursor.at, cursor.id)
	if err != nil {
		return err
	}
	defer rows.Close()
	var last aggregationCursor
	n := 0
	upsertFailed := false
	for rows.Next() {
		var linkID, tenantID uuid.UUID
		var url, linkType, title, status string
		var mc int
		var chatID int64
		var ct time.Time
		if err := rows.Scan(&linkID, &tenantID, &url, &linkType, &title, &mc, &status, &ct, &chatID); err != nil {
			return err
		}
		username := extractTmeUsername(url)
		if strings.HasSuffix(strings.ToLower(username), "bot") {
			last = aggregationCursor{at: ct, id: linkID}
			continue
		}
		gt := "public"
		if linkType == "private" {
			gt = "private"
		}
		st := "active"
		sg := &domain.SystemGroup{Source: "scraped", Title: firstNonEmpty(title, url), Username: username, TmeLink: url, GType: gt, MemberCount: mc, Status: st, TelegramChatID: chatID, ContributorTenantID: tenantID}
		if err := e.repo.Upsert(ctx, sg); err != nil {
			upsertFailed = true
			log.Printf("[grouplib] scraped upsert %s: %v", url, err)
			break
		}
		last = aggregationCursor{at: ct, id: linkID}
		n++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if upsertFailed {
		if last.id != uuid.Nil {
			if err := e.saveAggregationCursor(ctx, "scraped_links", last); err != nil {
				return fmt.Errorf("save scraped_links aggregation cursor: %w", err)
			}
			e.mu.Lock()
			e.slW = last
			e.mu.Unlock()
		}
		return errors.New("scraped links aggregation incomplete; cursor stopped before failed row")
	}
	if last.id != uuid.Nil {
		if err := e.saveAggregationCursor(ctx, "scraped_links", last); err != nil {
			return fmt.Errorf("save scraped_links aggregation cursor: %w", err)
		}
		e.mu.Lock()
		e.slW = last
		e.mu.Unlock()
	}
	if n > 0 {
		log.Printf("[grouplib] scraped 归并 %d 条", n)
	}
	return nil
}

// aggregateJoinedGroups 增量聚合所有租户已加入的公开群(私有群无 username, 别人也进不去, 不入库)
func (e *Engine) aggregateJoinedGroups(ctx context.Context) error {
	e.mu.Lock()
	cursor := e.gW
	e.mu.Unlock()
	var cursorErr error
	cursor, cursorErr = e.loadAggregationCursor(ctx, "groups", cursor)
	if cursorErr != nil {
		return fmt.Errorf("load groups aggregation cursor: %w", cursorErr)
	}
	// updated_at(065 迁移 + trigger) 负责捕获群资料更新，group_id 负责稳定分页。
	rows, err := e.pool.Query(ctx, `
		SELECT group_id, tenant_id, title, username, type, member_count, created_at, updated_at
		FROM groups
		WHERE is_joined = TRUE AND username IS NOT NULL AND username <> ''
		  AND COALESCE(type, '') <> 'channel'
		  AND (GREATEST(created_at, updated_at), group_id) > ($1, $2)
		ORDER BY GREATEST(created_at, updated_at) ASC, group_id ASC LIMIT 5000`, cursor.at, cursor.id)
	if err != nil {
		return err
	}
	defer rows.Close()
	var last aggregationCursor
	n := 0
	upsertFailed := false
	for rows.Next() {
		var groupID, tenantID uuid.UUID
		var title, username, gtype string
		var mc int
		var ct, ut time.Time
		if err := rows.Scan(&groupID, &tenantID, &title, &username, &gtype, &mc, &ct, &ut); err != nil {
			return err
		}
		if ut.After(ct) {
			ct = ut
		}
		gt := "public"
		switch strings.ToLower(strings.TrimSpace(gtype)) {
		case "channel":
			gt = "channel"
		case "supergroup", "megagroup":
			gt = "supergroup"
		}
		sg := &domain.SystemGroup{
			Source: "joined", Title: firstNonEmpty(title, "@"+username), Username: username,
			TmeLink: "https://t.me/" + username, GType: gt, MemberCount: mc, Status: "active",
			ContributorTenantID: tenantID,
		}
		if err := e.repo.Upsert(ctx, sg); err != nil {
			upsertFailed = true
			log.Printf("[grouplib] joined upsert %s: %v", username, err)
			break
		}
		last = aggregationCursor{at: ct, id: groupID}
		n++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if upsertFailed {
		if last.id != uuid.Nil {
			if err := e.saveAggregationCursor(ctx, "groups", last); err != nil {
				return fmt.Errorf("save groups aggregation cursor: %w", err)
			}
			e.mu.Lock()
			e.gW = last
			e.mu.Unlock()
		}
		return errors.New("joined groups aggregation incomplete; cursor stopped before failed row")
	}
	if last.id != uuid.Nil {
		if err := e.saveAggregationCursor(ctx, "groups", last); err != nil {
			return fmt.Errorf("save groups aggregation cursor: %w", err)
		}
		e.mu.Lock()
		e.gW = last
		e.mu.Unlock()
	}
	if n > 0 {
		log.Printf("[grouplib] joined 归并 %d 条", n)
	}
	return nil
}

// aggregateMemberBios 简介挖掘: 从成员 bio 里解析 t.me 链接和 @用户名
func (e *Engine) aggregateMemberBios(ctx context.Context) error {
	e.mu.Lock()
	cursor := e.mW
	e.mu.Unlock()
	var cursorErr error
	cursor, cursorErr = e.loadAggregationCursor(ctx, "members_bio", cursor)
	if cursorErr != nil {
		return fmt.Errorf("load members_bio aggregation cursor: %w", cursorErr)
	}
	rows, err := e.pool.Query(ctx, `
		SELECT member_id, tenant_id, bio, scraped_at FROM members
		WHERE (scraped_at, member_id) > ($1, $2) AND bio IS NOT NULL AND bio <> ''
		  AND (bio ILIKE '%t.me%' OR bio ILIKE '%telegram%' OR bio ~* '(^|[^A-Za-z0-9_])@[A-Za-z][A-Za-z0-9_]{3,31}($|[^A-Za-z0-9_.-])')
		ORDER BY scraped_at ASC, member_id ASC LIMIT 5000`, cursor.at, cursor.id)
	if err != nil {
		return err
	}
	defer rows.Close()
	var last aggregationCursor
	n := 0
	for rows.Next() {
		var memberID, tenantID uuid.UUID
		var bio string
		var ct time.Time
		if err := rows.Scan(&memberID, &tenantID, &bio, &ct); err != nil {
			return err
		}
		processed, bioErr := e.upsertBioLinks(ctx, bio, tenantID)
		n += processed
		if bioErr != nil {
			return bioErr
		}
		last = aggregationCursor{at: ct, id: memberID}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if last.id != uuid.Nil {
		if err := e.saveAggregationCursor(ctx, "members_bio", last); err != nil {
			return fmt.Errorf("save members_bio aggregation cursor: %w", err)
		}
		e.mu.Lock()
		e.mW = last
		e.mu.Unlock()
	}
	if n > 0 {
		log.Printf("[grouplib] bio 挖掘归并 %d 条", n)
	}
	return nil
}

func (e *Engine) upsertBioLinks(ctx context.Context, bio string, tenantID uuid.UUID) (int, error) {
	n := 0
	seen := map[string]bool{}
	for _, m := range tmeRe.FindAllStringSubmatch(bio, -1) {
		isJoin := m[1] != ""
		key := m[2]
		if !isJoin && strings.HasSuffix(strings.ToLower(key), "bot") {
			continue
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		sg := &domain.SystemGroup{Source: "bio", ContributorTenantID: tenantID}
		if isJoin {
			sg.TmeLink = "https://t.me/joinchat/" + key
			sg.GType = "private"
			sg.Title = sg.TmeLink
		} else {
			sg.Username = key
			sg.TmeLink = "https://t.me/" + key
			sg.GType = "public"
			sg.Title = "@" + key
		}
		if err := e.repo.Upsert(ctx, sg); err != nil {
			log.Printf("[grouplib] bio upsert %s: %v", key, err)
			return n, err
		}
		n++
	}
	// 裸 @用户名(前面不是链接的一部分)——保守: 只收出现在行首或空白后的
	// V28 排除逻辑(RE2 无负向断言, 代码层过滤): @用户名 后跟邮箱域名段的跳过
	for _, m := range atRe.FindAllStringSubmatchIndex(bio, -1) {
		key := bio[m[2]:m[3]]
		if strings.HasSuffix(strings.ToLower(key), "bot") {
			continue
		}
		if isEmailAddressMention(bio, m[0], m[1]) {
			continue
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		sg := &domain.SystemGroup{Source: "bio", Username: key, TmeLink: "https://t.me/" + key, GType: "public", Title: "@" + key, ContributorTenantID: tenantID}
		if err := e.repo.Upsert(ctx, sg); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// extractTmeUsername 从任意 url/文本里提取 t.me 用户名(私密链接返回空, 走 l: 链接键归并)
func extractTmeUsername(url string) string {
	m := tmeRe.FindStringSubmatch(url)
	if m == nil || m[1] != "" {
		return ""
	}
	return m[2]
}

// isEmailAddressMention rejects @ tokens that are part of an email address.
// It checks the local-part boundary (independent of TLD) and retains a
// conservative domain-suffix fallback for standalone email-like tokens.
func isEmailAddressMention(bio string, atStart, atEnd int) bool {
	if atStart > 0 {
		prev := bio[atStart-1]
		if (prev >= 'a' && prev <= 'z') || (prev >= 'A' && prev <= 'Z') ||
			(prev >= '0' && prev <= '9') || strings.ContainsRune("._%+-", rune(prev)) {
			return true
		}
	}
	j := atEnd
	for j < len(bio) && (bio[j] == '.' || bio[j] == '-' ||
		(bio[j] >= 'a' && bio[j] <= 'z') || (bio[j] >= 'A' && bio[j] <= 'Z') ||
		(bio[j] >= '0' && bio[j] <= '9')) {
		j++
	}
	tail := strings.ToLower(bio[atEnd:j])
	for _, tld := range emailTLDs {
		if strings.Contains(tail, tld) {
			return true
		}
	}
	return false
}

func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if v != "" {
			return v
		}
	}
	return ""
}

// 防 unused 误报(bio 里可能只有 @ 没有链接时 strings 仍被用到)
var _ = strings.TrimSpace
