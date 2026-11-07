package monitor

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"log"
	"regexp"
	"runtime/debug"

	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"

	"tg-sender/internal/config"
	"tg-sender/internal/domain"
	"tg-sender/internal/safego"
	"tg-sender/internal/sse"
)

// AccountEngine interface for monitor operations.
type AccountEngine interface {
	GetAllClients(tenantID uuid.UUID) []*telegram.Client
	GetClientByAccount(ctx context.Context, tenantID, accountID uuid.UUID) (*telegram.Client, interface{}, error)
	ReleaseClient(tenantID, accountID uuid.UUID, client *telegram.Client)
}

// SetVerifyNotifier R119: 注入验证结果通知回调
func (e *Engine) SetVerifyNotifier(fn func(tenantID, accountID uuid.UUID, groupTitle, status, reason string)) {
	e.verifyNotify = fn
}

// SetLoginCodeCallback R153: 注入登录验证码捕获回调
func (e *Engine) SetLoginCodeCallback(fn func(tenantID, accountID uuid.UUID, code string)) {
	e.loginCodeNotify = fn
}

// SetFlowDeadCallback R154: 注入假在线告警回调
func (e *Engine) SetFlowDeadCallback(fn func(tenantID, accountID uuid.UUID)) {
	e.flowDeadNotify = fn
}

// SetAccountExistsProvider R159: 注入账号存在性查询(删除账号后监控会话自动停止, 防僵尸会话 401 重试风暴)
func (e *Engine) SetAccountExistsProvider(fn func(tenantID, accountID uuid.UUID) bool) {
	e.accountExists = fn
}

// RuleRepository interface for reloading monitor rules.
type RuleRepository interface {
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.MonitorRule, error)
}

// Engine manages message monitoring and auto-reply for a tenant's accounts.
type Engine struct {
	cfg       *config.Config
	accEngine AccountEngine
	ruleRepo  RuleRepository
	sseBroker *sse.Broker
	mu        sync.RWMutex
	// lifecycleMu serializes StartMonitoring/StopMonitoring so concurrent
	// requests cannot register multiple handlers for the same account.
	lifecycleMu     sync.Mutex
	rateLimiter     *RateLimiter
	msgPushMu       sync.Mutex
	lastMsgPush     map[string]time.Time
	messageDedupMu  sync.Mutex
	seenMessages    map[messageDedupKey]time.Time
	messageDedupOps uint64
	done            chan struct{}
	// Rule cache: tenantID -> []MonitorRule + lastUpdate
	ruleCache   map[uuid.UUID][]domain.MonitorRule
	ruleCacheMu sync.RWMutex
	ruleCacheTS map[uuid.UUID]time.Time
	// tenantID -> accountID -> monitor session (cancel + done channel)
	sessions map[uuid.UUID]map[uuid.UUID]*monitorSession
	// reloadChs: tenantID:accountID -> channel for triggering rule reload
	reloadChs map[string]chan struct{}
	reloadMu  sync.Mutex

	// 入群验证(verifier.go): 依赖由 SetVerifierDeps 注入
	vlRepo        VerifierLogStore
	aiKeyProvider func(tenantID uuid.UUID) (provider, apiKey, model string, enabled bool)
	vstates       *verifierState
	// R119: 验证结果通知回调(pass→verify_solved 邮件, container 注入)
	verifyNotify    func(tenantID, accountID uuid.UUID, groupTitle, status, reason string)
	loginCodeNotify func(tenantID, accountID uuid.UUID, code string) // R153: 登录验证码捕获
	flowDeadNotify  func(tenantID, accountID uuid.UUID)              // R154: 假在线(流停滞)告警
	accountExists   func(tenantID, accountID uuid.UUID) bool         // R159: 账号删除后监控会话自清理

	// 抽奖助手(raffle.go): 开关由 SetRaffleProvider 注入
	raffleEnabled func(tenantID uuid.UUID) bool
	// 账号级抽奖开关(改进#1): 单账号可关闭, 默认 true 全部参与
	accountRaffleEnabled func(tenantID, accountID uuid.UUID) bool
	// 中奖回复文案(060): 返回 (是否开启, 文案), 中奖账号自动在抽奖群发感谢语
	raffleReply func(tenantID uuid.UUID) (bool, string)
	// 私聊机器人激活(061): 参与后私聊机器人发激活语, 建立私聊通道(机器人中奖私聊通知=最可靠中奖信号)
	raffleBotPing func(tenantID uuid.UUID) (bool, string)
	// 抽奖结果追踪(开奖解析/参与回执/持久化防重)
	raffleWatch *raffleWatcher

	// 语音振铃冷却: key=tenantID:accountID:calleeID → 上次拨打时间。
	// 通话是强打扰动作, 同一监控号对同一目标 10 分钟内最多拨一次, 防风控封号和恶意骚扰。
	callMu   sync.Mutex
	lastCall map[string]time.Time

	// DM 子动作状态按租户/账号/规则/目标/动作隔离，避免部分成功被整体去重吞掉。
	dmActionMu sync.Mutex
	dmActions  map[dmActionKey]dmActionStatus
}

