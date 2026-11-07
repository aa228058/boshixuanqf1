package grouplib

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/tg"
	"tg-sender/internal/domain"
	"tg-sender/internal/repository"
)

// importLinkRe 保留为兼容性测试用的候选识别规则；生产导入统一走 filterImportLinks。
// 与 engine.go 的 tmeRe 同款口径, 避免两种采集规则不一致。
var telegramFloodWaitRe = regexp.MustCompile(`(?i)FLOOD_WAIT_(\d+)`)

func telegramMTProtoRateLimit(err error) error {
	if err == nil {
		return nil
	}
	match := telegramFloodWaitRe.FindStringSubmatch(err.Error())
	if len(match) != 2 {
		return nil
	}
	seconds, parseErr := strconv.Atoi(match[1])
	if parseErr != nil || seconds < 1 {
		seconds = 1
	}
	return &TelegramRateLimitError{RetryAfter: time.Duration(seconds) * time.Second, Endpoint: "Telegram MTProto"}
}

var importLinkRe = regexp.MustCompile(`(?i)(?:(?:https?://)?(?:www\.)?(?:t\.me|telegram\.me|telegram\.dog)/(s/|joinchat/|\+)?|@)([+a-zA-Z0-9_\-]{5,64})`)

// ImportResult 批量导入结果汇总。
type ImportResult struct {
	Processed int                  // 已完成解析的链接数；遇到 429 的当前链接不计入
	Imported  []domain.SystemGroup // 成功解析并入群库的条目
	Failed    []string             // 解析失败的原因(如 "xxx: 群组不存在")
	Skipped   int                  // 重复/无效的原始行数
}

// ImportByLinks 批量导入: 逐条解析链接→调 TG 补全群名/成员数/类型→归并入群库。
//
// 解析规则：只有成功取得群组身份信息后才允许写入正式群组库。
// 私密邀请链接如果 Telegram 不提供预览信息，保留为失败/待补充，不创建占位群组。
func (e *Engine) ImportByLinks(ctx context.Context, tenantID, accountID uuid.UUID, lines []string) (*ImportResult, error) {
	return e.importByLinks(ctx, tenantID, accountID, lines, false)
}

