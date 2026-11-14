package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"tg-sender/internal/domain"
	"tg-sender/internal/domain/task"
	"tg-sender/internal/middleware"
	"tg-sender/internal/sse"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
)

// @Router /tasks/{id}/logs [get]
// DeleteTaskLogs handles DELETE /tasks/:id/logs 清空该任务的日志。
func (h *TaskHandler) DeleteTaskLogs(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}
	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid task id")
	}
	if err := h.svc.DeleteTaskLogs(c.Request().Context(), tenantID, taskID); err != nil {
		log.Printf("[handler] 内部错误已脱敏: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "操作失败，请稍后重试")
	}
	return c.JSON(http.StatusOK, map[string]any{"success": true})
}

// fillLogMessages 给无 message 的历史日志补可读文案。
func fillLogMessages(logs []*domain.TaskLog) {
	for _, l := range logs {
		if l.Message == "" {
			l.Message = task.FormatLogMessage(l)
		}
	}
}

// GetLogs returns task logs.
func (h *TaskHandler) GetLogs(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}

	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "任务ID格式错误")
	}

	page := 1
	if value, parseErr := strconv.Atoi(c.QueryParam("page")); parseErr == nil && value > 0 {
		page = value
	}
	pageSize := 50
	if value, parseErr := strconv.Atoi(c.QueryParam("page_size")); parseErr == nil && value > 0 {
		pageSize = min(value, 200)
	}

	var from, to *time.Time
	for _, bound := range []struct {
		name string
		dest **time.Time
	}{{"from", &from}, {"to", &to}} {
		raw := strings.TrimSpace(c.QueryParam(bound.name))
		if raw == "" {
			continue
		}
		parsed, parseErr := time.Parse(time.RFC3339Nano, raw)
		if parseErr != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "日志时间范围格式错误，需使用 RFC3339")
		}
		*bound.dest = &parsed
	}
	if from != nil && to != nil && !from.Before(*to) {
		return echo.NewHTTPError(http.StatusBadRequest, "日志开始时间必须早于结束时间")
	}

	var logs []*domain.TaskLog
	if cursor := c.QueryParam("before_log_id"); cursor != "" {
		beforeLogID, parseErr := uuid.Parse(cursor)
		if parseErr != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "日志游标格式错误")
		}
		if from != nil || to != nil {
			logs, err = h.svc.GetLogsBeforeInRange(c.Request().Context(), tenantID, taskID, beforeLogID, from, to, pageSize)
		} else {
			logs, err = h.svc.GetLogsBefore(c.Request().Context(), tenantID, taskID, beforeLogID, pageSize)
		}
	} else if from != nil || to != nil {
		logs, err = h.svc.GetLogsInRange(c.Request().Context(), tenantID, taskID, from, to, page, pageSize)
	} else {
		logs, err = h.svc.GetLogs(c.Request().Context(), tenantID, taskID, page, pageSize)
	}
	if err != nil {
		log.Printf("[handler] 获取任务日志失败: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "服务器内部错误")
	}
	fillLogMessages(logs)
	return Success(c, logs)
}

// GetStatsByAccount godoc
// @Summary 单任务按账号聚合成功/失败/跳过
// @Tags tasks
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} map[string]map[string]int
// @Router /tasks/{id}/stats-by-account [get]
func (h *TaskHandler) GetStatsByAccount(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}
	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "无效的任务ID")
	}
	stats, err := h.svc.TaskLogStatsByAccount(c.Request().Context(), tenantID, taskID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "统计失败")
	}
	return c.JSON(http.StatusOK, stats)
}

// GetAccountLogs godoc
// @Summary 获取账号日志
// @Description 获取指定账号的所有任务日志（独立日志页面用）
// @Tags 任务
// @Accept json
// @Produce json
// @Param account_id query string true "账号ID"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(50)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks/logs [get]
// GetAccountLogs returns all task logs for a specific account.
func (h *TaskHandler) GetAccountLogs(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}

	accountIDStr := c.QueryParam("account_id")
	if accountIDStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "需要指定账号ID")
	}
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "账号ID格式错误")
	}

	page := 1
	if p := c.QueryParam("page"); p != "" {
		if pi, err := strconv.Atoi(p); err == nil && pi > 0 {
			page = pi
		}
	}

	pageSize := 50
	if ps := c.QueryParam("page_size"); ps != "" {
		if psi, err := strconv.Atoi(ps); err == nil && psi > 0 && psi <= 200 {
			pageSize = psi
		}
	}

	logs, total, err := h.svc.GetLogsByAccount(c.Request().Context(), tenantID, accountID, page, pageSize)
	if err != nil {
		log.Printf("[handler] 获取账号日志失败: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "服务器内部错误")
	}

	return SuccessList(c, logs, total, page, pageSize)
}