// monitorSession tracks one account's monitoring goroutine.
type monitorSession struct {
	// R154: 上次收到消息时间(Unix秒), 假在线检测用 —— 探活只证明连接活着, 不证明消息流活着
	lastMsgAt atomic.Int64
	cancel    context.CancelFunc
	done      chan struct{} // closed when the goroutine fully exits
}

// NewEngine creates a new monitor engine.
func NewEngine(cfg *config.Config, accEngine AccountEngine, ruleRepo RuleRepository, sseBroker *sse.Broker) *Engine {
	e := &Engine{
		cfg:          cfg,
		accEngine:    accEngine,
		ruleRepo:     ruleRepo,
		sseBroker:    sseBroker,
		lastMsgPush:  make(map[string]time.Time),
		seenMessages: make(map[messageDedupKey]time.Time),
		rateLimiter:  NewRateLimiter(),
		ruleCache:    make(map[uuid.UUID][]domain.MonitorRule),
		ruleCacheTS:  make(map[uuid.UUID]time.Time),
		sessions:     make(map[uuid.UUID]map[uuid.UUID]*monitorSession),
		raffleWatch:  newRaffleWatcher(),
		done:         make(chan struct{}),
		reloadChs:    make(map[string]chan struct{}),
		lastCall:     make(map[string]time.Time),
		dmActions:    make(map[dmActionKey]dmActionStatus),
	}
	// P1-35 审计修复(2026-10-22): 原实现每秒 ticker 调 rateLimiter.Reset()——把
	// "每分钟 100 条"的限速计数每秒清零, 实际宽松 60 倍。RateLimiter 内部自带每分钟
	// 自清零(resetInterval=Minute), 这里整段删除, 不再外部干预。
	return e
}

// IsMonitoring reports whether monitoring is currently running for an account.
func (e *Engine) IsMonitoring(tenantID, accountID uuid.UUID) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if tenantSessions, ok := e.sessions[tenantID]; ok {
		_, exists := tenantSessions[accountID]
		return exists
	}
	return false
}

func (e *Engine) StartMonitoring(ctx context.Context, tenantID, accountID uuid.UUID, rules []domain.MonitorRule) error {
	e.lifecycleMu.Lock()
	defer e.lifecycleMu.Unlock()
	// 先停掉同账号的旧会话, 并等待其 goroutine 完全退出——否则旧 goroutine
	// 的清理逻辑可能在新 goroutine 注册 OnMessage 之后执行, 造成双注册/竞态。
	e.mu.Lock()
	var oldDone chan struct{}
	if tenantSessions, ok := e.sessions[tenantID]; ok {
		if sess, exists := tenantSessions[accountID]; exists {
			sess.cancel()
			oldDone = sess.done
			delete(tenantSessions, accountID)
		}
	}
	e.mu.Unlock()
	if oldDone != nil {
		select {
		case <-oldDone:
		case <-time.After(35 * time.Second):
			// 旧回调仍可能挂在 Telegram client 上。未确认退出前不能启动
			// 新会话，否则同一账号会重复注册消息处理器。
			log.Printf("[monitor] 等待旧监听退出超时，取消本次启动 tenant=%s account=%s", tenantID, accountID)
			clientKey := tenantID.String() + ":" + accountID.String()
			e.reloadMu.Lock()
			delete(e.reloadChs, clientKey)
			e.reloadMu.Unlock()
			return fmt.Errorf("previous monitor session did not stop within 35 seconds")
		}
	}

	client, _, err := e.accEngine.GetClientByAccount(ctx, tenantID, accountID)
	if err != nil {
		return fmt.Errorf("get client for monitor: %w", err)
	}
	// 不能持有客户端锁到会话结束——那会阻塞同账号的群发/采集任务。
	// 会话期间由 monitorLoop 里的保活逻辑周期性刷新 lastUsed 防回收。
	e.accEngine.ReleaseClient(tenantID, accountID, client)

	// 必须用 context.Background() 派生: 若用 HTTP 请求的 ctx, 请求结束后
	// 监听会立刻被取消(此处为 /monitor/start 接口路径的隐形 bug)。
	monitorCtx, cancel := context.WithCancel(context.Background())

	// Create reload channel for this account
	clientKey := tenantID.String() + ":" + accountID.String()
	e.reloadMu.Lock()
	// 旧会话通过 ctx.Done() 退出；不要关闭它正在监听的 channel，
	// 因为关闭后的 channel 会持续可读，可能让未退出的旧 loop 空转。
	e.reloadChs[clientKey] = make(chan struct{}, 1)
	e.reloadMu.Unlock()

	done := make(chan struct{})
	e.mu.Lock()
	if e.sessions[tenantID] == nil {
		e.sessions[tenantID] = make(map[uuid.UUID]*monitorSession)
	}
	e.sessions[tenantID][accountID] = &monitorSession{cancel: cancel, done: done}
	e.sessions[tenantID][accountID].lastMsgAt.Store(time.Now().Unix())
	e.mu.Unlock()

	safego.Go(func() { e.monitorLoop(monitorCtx, client, rules, tenantID, accountID, done) })
	return nil
}