// importByLinks optionally prefers the shared web resolver for large background jobs, using MTProto for private invites and web fallbacks.
func (e *Engine) importByLinks(ctx context.Context, tenantID, accountID uuid.UUID, lines []string, preferWeb bool) (*ImportResult, error) {
	// Direct callers and queued jobs must share the exact same strict parser.
	// This prevents bypassing token-boundary, bot, and URL validation checks.
	type entry struct {
		private  bool
		username string
		hash     string
		link     string
	}
	filtered := filterImportLinks(strings.Join(lines, "\n"))
	entries := make([]entry, 0, len(filtered.Links))
	for _, link := range filtered.Links {
		target := strings.TrimPrefix(link, "https://t.me/")
		if strings.HasPrefix(target, "+") {
			hash := strings.TrimPrefix(target, "+")
			entries = append(entries, entry{private: true, hash: hash, link: "t.me/+" + hash})
		} else {
			entries = append(entries, entry{username: target, link: link})
		}
	}
	skipped := filtered.Duplicate + filtered.Invalid
	if len(entries) == 0 {
		return &ImportResult{Skipped: skipped}, fmt.Errorf("没有解析到任何有效链接(支持 t.me/xxx、@xxx、t.me/+邀请hash)")
	}

	var client *telegram.Client
	if accountID != uuid.Nil {
		var err error
		client, err = e.acc.GetClient(ctx, tenantID, accountID)
		if err != nil {
			return nil, fmt.Errorf("取账号连接失败: %w", err)
		}
		defer e.acc.ReleaseClient(tenantID, accountID, client)
	}

	res := &ImportResult{Skipped: skipped}
	pendingGroups := make([]*domain.SystemGroup, 0, len(entries))
	flushPending := func(writeCtx context.Context) {
		if len(pendingGroups) == 0 {
			return
		}
		if err := e.repo.UpsertBatch(writeCtx, pendingGroups); err == nil {
			for _, group := range pendingGroups {
				res.Imported = append(res.Imported, *group)
			}
			pendingGroups = pendingGroups[:0]
			return
		}
		// Preserve per-link result reporting if the atomic batch fails; the batch transaction has rolled back.
		for _, group := range pendingGroups {
			if err := upsertImportGroupWithRetry(writeCtx, e.repo, group); err != nil {
				res.Failed = append(res.Failed, fmt.Sprintf("[database] %s: 入库失败 %v", group.TmeLink, err))
			} else {
				res.Imported = append(res.Imported, *group)
			}
		}
		pendingGroups = pendingGroups[:0]
	}
	// 大批量或无账号时改用共享 HTTP 解析器，仍然只允许完整解析后正式入库。
	bulkMode := preferWeb || len(entries) > 200 || client == nil
	for _, en := range entries {
		if err := ctx.Err(); err != nil {
			writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			flushPending(writeCtx)
			cancel()
			return res, err
		}
		sg := domain.SystemGroup{Source: "import", Status: "active", TmeLink: en.link, ContributorTenantID: tenantID}
		if en.private {
			sg.GType = "private"
		} else {
			sg.Username = en.username
			sg.GType = "unknown"
		}
		var title string
		var count int
		var gtype string
		var chatID int64
		var rerr error
		cacheKey := "u:" + strings.ToLower(en.username)
		if en.private {
			cacheKey = "p:" + en.hash // Telegram invite hashes are case-sensitive.
		}
		// MTProto results carry a real Chat ID; keep them separate from web previews.
		if client != nil && (!bulkMode || en.private) {
			cacheKey += ":mtproto"
		} else {
			cacheKey += ":web"
		}
		title, count, gtype, chatID, rerr = e.resolveCachedImportTarget(ctx, cacheKey, func() (string, int, string, int64, error) {
			// 私密邀请链接优先使用 MTProto：checkChatInvite 可读取 Telegram 提供的预览，
			// 不要求账号加入群组。未选择账号时才退回网页预览。
			if client != nil && en.private {
				return resolveInviteInfo(ctx, client, en.hash)
			}
			if client != nil && !bulkMode {
				return resolveGroupInfo(ctx, client, en.username)
			}
			// 大批量公开链接优先走共享网页调度器，避免把一个账号压成逐条串行瓶颈。
			title, count, gtype, webErr := resolveWebEntryWithRetry(ctx, en.private, en.username, en.hash)
			if webErr == nil || client == nil {
				return title, count, gtype, 0, webErr
			}
			var rateLimitErr *TelegramRateLimitError
			if errors.As(webErr, &rateLimitErr) || ctx.Err() != nil {
				return "", 0, "", 0, webErr
			}
			// 网页预览不可用/信息不完整时，用用户明确选择的在线账号补充 MTProto 解析。
			return resolveGroupInfo(ctx, client, en.username)
		})
		if ctx.Err() != nil {
			writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			flushPending(writeCtx)
			cancel()
			return res, ctx.Err()
		}
		if rerr == nil && gtype == "user" {
			rerr = fmt.Errorf("链接指向用户或机器人，不是群组/频道")
		}
		if rerr != nil || strings.TrimSpace(title) == "" {
			if rerr == nil {
				rerr = fmt.Errorf("解析结果缺少群组名称，暂不入库")
			}
			// FloodWait/429 表示当前链接尚未完成，不写入失败清单；冷却后由 Worker 从此处继续。
			var rateLimitErr *TelegramRateLimitError
			if errors.As(rerr, &rateLimitErr) {
				flushPending(ctx)
				return res, rateLimitErr
			}
			res.Failed = append(res.Failed, fmt.Sprintf("[%s] %s: %v", classifyImportFailure(rerr), en.link, rerr))
			res.Processed++
			continue
		}
		sg.Title, sg.MemberCount, sg.GType, sg.TelegramChatID = title, count, gtype, chatID
		pendingGroups = append(pendingGroups, &sg)
		res.Processed++
		// MTProto 请求保守限速；HTTP 解析由连接池与 Worker 调度统一控制。
		if client != nil && (!bulkMode || en.private || rerr != nil) {
			select {
			case <-ctx.Done():
				writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
				flushPending(writeCtx)
				cancel()
				return res, ctx.Err()
			case <-time.After(time.Second):
			}
		}
	}
	flushPending(ctx)
	return res, nil
}

const importResolutionCacheTTL = 10 * time.Minute
const importResolutionCacheLimit = 5000

type cachedImportResolution struct {
	title     string
	count     int
	gtype     string
	chatID    int64
	expiresAt time.Time
}

type importResolveCall struct {
	done chan struct{}
}

