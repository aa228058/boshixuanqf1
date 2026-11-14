package handler

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"tg-sender/internal/domain"
	"tg-sender/internal/domain/task"
	"tg-sender/internal/middleware"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// CreateBroadcastRequest represents broadcast task request.
type CreateBroadcastRequest struct {
	AccountIDs  []string `json:"account_ids" validate:"required"`
	TargetIDs   []string `json:"target_ids" validate:"required"`
	Message     string   `json:"message" validate:"required"`
	ScheduledAt string   `json:"scheduled_at,omitempty"`
	IntervalMin int      `json:"interval_min,omitempty"` // 最小发送间隔(秒)
	IntervalMax int      `json:"interval_max,omitempty"` // 最大发送间隔(秒)
	// Mappings enables precise account-target mapping mode.
	// If provided, each account sends only to its assigned targets (no cartesian product).
	// Format: [{"account_id": "uuid", "targets": ["@group1", "-100..."]}, ...]
	Mappings []struct {
		AccountID string   `json:"account_id"`
		Targets   []string `json:"targets"`
	} `json:"mappings,omitempty"`
	// 富媒体与 URL 按钮(F1), 注入任务 payload 供执行器使用
	Media   map[string]any   `json:"media,omitempty"`   // {"type":"photo|video","source":"url或路径"}
	Buttons []map[string]any `json:"buttons,omitempty"` // [{"text":"...","url":"..."}]
	// Priority 任务优先级 1-10(默认 5; 8=优先插队, ClaimTasks 按 priority DESC 排序)
	Priority  int  `json:"priority,omitempty"`
	AutoVary  bool `json:"auto_vary,omitempty"`
	Obfuscate bool `json:"obfuscate,omitempty"`
}

// broadcastExtras 提取请求中的富媒体/URL按钮等扩展字段, 注入任务 payload
func broadcastExtras(req *CreateBroadcastRequest) map[string]any {
	log.Printf("[broadcast] 创建请求 media=%v buttons=%d", req.Media, len(req.Buttons))
	extra := map[string]any{}
	if len(req.Media) > 0 {
		extra["media"] = req.Media
	}
	if len(req.Buttons) > 0 {
		extra["buttons"] = req.Buttons
	}
	if req.Priority >= 1 && req.Priority <= 10 {
		extra["priority"] = req.Priority
	}
	// Broadcast executor already supports these payload switches; preserve them through HTTP DTO parsing.
	if req.AutoVary {
		extra["auto_vary"] = true
	}
	if req.Obfuscate {
		extra["obfuscate"] = true
	}
	if len(extra) == 0 {
		return nil
	}
	return extra
}

// CreateRequest represents generic task creation request.
type CreateTaskRequest struct {
	Type        string   `json:"type" validate:"required"` // broadcast | scrape_members | join_groups
	AccountIDs  []string `json:"account_ids,omitempty"`
	AccountID   string   `json:"account_id,omitempty"`
	TargetIDs   []string `json:"target_ids,omitempty"`
	GroupID     string   `json:"group_id,omitempty"`
	Message     string   `json:"message,omitempty"`
	Links       []string `json:"links,omitempty"`
	ScheduledAt string   `json:"scheduled_at,omitempty"`
	// Priority 任务优先级 1-10(默认 5; 8=优先插队, ClaimTasks 按 priority DESC 排序)
	Priority int `json:"priority,omitempty"`
}