// StopMonitoring stops monitoring for a specific account.
func (e *Engine) StopMonitoring(tenantID, accountID uuid.UUID) {
	e.lifecycleMu.Lock()
	defer e.lifecycleMu.Unlock()
	e.mu.Lock()
	var done chan struct{}
	if tenantSessions, ok := e.sessions[tenantID]; ok {
		if sess, exists := tenantSessions[accountID]; exists {
			sess.cancel()
			done = sess.done
			delete(tenantSessions, accountID)
		}
	}
	e.mu.Unlock()

	// 等待 goroutine 完全退出, 保证其 OnMessage 回调先于新会话停止处理,
	// 避免同一条消息被新旧两个 handler 重复处理。
	if done != nil {
		select {
		case <-done:
		case <-time.After(35 * time.Second):
			log.Printf("[monitor] 等待监听退出超时 tenant=%s account=%s", tenantID, accountID)
		}
	}

	// Cleanup reload channel
	clientKey := tenantID.String() + ":" + accountID.String()
	e.reloadMu.Lock()
	// monitorLoop 通过 ctx.Done() 退出。不要 close 它持有的 reload channel：
	// channel 关闭后会一直可读，若 loop 尚未完全退出可能导致忙循环。
	delete(e.reloadChs, clientKey)
	e.reloadMu.Unlock()
}

// EnsureSession 会话已在线则跳过, 否则以给定规则启动(抽奖助手自动拉起用)。
// 抽奖抢答(tapRaffle)挂在 OnMessage 上, 必须保证会话存在才会触发。
func (e *Engine) EnsureSession(ctx context.Context, tenantID, accountID uuid.UUID, rules []domain.MonitorRule) error {
	e.mu.RLock()
	alive := false
	if ts, ok := e.sessions[tenantID]; ok {
		if sess, exists := ts[accountID]; exists && sess != nil {
			alive = true
		}
	}
	e.mu.RUnlock()
	if alive {
		return nil
	}
	return e.StartMonitoring(ctx, tenantID, accountID, rules)
}

// StopAll stops all monitoring sessions.
func (e *Engine) StopAll() {
	e.lifecycleMu.Lock()
	defer e.lifecycleMu.Unlock()
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, tenantSessions := range e.sessions {
		for _, sess := range tenantSessions {
			sess.cancel()
		}
	}
	select {
	case <-e.done:
		// already closed
	default:
		close(e.done)
	}
	e.sessions = make(map[uuid.UUID]map[uuid.UUID]*monitorSession)
}

// ReloadRules triggers rule reload for all active monitoring sessions of a tenant.
func (e *Engine) ReloadRules(ctx context.Context, tenantID uuid.UUID) error {
	// 在锁内复制账号 ID；不能把 sessions 子 map 的引用带出锁后遍历，
	// StartMonitoring/StopMonitoring 会并发写入该 map。
	e.mu.RLock()
	tenantSessions := e.sessions[tenantID]
	accountIDs := make([]uuid.UUID, 0, len(tenantSessions))
	for accountID := range tenantSessions {
		accountIDs = append(accountIDs, accountID)
	}
	e.mu.RUnlock()

	for _, accountID := range accountIDs {
		clientKey := tenantID.String() + ":" + accountID.String()
		e.reloadMu.Lock()
		if ch, exists := e.reloadChs[clientKey]; exists {
			select {
			case ch <- struct{}{}:
			default:
			}
		}
		e.reloadMu.Unlock()
	}
	return nil
}