func (e *Engine) resolveCachedImportTarget(ctx context.Context, key string, resolve func() (string, int, string, int64, error)) (string, int, string, int64, error) {
	for {
		e.mu.Lock()
		now := time.Now()
		if cached, ok := e.resolveCache[key]; ok {
			if now.Before(cached.expiresAt) {
				e.mu.Unlock()
				return cached.title, cached.count, cached.gtype, cached.chatID, nil
			}
			delete(e.resolveCache, key)
		}
		if e.resolveFlights == nil {
			e.resolveFlights = make(map[string]*importResolveCall)
		}
		if call := e.resolveFlights[key]; call != nil {
			done := call.done
			e.mu.Unlock()
			select {
			case <-ctx.Done():
				return "", 0, "unknown", 0, ctx.Err()
			case <-done:
				continue
			}
		}
		call := &importResolveCall{done: make(chan struct{})}
		e.resolveFlights[key] = call
		e.mu.Unlock()

		title, count, gtype, chatID, err := resolve()
		e.mu.Lock()
		if err == nil && strings.TrimSpace(title) != "" {
			if e.resolveCache == nil {
				e.resolveCache = make(map[string]cachedImportResolution)
			}
			for cacheKey, value := range e.resolveCache {
				if !time.Now().Before(value.expiresAt) {
					delete(e.resolveCache, cacheKey)
				}
			}
			if len(e.resolveCache) >= importResolutionCacheLimit {
				for cacheKey := range e.resolveCache {
					delete(e.resolveCache, cacheKey)
					break
				}
			}
			e.resolveCache[key] = cachedImportResolution{
				title: title, count: count, gtype: gtype, chatID: chatID,
				expiresAt: time.Now().Add(importResolutionCacheTTL),
			}
		}
		delete(e.resolveFlights, key)
		close(call.done)
		e.mu.Unlock()
		return title, count, gtype, chatID, err
	}
}

const importDatabaseRetryAttempts = 3

func isTransientImportDatabaseError(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if len(pgErr.Code) >= 2 {
			switch pgErr.Code[:2] {
			case "08", "40", "53":
				return true
			}
		}
		switch pgErr.Code {
		case "57P01", "57P02", "57P03":
			return true
		}
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var urlErr *url.Error
	return errors.As(err, &urlErr)
}