// Create godoc
// @Summary 创建任务
// @Description 根据类型创建不同类型的任务(broadcast/scrape_members/join_groups)
// @Tags 任务
// @Accept json
// @Produce json
// @Param body body CreateTaskRequest true "创建任务请求"
// @Success 201 {object} TaskResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks [post]
// Create dispatches to specific task creation handlers based on type.
func (h *TaskHandler) Create(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}

	// Read body once and restore so sub-handlers can re-bind
	if c.Request().ContentLength > 10*1024*1024 {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "请求体过大（最大10MB）")
	}
	bodyBytes, err := io.ReadAll(c.Request().Body)
	if err != nil {
		log.Printf("[handler] 读取任务请求体失败: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "请求参数错误")
	}
	c.Request().Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var req CreateTaskRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("[handler] 解析任务请求失败: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "请求参数错误")
	}

	// 前端部分页面(如加群)传单数 account_id，此处归一化为 account_ids，
	// 否则任务不带账号，执行时永远"加入群聊中"空转
	if len(req.AccountIDs) == 0 && req.AccountID != "" {
		req.AccountIDs = []string{req.AccountID}
	}

	log.Printf("[handler] 创建任务: type=%s account_id=%q account_ids=%v target_ids=%v links=%d",
		req.Type, req.AccountID, req.AccountIDs, req.TargetIDs, len(req.Links))

	// 审计: 记录任务创建(谁/何时/什么类型/规模)
	if h.auditRepo != nil {
		_ = h.auditRepo.Create(c.Request().Context(), &domain.AuditLog{
			LogID:     uuid.New(),
			TenantID:  tenantID,
			UserID:    tenantID.String(),
			Action:    "task_create",
			Target:    fmt.Sprintf("%s accounts=%d targets=%d", req.Type, len(req.AccountIDs), len(req.TargetIDs)),
			IP:        c.RealIP(),
			CreatedAt: time.Now(),
		})
	}

	// 根据任务类型检查套餐权限
	if h.planChecker != nil {
		switch req.Type {
		case "broadcast":
			if err := h.planChecker.CheckFeature(c, tenantID.String(), middleware.FeatureBroadcast); err != nil {
				return err
			}
		case "scrape_members":
			if err := h.planChecker.CheckFeature(c, tenantID.String(), middleware.FeatureScrapeMembers); err != nil {
				return err
			}
		case "scrape_links":
			if err := h.planChecker.CheckFeature(c, tenantID.String(), middleware.FeatureScrapeLinks); err != nil {
				return err
			}
		case "join_groups":
			if err := h.planChecker.CheckFeature(c, tenantID.String(), middleware.FeatureJoinGroups); err != nil {
				return err
			}
		case "scheduled_broadcast":
			if err := h.planChecker.CheckFeature(c, tenantID.String(), middleware.FeatureScheduledTasks); err != nil {
				return err
			}
		}
	}

	// Restore body for sub-handler
	c.Request().Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	switch req.Type {
	case "broadcast":
		return h.CreateBroadcast(c)
	case "scrape_members":
		if len(req.TargetIDs) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "成员采集需要指定目标群组")
		}
		return h.CreateScrapeMembers(c)
	case "scrape_links":
		return h.CreateScrapeLinks(c)
	case "join_groups":
		return h.CreateJoinGroups(c)
	case "monitor_scrape":
		return h.CreateMonitorScrape(c)
	case "scheduled_broadcast":
		return h.CreateScheduledBroadcast(c)
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "无效的任务类型: "+req.Type)
	}
}