func (e *Engine) monitorLoop(ctx context.Context, client *telegram.Client, initialRules []domain.MonitorRule, tenantID, accountID uuid.UUID, done chan struct{}) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[monitor] loop panic: %v\n%s", r, debug.Stack())
		}
	}()
	// 通知 StartMonitoring/StopMonitoring 本 goroutine 已完全退出。
	defer func() {
		e.mu.Lock()
		if tenantSessions := e.sessions[tenantID]; tenantSessions != nil {
			if current := tenantSessions[accountID]; current != nil && current.done == done {
				delete(tenantSessions, accountID)
				if len(tenantSessions) == 0 {
					delete(e.sessions, tenantID)
				}
			}
		}
		e.mu.Unlock()
		clientKey := tenantID.String() + ":" + accountID.String()
		e.reloadMu.Lock()
		delete(e.reloadChs, clientKey)
		e.reloadMu.Unlock()
		close(done)
	}()

	clientKey := tenantID.String() + ":" + accountID.String()

	// Use a pointer to allow rule updates without re-registering OnMessage
	rules := initialRules
	var rulesMu sync.RWMutex

	// Pre-compile regex patterns for performance (avoid recompilation on every message)
	compiledPatterns := make(map[string]*regexp.Regexp)
	var patternsMu sync.RWMutex
	compilePatterns := func(ruleList []domain.MonitorRule) {
		newPatterns := make(map[string]*regexp.Regexp)
		for _, r := range ruleList {
			if r.MatchType == "regex" {
				for _, p := range r.Keywords {
					if _, ok := newPatterns[p]; !ok {
						if re, err := regexp.Compile(p); err == nil {
							newPatterns[p] = re
						}
					}
				}
			}
		}
		patternsMu.Lock()
		compiledPatterns = newPatterns
		patternsMu.Unlock()
	}
	compilePatterns(rules)

	log.Printf("[monitor] 监听已启动 tenant=%s account=%s 规则数=%d", tenantID, accountID, len(initialRules))

	// Get reload channel
	e.reloadMu.Lock()
	reloadCh := e.reloadChs[clientKey]
	e.reloadMu.Unlock()

	// currentClient 跟踪 OnMessage 当前挂载的底层连接实例;
	// 连接被池回收重建后需要重挂(见下方 keepAlive)。
	currentClient := client

	// registerOn 把消息处理逻辑挂到指定客户端上, 可被重复调用于换线重挂。
	registerOn := func(cl *telegram.Client) {
		cl.OnMessage(func(c *telegram.Client, msg *types.Message) {
			select {
			case <-ctx.Done():
				return
			default:
			}
			// Rate limit: max 50 msg/sec per account, burst 100
			if !e.rateLimiter.Allow(clientKey) {
				return
			}
			if msg == nil {
				return
			}
			if msg.ChatID == 0 || msg.ID == 0 {
				return
			}
			// Deduplicate the same Telegram update across overlapping handlers/reconnects.
			if !e.markMessageSeen(tenantID, accountID, msg.ChatID, msg.ID) {
				return
			}
			// 入群验证: 命中验证消息时自动处理(详见 verifier.go / verifier_tap.go)
			// R153: 登录验证码捕获(TG官方777000/私聊), 异步处理不阻塞消息循环
			if code := extractLoginCode(msg.Text, msg.FromID, msg.ChatID); code != "" && e.loginCodeNotify != nil {
				safego.Go(func() { e.loginCodeNotify(tenantID, accountID, code) })
			}
			e.tapVerification(ctx, cl, tenantID, accountID, msg.ChatID, "", fmt.Sprint(msg.FromID), msg.Text, rawTG(msg))
			// 抽奖助手: 命中抽奖公告自动参与(raffle.go)。
			// 消息包装对象由 Telegram 客户端管理，不跨 goroutine 捕获原指针；
			// 复制包装结构后再异步处理，避免回调返回后对象被复用导致字段错乱。
			msgSnapshot := *msg
			safego.Go(func() { e.tapRaffle(ctx, cl, tenantID, accountID, &msgSnapshot) })
			// 抽奖结果追踪: 开奖解析(text_mention实体) + 参与回执确认(2026-10-20)
			// + 中奖自动回复(060, 需 cl 发送)
			e.watchRaffleMessage(ctx, tenantID, accountID, msg.ChatID, msg, cl)
			textPreview := msg.Text
			if len(textPreview) > 30 {
				textPreview = textPreview[:30] + "..."
			}
			e.mu.RLock()
			if _sess := e.sessions[tenantID][accountID]; _sess != nil {
				_sess.lastMsgAt.Store(time.Now().Unix())
			}
			e.mu.RUnlock()
			log.Printf("[monitor] 收到消息 chat=%d from=%d text=%q", msg.ChatID, msg.FromID, textPreview)

			// 节流: 同群 1 秒内最多推 1 条, 避免高频消息刷爆前端
			now := time.Now()
			pushKey := fmt.Sprintf("%s:%s:%d", tenantID, accountID, msg.ChatID)
			e.msgPushMu.Lock()
			shouldPush := now.Sub(e.lastMsgPush[pushKey]) >= time.Second
			if shouldPush {
				e.lastMsgPush[pushKey] = now
			}
			// 防内存泄漏：map 只增不减，监控账号漫游大量群组后无界增长。
			// 超过阈值时重建并丢弃过期条目（保留最近 5 分钟），摊销清理成本。
			if len(e.lastMsgPush) > 10000 {
				fresh := make(map[string]time.Time, len(e.lastMsgPush))
				for k, v := range e.lastMsgPush {
					if now.Sub(v) < 5*time.Minute {
						fresh[k] = v
					}
				}
				e.lastMsgPush = fresh
			}
			e.msgPushMu.Unlock()
			// 仅节流前端预览推送；绝不能 return，否则会跳过后续规则匹配和自动动作。
			if shouldPush {
				e.sseBroker.BroadcastToTenant(tenantID, "new_message", map[string]any{
					"account_id": accountID.String(),
					"chat_id":    msg.ChatID,
					"message_id": msg.ID,
					"text":       msg.Text,
					"from_id":    msg.FromID,
					"date":       msg.Date,
				})
			}

			// 匹配+动作整体异步：规则多时匹配循环在回调里同步跑会阻塞消息消费，
			// 与动作执行同款问题。先拷贝标量字段（msg 指针由 mtgo 库管理），整体移入 goroutine。
			msgChatID2, msgFromID2 := msg.ChatID, msg.FromID
			msgID2 := msg.ID
			msgText2 := msg.Text
			msgHasMedia2 := msg.Media != nil
			safego.Go(func() {
				// Read rules with lock
				rulesMu.RLock()
				currentRules := rules
				rulesMu.RUnlock()

				activeCount := 0
				for _, r := range currentRules {
					if r.IsActive {
						activeCount++
					}
				}
				log.Printf("[monitor] 消息进入规则匹配: chat=%d 活跃规则=%d", msgChatID2, activeCount)

				// Process rules: forward or reply
				for _, rule := range currentRules {
					if !rule.IsActive {
						continue
					}

					// 2026-09-22 规则账号隔离：rule.AccountID 非 nil 时仅属于该账号，
					// nil 表示"全部账号"（老数据兼容）。否则 A 号规则会触发 B 号（用户实测）。
					if rule.AccountID != nil && *rule.AccountID != accountID {
						continue
					}

					// 仅处理规则指定来源群的消息(未指定则监听所有群)
					if len(rule.SourceChatIDs) > 0 {
						inSource := false
						chatIDStr := strconv.FormatInt(msgChatID2, 10)
						for _, src := range rule.SourceChatIDs {
							if src == chatIDStr {
								inSource = true
								break
							}
						}
						if !inSource {
							continue
						}
					}

					// Determine if message matches rule
					matched := false
					switch rule.MatchType {
					case "contains":
						for _, keyword := range rule.Keywords {
							if strings.Contains(msgText2, keyword) {
								matched = true
								break
							}
						}
					case "exact":
						for _, keyword := range rule.Keywords {
							if msgText2 == keyword {
								matched = true
								break
							}
						}
					case "regex":
						patternsMu.RLock()
						for _, pattern := range rule.Keywords {
							if re, ok := compiledPatterns[pattern]; ok && re.MatchString(msgText2) {
								matched = true
								break
							}
						}
						patternsMu.RUnlock()
					case "from_user":
						for _, username := range rule.Keywords {
							// 仅支持数值 ID 匹配：msg.FromID 是 int64，与 @字符串用户名
							// 比较永不成立（原 "Also check without @ prefix" 分支是死代码，
							// 拿数值 ID 和用户名文本比对）。如需 @username 匹配，
							// 需从消息实体解析 sender username，另起功能实现。
							idStr := strings.TrimPrefix(username, "@")
							if msgFromID2 != 0 && fmt.Sprintf("%d", msgFromID2) == idStr {
								matched = true
								break
							}
						}
					default:
						log.Printf("[monitor] 规则 %s 不支持匹配类型 %s", rule.RuleID, rule.MatchType)
					}
					if !matched {
						continue
					}

					log.Printf("[monitor] 规则命中: rule=%s type=%s chat=%d text=%q", rule.Name, rule.Type, msgChatID2, func() string {
						t := msgText2
						if len(t) > 30 {
							t = t[:30] + "..."
						}
						return t
					}())

					// 已在外层 goroutine：直接顺序执行，不再嵌套 goroutine
					e.executeRuleAction(ctx, c, tenantID, accountID, rule, msgChatID2, msgFromID2, int(msgID2), msgText2, msgHasMedia2)
				}
			})
		})
	}

	// Start goroutine to reload rules periodically or on demand
	safego.Go(func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[monitor] 规则重载异常: %v", r)
			}
		}()
		ticker := time.NewTicker(e.cfg.MonitorRuleReloadInterval)
		defer ticker.Stop()
		// 保活: OnMessage 回调不会刷新客户端 lastUsed, 若不加保活,
		// 池清理器(默认5分钟)会把监听依附的连接当空闲连接回收。
		keepAlive := func() bool {
			// R159: 账号已被删除 → 停止本监控会话(否则僵尸会话会继续向池要连接,
			// 对不存在的会话反复重连, 刷 401 AUTH_KEY_UNREGISTERED)
			if e.accountExists != nil && !e.accountExists(tenantID, accountID) {
				log.Printf("[monitor] 账号 %s 已删除, 停止监控会话", accountID)
				// Let this loop exit; its deferred cleanup removes only its own
				// session (never a newer replacement session).
				return false
			}
			cl, _, err := e.accEngine.GetClientByAccount(ctx, tenantID, accountID)
			if err != nil {
				return true
			}
			e.accEngine.ReleaseClient(tenantID, accountID, cl)
			// 底层连接被回收后重建了新实例: 把 OnMessage 重挂到新连接上。
			if cl != currentClient {
				log.Printf("[monitor] 底层连接已更换, 重新注册监听 tenant=%s account=%s", tenantID, accountID)
				currentClient = cl
				registerOn(cl)
			}
			// R154: 假在线检测 —— 连接探活通过 ≠ 消息流活着(半死连接/PTS缺口/他端挤占都会让流静默死亡)。
			// 超过 30 分钟无任何消息流过 → 告警 + 强制重挂回调 + 通知上层处理。
			e.mu.RLock()
			_sess := e.sessions[tenantID][accountID]
			e.mu.RUnlock()
			if _sess != nil {
				if last := _sess.lastMsgAt.Load(); last > 0 && time.Since(time.Unix(last, 0)) > 30*time.Minute {
					// R156: 只告警不自动重挂(重复 registerOn 可能造成会话异常掉线), 连接死活交给底层池探活
					log.Printf("[monitor] 账号 %s 疑似假在线(30分钟无消息流过), 已告警", accountID)
					_sess.lastMsgAt.Store(time.Now().Unix()) // 防 tick 风暴
					if e.flowDeadNotify != nil {
						safego.Go(func() { e.flowDeadNotify(tenantID, accountID) })
					}
				}
			}
			return true
		}
		for {
			select {
			case <-ticker.C:
				if !keepAlive() {
					return
				}
				if e.ruleRepo != nil {
					newRules, err := e.ruleRepo.ListByTenant(ctx, tenantID)
					if err == nil {
						rulesMu.Lock()
						rules = make([]domain.MonitorRule, len(newRules))
						for i, r := range newRules {
							rules[i] = *r
						}
						rulesMu.Unlock()
						compilePatterns(rules)
					}
				}
			case <-reloadCh:
				if e.ruleRepo != nil {
					newRules, err := e.ruleRepo.ListByTenant(ctx, tenantID)
					if err == nil {
						rulesMu.Lock()
						rules = make([]domain.MonitorRule, len(newRules))
						for i, r := range newRules {
							rules[i] = *r
						}
						rulesMu.Unlock()
						compilePatterns(rules)
					}
				}
			case <-ctx.Done():
				return
			}
		}
	})

	// 初始挂载
	registerOn(currentClient)

	<-ctx.Done()
}