func upsertImportGroupWithRetry(ctx context.Context, repo *repository.SystemGroupRepository, group *domain.SystemGroup) error {
	var err error
	for attempt := 0; attempt < importDatabaseRetryAttempts; attempt++ {
		err = repo.Upsert(ctx, group)
		if err == nil || !isTransientImportDatabaseError(err) || attempt == importDatabaseRetryAttempts-1 {
			return err
		}
		timer := time.NewTimer(time.Duration(attempt+1) * 250 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return err
}

func classifyImportFailure(err error) string {
	if err == nil {
		return "unknown"
	}
	var rateLimitErr *TelegramRateLimitError
	if errors.As(err, &rateLimitErr) {
		return "rate_limited"
	}
	var statusErr *TelegramHTTPStatusError
	if errors.As(err, &statusErr) {
		switch statusErr.StatusCode {
		case 404, 410:
			return "not_found"
		case 401, 403:
			return "forbidden"
		default:
			if statusErr.Retryable() {
				return "transient"
			}
		}
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "invite_hash_expired"), strings.Contains(message, "邀请已过期"), strings.Contains(message, "已过期"):
		return "expired"
	case strings.Contains(message, "invite_hash_invalid"), strings.Contains(message, "邀请无效"), strings.Contains(message, "格式无效"):
		return "invalid"
	case strings.Contains(message, "不存在"), strings.Contains(message, "not found"), strings.Contains(message, "username not occupied"):
		return "not_found"
	case isTransientResolveError(err):
		return "network"
	default:
		return "unknown"
	}
}

func isTransientResolveError(err error) bool {
	if err == nil {
		return false
	}
	var rateLimitErr *TelegramRateLimitError
	if errors.As(err, &rateLimitErr) {
		return false
	}
	var statusErr *TelegramHTTPStatusError
	if errors.As(err, &statusErr) {
		return statusErr.Retryable()
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return true
	}
	return errors.Is(err, io.ErrUnexpectedEOF)
}

// resolveWebEntryWithRetry keeps the import call site stable, while retry
// ownership lives in the shared scheduler. Keeping retries in both layers
// multiplied transient requests (up to six attempts per link).
func resolveWebEntryWithRetry(ctx context.Context, private bool, username, hash string) (string, int, string, error) {
	if private {
		return resolveInviteInfoViaWeb(ctx, hash)
	}
	return resolvePublicGroupInfoViaWeb(ctx, username)
}

// resolveInviteInfo 用 messages.checkChatInvite 解析私密邀请链接:
// 无需加入即可拿到群名、成员数、类型(ChatInvite 结构)。hash 为 t.me/+ 后的部分。
func resolveInviteInfo(ctx context.Context, client *telegram.Client, hash string) (title string, count int, gtype string, chatID int64, err error) {
	rctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	res, err := client.Invoke(rctx, &tg.MessagesCheckChatInviteRequest{Hash: hash})
	if err != nil {
		if rateLimitErr := telegramMTProtoRateLimit(err); rateLimitErr != nil {
			return "", 0, "", 0, rateLimitErr
		}
		s := strings.ToLower(err.Error())
		switch {
		case strings.Contains(s, "invite_hash_expired"):
			return "", 0, "", 0, fmt.Errorf("邀请已过期")
		case strings.Contains(s, "invite_hash_invalid"), strings.Contains(s, "invite_hash_empty"):
			return "", 0, "", 0, fmt.Errorf("邀请无效")
		default:
			// 保留 TG 原文, 供调用方识别 CHANNEL_PRIVATE 等场景走网页兜底
			return "", 0, "", 0, fmt.Errorf("校验邀请失败: %v", err)
		}
	}
	inv, ok := res.(*tg.ChatInvite)
	if !ok {
		switch v := res.(type) {
		case *tg.ChatInviteAlready:
			// 该账号已在群里, 直接返回群信息
			switch c := v.Chat.(type) {
			case *tg.Chat:
				return c.Title, int(c.ParticipantsCount), "group", int64(c.ID), nil
			case *tg.Channel:
				gt := "channel"
				if c.Megagroup {
					gt = "supergroup"
				}
				return c.Title, int(c.ParticipantsCount), gt, int64(c.ID), nil
			}
		case *tg.ChatInvitePeek:
			// TG 官方"不入群预览"变体: 内嵌完整 chat 对象
			switch c := v.Chat.(type) {
			case *tg.Chat:
				return c.Title, int(c.ParticipantsCount), "group", int64(c.ID), nil
			case *tg.Channel:
				gt := "channel"
				if c.Megagroup {
					gt = "supergroup"
				}
				return c.Title, int(c.ParticipantsCount), gt, int64(c.ID), nil
			}
		}
		return "", 0, "", 0, fmt.Errorf("邀请响应异常")
	}
	gt := "group"
	if inv.Broadcast {
		gt = "channel"
	}
	return inv.Title, int(inv.ParticipantsCount), gt, 0, nil
}

// resolveGroupInfo 解析公开群/频道的真实群名、成员数、类型。
func resolveGroupInfo(ctx context.Context, client *telegram.Client, username string) (title string, count int, gtype string, chatID int64, err error) {
	rctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	peer, err := client.ResolvePeer(rctx, "t.me/"+username)
	if err != nil {
		if rateLimitErr := telegramMTProtoRateLimit(err); rateLimitErr != nil {
			return "", 0, "", 0, rateLimitErr
		}
		return "", 0, "", 0, fmt.Errorf("解析失败(可能不存在或已注销)")
	}
	switch p := peer.(type) {
	case *tg.InputPeerChannel:
		fullRes, err := client.Invoke(rctx, &tg.ChannelsGetFullChannelRequest{
			Channel: &tg.InputChannel{ChannelID: p.ChannelID, AccessHash: p.AccessHash},
		})
		if err != nil {
			if rateLimitErr := telegramMTProtoRateLimit(err); rateLimitErr != nil {
				return "", 0, "", 0, rateLimitErr
			}
			return "", 0, "", 0, fmt.Errorf("读取群信息失败")
		}
		full, ok := fullRes.(*tg.MessagesChatFull)
		if !ok {
			return "", 0, "", 0, fmt.Errorf("群信息响应异常")
		}
		for _, c := range full.Chats {
			if ch, ok := c.(*tg.Channel); ok && ch.ID == p.ChannelID {
				title = ch.Title
				chatID = int64(ch.ID)
				if ch.Megagroup {
					gtype = "supergroup"
				} else {
					gtype = "channel"
				}
				break
			}
		}
		if cf, ok := full.FullChat.(*tg.ChannelFull); ok {
			count = int(cf.ParticipantsCount)
		}
	case *tg.InputPeerChat:
		fullRes, err := client.Invoke(rctx, &tg.MessagesGetFullChatRequest{ChatID: p.ChatID})
		if err != nil {
			return "", 0, "", 0, fmt.Errorf("读取群信息失败")
		}
		if full, ok := fullRes.(*tg.MessagesChatFull); ok {
			// mtgo 的 tg.ChatFull 没有 ParticipantsCount, 普通群成员数取自 tg.Chat 对象
			for _, c := range full.Chats {
				if ch, ok := c.(*tg.Chat); ok && ch.ID == p.ChatID {
					title, gtype, count, chatID = ch.Title, "group", int(ch.ParticipantsCount), int64(ch.ID)
					break
				}
			}
		}
	default:
		return "", 0, "", 0, fmt.Errorf("目标不是群组/频道")
	}
	if strings.TrimSpace(title) == "" {
		return "", 0, "", 0, fmt.Errorf("Telegram 未返回有效群组名称")
	}
	return title, count, gtype, chatID, nil
}
