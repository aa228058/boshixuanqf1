package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tg-sender/internal/domain"
	"tg-sender/internal/domain/task"
	"tg-sender/internal/middleware"
	"tg-sender/internal/repository"
	"tg-sender/internal/safego"
	"tg-sender/internal/service"
	"tg-sender/internal/sse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
)

// TaskHandler handles task APIs.
type TaskHandler struct {
	svc         *service.TaskService
	planChecker *middleware.PlanChecker
	auditRepo   *repository.AuditRepository
	engine      *task.Engine
	sseBroker   *sse.Broker // R98: 任务详情流实时转发日志用
}

// NewTaskHandler creates a new task handler.
func NewTaskHandler(svc *service.TaskService, planChecker *middleware.PlanChecker, auditRepo *repository.AuditRepository, engine *task.Engine) *TaskHandler {
	return &TaskHandler{svc: svc, planChecker: planChecker, auditRepo: auditRepo, engine: engine}
}

// SetSSEBroker injects the SSE broker (R98: StreamProgress 订阅事件流转发实时日志).
func (h *TaskHandler) SetSSEBroker(b *sse.Broker) { h.sseBroker = b }

// TaskListResponse represents a list of tasks response.
type TaskListResponse struct {
	Data     []*domain.Task `json:"data"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

// TaskResponse represents a single task response.
type TaskResponse struct {
	Data *domain.Task `json:"data"`
}

// List godoc
// @Summary 获取任务列表
// @Description 获取租户下的所有任务，支持按状态筛选和分页
// @Tags 任务
// @Accept json
// @Produce json
// @Param status query string false "任务状态筛选" Enums(pending,running,paused,completed,failed,cancelled)
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} TaskListResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks [get]
// List returns tasks for the tenant with pagination.
// ValidateTargets 发送前目标预检: 批量校验目标有效性, 返回可直接发送的与无效目标(附原因)
func (h *TaskHandler) ValidateTargets(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}
	var req struct {
		TargetIDs []string `json:"target_ids"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	valid, invalid := h.svc.ValidateTargets(c.Request().Context(), tenantID, req.TargetIDs)
	return c.JSON(http.StatusOK, map[string]any{
		"total":   len(req.TargetIDs),
		"valid":   valid,
		"invalid": invalid,
	})
}

func (h *TaskHandler) List(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}

	statusStr := c.QueryParam("status")
	var status *domain.TaskStatus
	if statusStr != "" {
		s := domain.TaskStatus(statusStr)
		status = &s
	}

	page := 1
	if p := c.QueryParam("page"); p != "" {
		if pi, err := strconv.Atoi(p); err == nil && pi > 0 {
			page = pi
		}
	}

	pageSize := 20
	if ps := c.QueryParam("page_size"); ps != "" {
		if psi, err := strconv.Atoi(ps); err == nil && psi > 0 && psi <= 100 {
			pageSize = psi
		}
	}

	// Parse optional account_id filter for account-level isolation
	var accountID *uuid.UUID
	var taskType *domain.TaskType
	if tStr := c.QueryParam("type"); tStr != "" {
		t := domain.TaskType(tStr)
		taskType = &t
	}
	accountID, err = parseOptionalUUIDParam(c.QueryParam("account_id"), "account_id")
	if err != nil {
		return err
	}

	// hide_loops=1 时隐藏循环包装任务(任务列表页用, 只显示实际执行记录)
	hideLoops := c.QueryParam("hide_loops") == "1"

	// 时间范围筛选: range=today(今日) | 7d(近7日) | 缺省=全部(兼容旧调用)
	var since *time.Time
	switch c.QueryParam("range") {
	case "today":
		t := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())
		since = &t
	case "7d":
		t := time.Now().AddDate(0, 0, -7)
		since = &t
	}

	// 精确编号搜索：保留 UUID 作为详情/执行主键，编号仅作为全局展示索引。
	if numberStr := strings.TrimSpace(c.QueryParam("task_number")); numberStr != "" {
		number, parseErr := strconv.ParseInt(numberStr, 10, 64)
		if parseErr != nil || number <= 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "任务编号格式错误")
		}
		found, findErr := h.svc.GetByNumber(c.Request().Context(), tenantID, number)
		if findErr != nil {
			if errors.Is(findErr, pgx.ErrNoRows) {
				return SuccessList(c, []*domain.Task{}, 0, 1, 20)
			}
			log.Printf("[handler] 按编号查询任务失败: %v", findErr)
			return echo.NewHTTPError(http.StatusInternalServerError, "服务器内部错误")
		}
		return SuccessList(c, []*domain.Task{found}, 1, 1, 1)
	}

	tasks, err := h.svc.List(c.Request().Context(), tenantID, status, accountID, taskType, hideLoops, since, page, pageSize)
	if err != nil {
		log.Printf("[handler] 获取任务列表失败: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "服务器内部错误")
	}
	total, err := h.svc.Count(c.Request().Context(), tenantID, status, accountID, taskType, since)
	if err != nil {
		total = len(tasks)
	}

	return SuccessList(c, tasks, total, page, pageSize)
}