// executeRuleAction 执行命中的监控规则（forward/reply/dm），由 OnMessage 异步触发。
// 抽成方法而非包级函数：语音振铃需要访问引擎级拨打冷却表(lastCall)。
// 动作是联网调用，若在消息回调里同步执行，高频群消息会互相阻塞、拖垮整个监听会话。
// 参数只传标量副本（msg 指针生命周期由 mtgo 库管理，不可跨 goroutine 捕获）。
func (e *Engine) executeRuleAction(ctx context.Context, c *telegram.Client, tenantID, accountID uuid.UUID, rule domain.MonitorRule, msgChatID, msgFromID int64, msgID int, msgText string, msgHasMedia bool) {
	select {
	case <-ctx.Done():
		return
	default:
	}
	switch rule.Type {
	case "forward":
		// Forward message to target chats
		for _, targetChatID := range rule.TargetChatIDs {
			if ctx.Err() != nil {
				return
			}
			fwdCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			var tgtID int64
			if parsed, err := strconv.ParseInt(targetChatID, 10, 64); err == nil {
				tgtID = parsed
			} else {
				peer, err := c.ResolvePeer(fwdCtx, targetChatID)
				if err != nil {
					log.Printf("[monitor] 转发目标 %s 解析失败: %v", targetChatID, err)
					cancel()
					continue
				}
				// Extract ID from peer (simplified)
				switch p := peer.(type) {
				case *tg.InputPeerChat:
					tgtID = p.ChatID
				case *tg.InputPeerChannel:
					tgtID = p.ChannelID
				default:
					log.Printf("[monitor] 转发不支持的目标类型: %T", peer)
					cancel()
					continue
				}
			}
			// Skip media-only messages (no text) to avoid sending empty messages
			if msgText == "" && msgHasMedia {
				log.Printf("[monitor] 仅媒体消息跳过转发到 %s", targetChatID)
				cancel()
				continue
			}
			_, err := c.SendMessage(fwdCtx, tgtID, msgText, &params.SendMessage{})
			cancel()
			if err != nil {
				log.Printf("[monitor] 转发到 %s 失败: %v", targetChatID, err)
			}
		}
	case "reply":
		// Use user-configured reply content instead of hard-coded text
		replyText := rule.ReplyContent
		if replyText == "" {
			// Fallback to a generic message if reply content is empty
			replyText = "收到您的消息"
		}
		if msgID < 0 || msgID > 2147483647 {
			log.Printf("[monitor] 自动回复跳过：消息 ID 超出 Telegram int32 范围: %d", msgID)
			return
		}
		replyCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		if _, sendErr := c.SendMessage(replyCtx, msgChatID, replyText, &params.SendMessage{
			ReplyToMessageID: int32(msgID),
		}); sendErr != nil {
			log.Printf("[monitor] 自动回复发送失败: %v", sendErr)
		}
		cancel()
	case "dm":
		// 私聊触达: 监控号对触发人执行规则勾选的子动作(私聊文本/联系人名片/语音振铃), 可任意组合
		executeDM(ctx, e, c, tenantID, accountID, rule, msgFromID)
	default:
		log.Printf("[monitor] 规则 %s 不支持的规则类型 %s", rule.Type, rule.RuleID)
	}
}