// CreateScheduledBroadcast creates a looping scheduled broadcast task.
func (h *TaskHandler) CreateScheduledBroadcast(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}
	var req struct {
		AccountIDs          []string       `json:"account_ids"`
		TargetIDs           []string       `json:"target_ids"`
		Payload             map[string]any `json:"payload"`
		IsLoop              *bool          `json:"is_loop"`
		LoopIntervalMinutes int            `json:"loop_interval_minutes"`
		ScheduleMode        string         `json:"schedule_mode"`
		Timezone            string         `json:"timezone"`
		LocalTime           string         `json:"local_time"`
		Weekdays            []int          `json:"weekdays"`
		FirstRunAt          string         `json:"first_run_at"`
		FirstRunLocal       string         `json:"first_run_local"`
		MaxRounds           int            `json:"max_rounds"`
		EndAt               string         `json:"end_at"`
		MinIntervalSeconds  int            `json:"min_interval_seconds"`
		MaxIntervalSeconds  int            `json:"max_interval_seconds"`
		Mappings            []struct {
			AccountID string   `json:"account_id"`
			Targets   []string `json:"targets"`
		} `json:"mappings,omitempty"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "请求参数错误")
	}
	message, _ := req.Payload["message"].(string)
	if message == "" || len(req.TargetIDs) == 0 || (len(req.AccountIDs) == 0 && len(req.Mappings) == 0) {
		return echo.NewHTTPError(http.StatusBadRequest, "定时广播需要指定账号、目标群组和消息内容")
	}
	accountIDs := make([]uuid.UUID, 0, len(req.AccountIDs))
	for _, id := range req.AccountIDs {
		uid, err := uuid.Parse(id)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "账号ID格式错误: "+id)
		}
		accountIDs = append(accountIDs, uid)
	}
	mappings := make([]domain.AccountTargetMapping, 0, len(req.Mappings))
	for _, m := range req.Mappings {
		accountID, parseErr := uuid.Parse(m.AccountID)
		if parseErr != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "映射账号ID格式错误: "+m.AccountID)
		}
		mappings = append(mappings, domain.AccountTargetMapping{AccountID: accountID, Targets: m.Targets})
	}
	isLoop := true
	if req.IsLoop != nil {
		isLoop = *req.IsLoop
	}
	if req.LoopIntervalMinutes < 0 || req.LoopIntervalMinutes > 10080 {
		return echo.NewHTTPError(http.StatusBadRequest, "循环间隔须为 0–10080 分钟")
	}
	interval := req.LoopIntervalMinutes
	if interval == 0 {
		interval = 60
	}
	if req.MaxRounds < 0 || req.MaxRounds > 10000 {
		return echo.NewHTTPError(http.StatusBadRequest, "最大执行轮数须为 0–10000，0 表示不限")
	}

	mode := req.ScheduleMode
	if mode == "" {
		mode = "interval"
	}
	schedule := map[string]any{"schedule_mode": mode}
	var scheduleLocation *time.Location
	switch mode {
	case "interval":
	case "daily", "weekly":
		if !isLoop {
			return echo.NewHTTPError(http.StatusBadRequest, "每日/每周日历排期仅适用于循环任务")
		}
		req.Timezone = strings.TrimSpace(req.Timezone)
		if req.Timezone == "" || req.Timezone == "Local" {
			return echo.NewHTTPError(http.StatusBadRequest, "请指定明确的 IANA 时区，不能使用服务器本地时区")
		}
		location, locationErr := time.LoadLocation(req.Timezone)
		if locationErr != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "时区无效，请选择有效的 IANA 时区")
		}
		scheduleLocation = location
		if _, err := time.Parse("15:04", req.LocalTime); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "固定执行时间格式错误，应为 HH:MM")
		}
		schedule["timezone"] = req.Timezone
		schedule["local_time"] = req.LocalTime
		if mode == "weekly" {
			seen := make(map[int]bool)
			for _, day := range req.Weekdays {
				if day < 1 || day > 7 || seen[day] {
					return echo.NewHTTPError(http.StatusBadRequest, "每周执行日期须为不重复的 1–7（周一至周日）")
				}
				seen[day] = true
			}
			if len(seen) == 0 {
				return echo.NewHTTPError(http.StatusBadRequest, "每周排期至少选择一个星期")
			}
			schedule["weekdays"] = req.Weekdays
		}
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "不支持的循环排期模式")
	}
	if rawMedia, ok := req.Payload["media"]; ok && rawMedia != nil {
		media, valid := rawMedia.(map[string]any)
		if !valid {
			return echo.NewHTTPError(http.StatusBadRequest, "媒体参数格式错误")
		}
		mediaType, _ := media["type"].(string)
		mediaSource, _ := media["source"].(string)
		if (mediaType != "photo" && mediaType != "video") || strings.TrimSpace(mediaSource) == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "媒体类型或来源无效")
		}
		schedule["media"] = map[string]any{"type": mediaType, "source": strings.TrimSpace(mediaSource)}
	}

	var firstRunAt *time.Time
	if req.FirstRunLocal != "" {
		location := scheduleLocation
		if location == nil {
			location = time.Local
		}
		parsed, parseErr := task.ParseLocalDateTime(req.FirstRunLocal, location)
		if parseErr != nil || !parsed.After(time.Now()) {
			return echo.NewHTTPError(http.StatusBadRequest, "首次执行时间必须是所选时区中的有效未来日期和时间")
		}
		firstRunAt = &parsed
	} else if req.FirstRunAt != "" {
		parsed, parseErr := time.Parse(time.RFC3339, req.FirstRunAt)
		if parseErr != nil || !parsed.After(time.Now()) {
			return echo.NewHTTPError(http.StatusBadRequest, "首次执行时间必须是未来的 RFC3339 时间")
		}
		firstRunAt = &parsed
	}
	var endAt *time.Time
	if req.EndAt != "" {
		parsed, parseErr := time.Parse(time.RFC3339, req.EndAt)
		if parseErr != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "结束时间格式错误")
		}
		if !parsed.After(time.Now()) {
			return echo.NewHTTPError(http.StatusBadRequest, "结束时间必须晚于当前时间")
		}
		endAt = &parsed
	}
	if firstRunAt == nil && (mode == "daily" || mode == "weekly") {
		calculated, calcErr := task.NextBroadcastRun(schedule, time.Now(), interval)
		if calcErr != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "无法计算首次执行时间: "+calcErr.Error())
		}
		firstRunAt = &calculated
	}
	if endAt != nil && firstRunAt != nil && !endAt.After(*firstRunAt) {
		return echo.NewHTTPError(http.StatusBadRequest, "自动结束时间必须晚于首次执行时间")
	}
	taskRecord, err := h.svc.CreateScheduledBroadcastWithSchedule(c.Request().Context(), tenantID,
		mappings, accountIDs, req.TargetIDs, message, interval, req.MinIntervalSeconds, req.MaxIntervalSeconds,
		isLoop, req.MaxRounds, endAt, schedule, firstRunAt)
	if err != nil {
		log.Printf("[handler] 创建定时广播任务失败: %v", err)
		log.Printf("[handler] 内部错误已脱敏: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "操作失败，请稍后重试")
	}
	return Success(c, taskRecord)
}

// CreateBroadcast godoc
// @Summary 创建广播任务
// @Description 创建TG消息广播任务，支持笛卡尔积模式和精准映射模式
// @Tags 任务
// @Accept json
// @Produce json
// @Param body body CreateBroadcastRequest true "广播任务请求"
// @Success 201 {object} TaskResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks/broadcast [post]
// CreateBroadcast creates a broadcast task.
// Supports two modes:
//  1. Cartesian (legacy): all accounts send to all targets
//  2. Mapping (precise): each account sends only to its assigned targets
//
// Mapping mode is triggered when req.Mappings is non-empty.
func (h *TaskHandler) CreateBroadcast(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}

	var req CreateBroadcastRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("[handler] 解析群发请求失败: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "请求参数错误")
	}

	// Parse scheduled_at (支持 ISO 格式和 datetime-local 格式)
	var scheduledAt *time.Time
	if req.ScheduledAt != "" {
		var t time.Time
		var err error
		// 先尝试 RFC3339/ISO 格式 (前端 toISOString)
		t, err = time.Parse(time.RFC3339, req.ScheduledAt)
		if err != nil {
			// 回退到 datetime-local 格式
			t, err = time.ParseInLocation("2006-01-02T15:04", req.ScheduledAt, time.Local)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "计划时间格式错误，应为: YYYY-MM-DDTHH:MM")
			}
		}
		scheduledAt = &t
	}

	// Mapping mode: precise account-target control
	if len(req.Mappings) > 0 {
		mappings := make([]domain.AccountTargetMapping, len(req.Mappings))
		for i, m := range req.Mappings {
			accID, err := uuid.Parse(m.AccountID)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "映射账号ID格式错误: "+m.AccountID)
			}
			mappings[i] = domain.AccountTargetMapping{
				AccountID: accID,
				Targets:   m.Targets,
			}
		}
		task, err := h.svc.CreateBroadcastWithMapping(c.Request().Context(), tenantID, mappings, nil, nil, req.Message, scheduledAt, req.IntervalMin, req.IntervalMax, broadcastExtras(&req))
		if err != nil {
			log.Printf("[handler] 创建精准映射群发任务失败: %v", err)
			return echo.NewHTTPError(http.StatusBadRequest, "创建群发任务失败")
		}
		return Created(c, task)
	}

	// Cartesian mode (legacy): all accounts × all targets
	var accountIDs []uuid.UUID
	for _, id := range req.AccountIDs {
		uid, err := uuid.Parse(id)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "账号ID格式错误: "+id)
		}
		accountIDs = append(accountIDs, uid)
	}

	task, err := h.svc.CreateBroadcast(c.Request().Context(), tenantID, accountIDs, req.TargetIDs, req.Message, scheduledAt, req.IntervalMin, req.IntervalMax, broadcastExtras(&req))
	if err != nil {
		log.Printf("[handler] 创建群发任务失败: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "创建群发任务失败")
	}
	return Created(c, task)
}

// CreateScrapeMembersRequest represents scrape members request.
type CreateScrapeMembersRequest struct {
	AccountID string         `json:"account_id" validate:"required"`
	TargetIDs []string       `json:"target_ids" validate:"required,min=1,max=1"`
	Payload   map[string]any `json:"payload,omitempty"` // 扩展参数(如采集过滤 filter)
}

// CreateScrapeLinksRequest represents scrape links request.
type CreateScrapeLinksRequest struct {
	AccountID       string   `json:"account_id" validate:"required"`
	AccountIDs      []string `json:"account_ids,omitempty"` // 兼容数组格式(与其它任务类型一致)
	TargetIDs       []string `json:"target_ids,omitempty"`
	DurationMinutes int      `json:"duration_minutes,omitempty"` // 采集时长(分钟), 0=默认10分钟
}

// CreateScrapeMembers godoc
// @Summary 创建抓取成员任务
// @Description 创建抓取群组成员的任务
// @Tags 任务
// @Accept json
// @Produce json
// @Param body body CreateScrapeMembersRequest true "抓取成员请求"
// @Success 201 {object} TaskResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks/scrape-members [post]
// CreateScrapeMembers creates a scrape members task.
func (h *TaskHandler) CreateScrapeMembers(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}

	var req CreateScrapeMembersRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("[handler] 解析采集成员请求失败: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "请求参数错误")
	}

	accountID, err := uuid.Parse(req.AccountID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "账号ID格式错误")
	}

	var groupID uuid.UUID
	if len(req.TargetIDs) > 0 {
		groupID, err = uuid.Parse(req.TargetIDs[0])
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "群组ID格式错误")
		}
	}
	task, err := h.svc.CreateScrapeMembers(c.Request().Context(), tenantID, accountID, groupID, req.Payload)
	if err != nil {
		log.Printf("[handler] 创建采集成员任务失败: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "创建成员采集任务失败")
	}
	return Created(c, task)
}

// CreateScrapeLinks godoc
// @Summary 创建抓取链接任务
// @Description 创建抓取群组链接的任务
// @Tags 任务
// @Accept json
// @Produce json
// @Param body body CreateScrapeLinksRequest true "抓取链接请求"
// @Success 201 {object} TaskResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks/scrape-links [post]
// CreateScrapeLinks creates a scrape links task.
func (h *TaskHandler) CreateScrapeLinks(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}

	var req CreateScrapeLinksRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("[handler] 解析采集链接请求失败: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "请求参数错误")
	}

	// 兼容 account_id 单数与 account_ids 数组两种格式
	accStr := req.AccountID
	if accStr == "" && len(req.AccountIDs) > 0 {
		accStr = req.AccountIDs[0]
	}
	log.Printf("[handler] scrape_links 收到: account_id=%q account_ids=%v 取accStr=%q", req.AccountID, req.AccountIDs, accStr)
	accountID, err := uuid.Parse(accStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "账号ID格式错误")
	}

	task, err := h.svc.CreateScrapeLinks(c.Request().Context(), tenantID, accountID, req.TargetIDs, req.DurationMinutes)
	if err != nil {
		log.Printf("[handler] 创建采集链接任务失败: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "创建链接采集任务失败")
	}
	return Created(c, task)
}

// CreateJoinGroupsRequest represents join groups request.
type CreateJoinGroupsRequest struct {
	AccountID string   `json:"account_id" validate:"required"`
	Links     []string `json:"links" validate:"required"`
}

// CreateJoinGroups godoc
// @Summary 创建加群任务
// @Description 创建加入Telegram群组的任务
// @Tags 任务
// @Accept json
// @Produce json
// @Param body body CreateJoinGroupsRequest true "加群请求"
// @Success 201 {object} TaskResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks/join-groups [post]
// CreateJoinGroups creates a join groups task.
func (h *TaskHandler) CreateJoinGroups(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}

	var req CreateJoinGroupsRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("[handler] 解析加群请求失败: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "请求参数错误")
	}

	accountID, err := uuid.Parse(req.AccountID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "账号ID格式错误")
	}

	task, err := h.svc.CreateJoinGroups(c.Request().Context(), tenantID, accountID, req.Links)
	if err != nil {
		log.Printf("[handler] 创建加群任务失败: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "创建加群任务失败")
	}
	return Created(c, task)
}

// CreateMonitorScrape godoc
// @Summary 创建消息监控采集任务
// @Description 通过监控群组消息匹配规则来采集成员
// @Tags 任务
// @Accept json
// @Produce json
// @Param body body object true "监控采集请求"
// @Success 201 {object} TaskResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks/monitor-scrape [post]
func (h *TaskHandler) CreateMonitorScrape(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}

	var req struct {
		AccountIDs []string       `json:"account_ids"`
		TargetIDs  []string       `json:"target_ids"`
		Payload    map[string]any `json:"payload"`
	}
	if err := c.Bind(&req); err != nil {
		log.Printf("[handler] 解析监控采集请求失败: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "请求参数错误")
	}

	if len(req.AccountIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "需要指定执行账号")
	}
	if len(req.TargetIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "需要指定目标群组")
	}

	if len(req.AccountIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "account_ids is empty")
	}
	accountID, err := uuid.Parse(req.AccountIDs[0])
	if len(req.TargetIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "target_ids is empty")
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "账号ID格式错误")
	}
	groupID, err := uuid.Parse(req.TargetIDs[0])
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "群组ID格式错误")
	}

	payload := req.Payload
	matchType, _ := payload["match_type"].(string)
	matchContent, _ := payload["match_content"].(string)
	durationMinutes, _ := payload["duration_minutes"].(float64)
	maxResults, _ := payload["max_results"].(float64)

	task, err := h.svc.CreateMonitorScrape(c.Request().Context(), tenantID, accountID, groupID, matchType, matchContent, int(durationMinutes), int(maxResults))
	if err != nil {
		log.Printf("[handler] 创建监控采集任务失败: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "创建监控采集任务失败")
	}
	return Created(c, task)
}