// GetProgress godoc
// @Summary 获取任务进度
// @Description 获取指定任务的执行进度
// @Tags 任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} domain.TaskProgress
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id}/progress [get]
// GetProgress returns task progress.
func (h *TaskHandler) GetProgress(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}

	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "任务ID格式错误")
	}

	task, err := h.svc.GetByID(c.Request().Context(), tenantID, taskID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "任务不存在")
	}
	return Success(c, task.Progress)
}

// StreamProgress godoc
// @Summary 任务进度SSE流
// @Description 通过SSE实时推送任务进度
// @Tags 任务
// @Accept json
// @Produce text/event-stream
// @Param id path string true "任务ID"
// @Success 200 {object} domain.TaskProgress
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id}/progress/stream [get]
func (h *TaskHandler) StreamProgress(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}

	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "任务ID格式错误")
	}
	// Verify tenant ownership before committing the streaming response or allocating
	// a long-lived broker subscription. Unknown/foreign task IDs must fail fast.
	initialTask, err := h.svc.GetByID(c.Request().Context(), tenantID, taskID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "任务不存在")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "查询任务失败")
	}

	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	// R98: 订阅本租户事件流 —— 执行器 publishLog 经 BroadcastToTenant 发到
	// "tenant:<租户>:events" topic(双层包装: {type:"events", payload:{...内层...}})。
	// 此处解包内层, 把本任务的 task_log / task_status 实时转发给详情页连接。
	var evCh chan sse.Event
	if h.sseBroker != nil {
		topic := fmt.Sprintf("tenant:%s:events", tenantID.String())
		subID, ch := h.sseBroker.Subscribe(topic)
		defer h.sseBroker.Unsubscribe(topic, subID)
		evCh = ch
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Send the authorized snapshot immediately; later ticks use the live cache/DB.
	initialData, _ := json.Marshal(initialTask.Progress)
	fmt.Fprintf(c.Response(), "data: %s\n\n", initialData)
	c.Response().Flush()

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case ev, ok := <-evCh:
			if !ok {
				// broker 关闭后 channel 会被关闭, 不判 ok 会拿到零值事件并空转刷屏。
				return nil
			}
			// R98: 实时事件转发(仅本任务) —— 解包 BroadcastToTenant 的双层包装
			if outer, ok := ev.Payload.(map[string]interface{}); ok {
				m, _ := outer["payload"].(map[string]interface{})
				// task_id 是 uuid 对象(未经 JSON 序列化), 用 %v 统一成字符串比较
				tid := fmt.Sprintf("%v", m["task_id"])
				if tid == taskID.String() && (m["type"] == "task_log" || m["type"] == "task_status" || m["type"] == "task_wait") {
					if bts, err := json.Marshal(m); err == nil {
						fmt.Fprintf(c.Response(), "data: %s\n\n", bts)
						c.Response().Flush()
					}
				}
			}
		case <-ticker.C:
			// 优先读引擎内存进度缓存：N 个前端同时看进度时 DB QPS 归零
			// （此前每个连接每秒查一次 tasks 表整行）。未命中回退查库。
			if h.engine != nil {
				if p, ok := h.engine.GetProgress(taskID); ok {
					data, _ := json.Marshal(p)
					fmt.Fprintf(c.Response(), "data: %s\n\n", data)
					c.Response().Flush()
					continue
				}
			}
			currentTask, err := h.svc.GetByID(c.Request().Context(), tenantID, taskID)
			if err != nil {
				continue
			}
			data, _ := json.Marshal(currentTask.Progress)
			fmt.Fprintf(c.Response(), "data: %s\n\n", data)
			c.Response().Flush()
		}
	}
}