// anonymousSenderID 是 Telegram 匿名群管理员发消息时使用的虚拟用户 ID, 无法私聊/拨打。
const anonymousSenderID = 1087968824

// executeDM 执行 type=dm 规则的三个子动作。
func executeDM(ctx context.Context, e *Engine, c *telegram.Client, tenantID, accountID uuid.UUID, rule domain.MonitorRule, targetUser int64) {
	if ctx.Err() != nil {
		return
	}
	if targetUser <= 0 || targetUser == anonymousSenderID {
		log.Printf("[monitor] dm 动作跳过: 触发人不可触达(from=%d)", targetUser)
		return
	}
	// 每个子动作独立领取和记录结果，避免一个动作失败阻断其他动作。
	// ① 私聊文本
	if ctx.Err() != nil {
		return
	}
	if rule.DMContent != "" {
		key := e.dmActionKey(tenantID, accountID, rule.RuleID, targetUser, "text")
		if e.claimDMAction(key) {
			dmCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			_, err := c.SendMessage(dmCtx, targetUser, rule.DMContent, &params.SendMessage{})
			cancel()
			if err != nil {
				e.finishDMAction(key, dmActionOutcomeUnknown)
				log.Printf("[monitor] 私聊文本发送失败(结果可能不确定，不自动重试) target=%d: %v", targetUser, err)
			} else {
				e.finishDMAction(key, dmActionSucceeded)
			}
		}
	}
	// ② 联系人名片(即客户端回形针→联系人生成的绿色卡片: inputMediaContact)
	if ctx.Err() != nil {
		return
	}
	if rule.DMContact {
		key := e.dmActionKey(tenantID, accountID, rule.RuleID, targetUser, "contact")
		if e.claimDMAction(key) {
			dmCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			cleanPhone := strings.Map(func(r rune) rune {
				if r >= '0' && r <= '9' {
					return r
				}
				return -1
			}, rule.DMContactPhone)

			firstName := rule.DMContactFirst
			if firstName == "" {
				firstName = "User"
			}
			lastName := rule.DMContactLast
			peer, perr := c.ResolvePeer(dmCtx, strconv.FormatInt(targetUser, 10))
			if perr != nil {
				e.finishDMAction(key, dmActionFailedDefinite)
				log.Printf("[monitor] 名片发送失败(解析目标) target=%d: %v", targetUser, perr)
			} else {
				imported, ierr := c.Raw().ContactsImportContacts(dmCtx, &tg.ContactsImportContactsRequest{
					Contacts: []*tg.InputPhoneContact{{
						ClientID: time.Now().UnixNano(), Phone: cleanPhone,
						FirstName: firstName, LastName: lastName,
					}},
				})
				if ierr != nil {
					log.Printf("[monitor] 导入联系人失败: %v", ierr)
				} else if len(imported.Users) > 0 {
					if u, ok := imported.Users[0].(*tg.User); ok {
						log.Printf("[monitor] 确认联系人(复用): %s (TG ID: %d)", firstName, u.ID)
					}
				}
				ridBytes := make([]byte, 8)
				if _, rerr := crand.Read(ridBytes); rerr != nil {
					e.finishDMAction(key, dmActionFailedDefinite)
					log.Printf("[monitor] 名片发送失败(生成随机 ID) target=%d: %v", targetUser, rerr)
				} else {
					_, merr := c.Raw().MessagesSendMedia(dmCtx, &tg.MessagesSendMediaRequest{
						Peer: peer,
						Media: &tg.InputMediaContact{
							PhoneNumber: cleanPhone, FirstName: firstName,
							LastName: lastName, Vcard: "",
						},
						RandomID: int64(binary.LittleEndian.Uint64(ridBytes) >> 1),
					})
					if merr != nil {
						e.finishDMAction(key, dmActionOutcomeUnknown)
						log.Printf("[monitor] 联系人名片发送失败(结果可能不确定，不自动重试) target=%d: %v", targetUser, merr)
					} else {
						e.finishDMAction(key, dmActionSucceeded)
						log.Printf("[monitor] ✅ 联系人名片发送成功 target=%d", targetUser)
					}
				}
			}
			cancel()
		}
	}

	// ③ 语音振铃(只振铃不接通, 走通话信令: requestCall→响 N 秒→discardCall)
	if ctx.Err() != nil {
		return
	}
	if rule.DMCall {
		key := e.dmActionKey(tenantID, accountID, rule.RuleID, targetUser, "call")
		if e.claimDMAction(key) {
			if !e.callAllowed(tenantID, accountID, targetUser) {
				e.finishDMAction(key, dmActionFailedDefinite)
				log.Printf("[monitor] 语音振铃冷却中, 跳过 target=%d", targetUser)
				return
			}
			secs := rule.DMCallSeconds
			if secs <= 0 {
				secs = 5
			}
			if err := ringUser(ctx, c, targetUser, secs); err != nil {
				e.finishDMAction(key, dmActionOutcomeUnknown)
				log.Printf("[monitor] 语音振铃失败(结果可能不确定，不自动重试) target=%d: %v", targetUser, err)
			} else {
				e.finishDMAction(key, dmActionSucceeded)
			}
		}
	}
}