// BatchTaskIDsRequest represents a batch operation on multiple tasks.
type BatchTaskIDsRequest struct {
	TaskIDs []string `json:"task_ids" validate:"required,min=1"`
}

// Pause godoc
// @Summary 暂停任务
// @Description 暂停指定任务
// @Tags 任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id}/pause [post]
// Pause pauses a task.
func (h *TaskHandler) Pause(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}

	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "任务ID格式错误")
	}

	if err := h.svc.Pause(c.Request().Context(), tenantID, taskID); err != nil {
		log.Printf("[handler] 暂停任务失败: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "暂停任务失败")
	}
	return Success(c, map[string]string{"status": "paused"})
}

// Resume godoc
// @Summary 恢复任务
// @Description 恢复已暂停的任务
// @Tags 任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id}/resume [post]
// Resume resumes a task.
func (h *TaskHandler) Resume(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}

	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "任务ID格式错误")
	}

	if err := h.svc.Resume(c.Request().Context(), tenantID, taskID); err != nil {
		log.Printf("[handler] 恢复任务失败: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "恢复任务失败")
	}
	return Success(c, map[string]string{"status": "resumed"})
}

// Cancel godoc
// @Summary 取消任务
// @Description 取消指定任务
// @Tags 任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id}/cancel [post]
// Cancel cancels a task.
func (h *TaskHandler) Cancel(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}

	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "任务ID格式错误")
	}

	if err := h.svc.Cancel(c.Request().Context(), tenantID, taskID); err != nil {
		log.Printf("[handler] 取消任务失败: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "取消任务失败")
	}
	return Success(c, map[string]string{"status": "cancelled"})
}

// Retry godoc
// @Summary 重试任务
// @Description 重试失败的任务
// @Tags 任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id}/retry [post]
// Retry retries a failed task.
func (h *TaskHandler) Retry(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}

	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "任务ID格式错误")
	}

	// scope=failed 时仅对失败目标断点续发(克隆新任务), 否则整任务重置重跑
	scope := c.QueryParam("scope")

	if err := h.svc.Retry(c.Request().Context(), tenantID, taskID, scope); err != nil {
		log.Printf("[handler] 重试任务失败: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "重试任务失败")
	}
	return Success(c, map[string]string{"status": "retrying"})
}

// CleanupLogs handles DELETE /logs/cleanup?days=N 手动清理 N 天前日志。
// R116: CleanupLogs handler 已删(/logs/cleanup 路由删除后零调用)

// LeaveAllGroups godoc
// @Summary 一键退出账号加入的全部群组(频道/超级群)
// @Tags tasks
// @Produce json
// @Param id path string true "账号ID"
// @Success 200 {object} task.LeaveAllResult
// @Router /accounts/{id}/leave-all-groups [post]
func (h *TaskHandler) LeaveAllGroups(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "无效的账号ID")
	}
	res, err := h.svc.LeaveAllGroups(c.Request().Context(), tenantID, accountID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusOK, res)
}

// GetByID godoc
// @Summary 获取任务详情
// @Description 获取指定任务的详细信息
// @Tags 任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} TaskResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id} [get]
// GetByID returns a task by ID.
func (h *TaskHandler) GetByID(c echo.Context) error {
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
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "任务不存在")
		}
		log.Printf("[handler] 获取任务详情失败: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "服务器内部错误")
	}
	return Success(c, task)
}