// callAllowed 语音振铃冷却: 同一(租户,监控号,被叫) 10 分钟内只允许拨一次。
// 通话是强打扰动作, 不加冷却高频触发极易被风控封号或遭对方举报。
func (e *Engine) callAllowed(tenantID, accountID uuid.UUID, callee int64) bool {
	key := tenantID.String() + ":" + accountID.String() + ":" + strconv.FormatInt(callee, 10)
	e.callMu.Lock()
	defer e.callMu.Unlock()
	if last, ok := e.lastCall[key]; ok && time.Since(last) < 10*time.Minute {
		return false
	}
	e.lastCall[key] = time.Now()
	// 防内存泄漏: 超限重建, 仅保留冷却期内的条目(与 lastMsgPush 同款思路)
	if len(e.lastCall) > 10000 {
		fresh := make(map[string]time.Time, len(e.lastCall))
		cutoff := time.Now().Add(-10 * time.Minute)
		for k, v := range e.lastCall {
			if v.After(cutoff) {
				fresh[k] = v
			}
		}
		e.lastCall = fresh
	}
	return true
}

// ringUser 让目标用户手机振铃 ringSeconds 秒后挂断。
// 只走通话信令(phone.requestCall / phone.discardCall), 不实现 tgvoip 音频层:
// 对方手机会响铃, 挂断后显示未接来电, 正好满足"弹语音"提醒需求。
func ringUser(ctx context.Context, c *telegram.Client, userID int64, ringSeconds int) error {
	if ringSeconds < 0 || ringSeconds > 2147483647 {
		return fmt.Errorf("ring duration out of int32 range: %d", ringSeconds)
	}
	// 解析目标用户拿 access_hash(requestCall 的 InputUser 必需)
	rpCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	peer, err := c.ResolvePeer(rpCtx, strconv.FormatInt(userID, 10))
	cancel()
	if err != nil {
		return fmt.Errorf("解析目标用户: %w", err)
	}
	up, ok := peer.(*tg.InputPeerUser)
	if !ok {
		return fmt.Errorf("目标 peer 不是用户: %T", peer)
	}

	// g_a_hash 是 DH 公钥的 sha256。振铃即挂断、永不进入密钥协商, 随机字节即可;
	// random_id 为 int32 防重放。
	gaHash := make([]byte, 32)
	ridBytes := make([]byte, 4)
	if _, err := crand.Read(gaHash); err != nil {
		return fmt.Errorf("生成 g_a_hash: %w", err)
	}
	if _, err := crand.Read(ridBytes); err != nil {
		return fmt.Errorf("生成 random_id: %w", err)
	}

	// 发起呼叫 → 服务端推送来电给目标用户(手机响铃)
	reqCtx, reqCancel := context.WithTimeout(ctx, 20*time.Second)
	res, err := c.Raw().PhoneRequestCall(reqCtx, &tg.PhoneRequestCallRequest{
		UserID:   &tg.InputUser{UserID: up.UserID, AccessHash: up.AccessHash},
		RandomID: int32(binary.LittleEndian.Uint32(ridBytes) >> 1),
		GAHash:   gaHash,
		Protocol: &tg.PhoneCallProtocol{
			UDPReflector: true,
			MinLayer:     65,
			MaxLayer:     92,
		},
	})
	reqCancel()
	if err != nil {
		return fmt.Errorf("requestCall: %w", err)
	}

	// 从响应中取出通话句柄(phoneCallWaiting 里的 id/access_hash)
	var callID, callAccess int64
	switch pc := res.(type) {
	case *tg.PhonePhoneCall:
		if w, ok := pc.PhoneCall.(*tg.PhoneCallWaiting); ok {
			callID, callAccess = w.ID, w.AccessHash
		}
	case *tg.PhoneCallWaiting:
		callID, callAccess = pc.ID, pc.AccessHash
	}
	if callID == 0 {
		return fmt.Errorf("requestCall 响应缺少通话句柄: %T", res)
	}

	// 振铃 ringSeconds 秒; 监控会话被停止时立即挂断, 不留悬挂呼叫
	select {
	case <-time.After(time.Duration(ringSeconds) * time.Second):
	case <-ctx.Done():
	}

	// 挂断 → 对端收到未接来电。用 Background 派生, 避免监控停止导致电话挂不掉
	disCtx, disCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer disCancel()
	if _, err := c.Raw().PhoneDiscardCall(disCtx, &tg.PhoneDiscardCallRequest{
		Peer:         &tg.InputPhoneCall{ID: callID, AccessHash: callAccess},
		Duration:     int32(ringSeconds),
		Reason:       &tg.PhoneCallDiscardReasonHangup{},
		ConnectionID: 0,
	}); err != nil {
		return fmt.Errorf("discardCall: %w", err)
	}
	return nil
}