// Update godoc
// @Summary 更新任务
// @Description 更新待执行或已暂停的任务
// @Tags 任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Param body body object true "更新内容"
// @Success 200 {object} TaskResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id} [put]
// Update updates a task.
func (h *TaskHandler) Update(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}
	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "任务ID格式错误")
	}
	var req struct {
		AccountIDs []string       `json:"account_ids"`
		TargetIDs  []string       `json:"target_ids"`
		Payload    map[string]any `json:"payload"`
		Status     string         `json:"status"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "请求参数错误")
	}
	// 状态流转：暂停/取消/恢复
	if req.Status != "" {
		ctx := c.Request().Context()
		switch req.Status {
		case string(domain.TaskStatusPaused):
			err = h.svc.Pause(ctx, tenantID, taskID)
		case string(domain.TaskStatusCancelled):
			err = h.svc.Cancel(ctx, tenantID, taskID)
		case string(domain.TaskStatusRunning), string(domain.TaskStatusQueued):
			err = h.svc.Resume(ctx, tenantID, taskID)
		default:
			return echo.NewHTTPError(http.StatusBadRequest, "不支持的状态: "+req.Status)
		}
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return Success(c, map[string]string{"status": "updated"})
	}
	accIDs := make([]uuid.UUID, 0, len(req.AccountIDs))
	for _, id := range req.AccountIDs {
		uid, err := uuid.Parse(id)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "账号ID格式错误: "+id)
		}
		accIDs = append(accIDs, uid)
	}
	if err := h.svc.Update(c.Request().Context(), tenantID, taskID, accIDs, req.TargetIDs, req.Payload); err != nil {
		log.Printf("[handler] 更新任务失败: %v", err)
		log.Printf("[handler] 内部错误已脱敏: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "操作失败，请稍后重试")
	}
	return Success(c, map[string]string{"status": "updated"})
}

// Delete godoc
// @Summary 删除任务
// @Description 删除指定任务
// @Tags 任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id} [delete]
// Delete removes a task.
func (h *TaskHandler) Delete(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}

	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "任务ID格式错误")
	}

	if err := h.svc.Delete(c.Request().Context(), tenantID, taskID); err != nil {
		log.Printf("[handler] 删除任务失败: %v", err)
		if strings.Contains(err.Error(), "cannot delete task with status") {
			return echo.NewHTTPError(http.StatusConflict, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "删除任务失败")
	}
	return NoContent(c)
}

// parseBatchTaskIDs validates and parses a bounded batch task ID list.
func parseBatchTaskIDs(raw []string) ([]uuid.UUID, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("task_ids must contain at least one task ID")
	}
	if len(raw) > 500 {
		return nil, fmt.Errorf("task_ids must not contain more than 500 IDs")
	}
	ids := make([]uuid.UUID, 0, len(raw))
	for _, id := range raw {
		uid, err := uuid.Parse(id)
		if err != nil {
			return nil, fmt.Errorf("invalid task id: %s", id)
		}
		ids = append(ids, uid)
	}
	return ids, nil
}

// BatchCancel godoc
// @Summary 批量取消任务
// @Description 批量取消多个任务
// @Tags 任务
// @Accept json
// @Produce json
// @Param request body BatchTaskIDsRequest true "任务ID列表"
// @Success 200 {object} map[string]int
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks/batch/cancel [post]
func (h *TaskHandler) BatchCancel(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}
	var req BatchTaskIDsRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	taskIDs, err := parseBatchTaskIDs(req.TaskIDs)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	cancelled, failed, err := h.svc.BatchCancel(c.Request().Context(), tenantID, taskIDs)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "批量取消失败")
	}
	// F-31: 批量操作审计
	// 崩溃类修复: 脱协程必须用 safego —— 裸 go 里 panic 会崩整个进程
	// (Echo Recover 只罩请求 goroutine)。
	safego.Go(func() {
		if err := h.auditRepo.Create(context.Background(), &domain.AuditLog{
			TenantID: tenantID, UserID: "tenant", Action: "task_batch_cancel",
			Target: fmt.Sprintf("success=%d,failed=%d", cancelled, failed), IP: c.RealIP(),
		}); err != nil {
			log.Printf("[audit] write failed: %v", err)
		}
	})
	return Success(c, map[string]int{"cancelled": cancelled, "failed": failed})
}

// BatchDelete godoc
// @Summary 批量删除任务
// @Description 批量删除多个已完成/失败/已取消的任务
// @Tags 任务
// @Accept json
// @Produce json
// @Param request body BatchTaskIDsRequest true "任务ID列表"
// @Success 200 {object} map[string]int
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks/batch/delete [post]
func (h *TaskHandler) BatchDelete(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}
	var req BatchTaskIDsRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	taskIDs, err := parseBatchTaskIDs(req.TaskIDs)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	deleted, failed, err := h.svc.BatchDelete(c.Request().Context(), tenantID, taskIDs)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "批量删除失败")
	}
	// F-31: 批量操作审计
	// 崩溃类修复: 同 BatchCancel, 脱协程必须 safego。
	safego.Go(func() {
		if err := h.auditRepo.Create(context.Background(), &domain.AuditLog{
			TenantID: tenantID, UserID: "tenant", Action: "task_batch_delete",
			Target: fmt.Sprintf("success=%d,failed=%d", deleted, failed), IP: c.RealIP(),
		}); err != nil {
			log.Printf("[audit] write failed: %v", err)
		}
	})
	return Success(c, map[string]int{"deleted": deleted, "failed": failed})
}
