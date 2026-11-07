package handler

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"tg-sender/internal/domain"
	"tg-sender/internal/domain/grouplib"
	"tg-sender/internal/middleware"
	"tg-sender/internal/repository"
	"tg-sender/internal/service"
)

// GroupLibraryHandler 平台级群组库(找群)。所有登录租户可见, 不含来源租户身份。
type GroupLibraryHandler struct {
	repo       *repository.SystemGroupRepository
	taskSvc    *service.TaskService
	accRepo    *repository.AccountRepository
	lib        *grouplib.Engine
	planLimits *repository.PlanLimitRepository
}

func NewGroupLibraryHandler(repo *repository.SystemGroupRepository, taskSvc *service.TaskService, accRepo *repository.AccountRepository, lib *grouplib.Engine, planLimits *repository.PlanLimitRepository) *GroupLibraryHandler {
	return &GroupLibraryHandler{repo: repo, taskSvc: taskSvc, accRepo: accRepo, lib: lib, planLimits: planLimits}
}

// Search handles POST /group-library/search {"keyword": "...", "account_id": "可选"}
// 用本租户一个在线账号调 TG 全局搜索, 搜到的公开群/频道直接入库, 并同步返回结果。
func (h *GroupLibraryHandler) Search(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}
	var req struct {
		Keyword   string    `json:"keyword"`
		AccountID uuid.UUID `json:"account_id"`
	}
	if err := c.Bind(&req); err != nil || strings.TrimSpace(req.Keyword) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "keyword required")
	}
	// 未指定账号则取本租户第一个在线账号
	accID := req.AccountID
	if accID == uuid.Nil {
		accs, err := h.accRepo.List(c.Request().Context(), tenantID, "")
		if err != nil || len(accs) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "没有可用账号, 请先添加并上线一个账号")
		}
		for _, a := range accs {
			if a.Status == domain.AccountStatusOnline {
				accID = a.AccountID
				break
			}
		}
		if accID == uuid.Nil {
			return echo.NewHTTPError(http.StatusBadRequest, "没有在线账号, 请先上线一个再搜索")
		}
	}
	// 套餐日额度(051): 免费体验版限 3 次/天
	// V19 审计修复: 原来 err != nil 时静默跳过限额检查(fail-open)——DB 抖动=免费版无限搜群。
	// 改 fail-closed: 限额读不出来就拒绝本次操作(宁可误伤不可漏检), 并提示稍后再试。
	if h.planLimits != nil {
		_, dailySearch, err := h.planLimits.GetPlanLimits(c.Request().Context(), tenantID)
		if err != nil {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "额度服务暂不可用, 请稍后重试")
		}
		if dailySearch > 0 {
			// P2-47 审计修复: 原来 AddDailyUsage(+1) 在超限判断之前——超限的这次也扣了
			// 次数。改为先查当前用量, 超限直接拒绝不扣, 没超才 +1。
			used, err := h.planLimits.GetDailyUsage(c.Request().Context(), tenantID, repository.KindGroupSearch)
			if err != nil {
				return echo.NewHTTPError(http.StatusServiceUnavailable, "用量服务暂不可用, 请稍后重试")
			}
			if used >= dailySearch {
				return echo.NewHTTPError(http.StatusTooManyRequests, fmt.Sprintf("今日搜群次数已用完(体验版 %d 次/天), 升级套餐不限量", dailySearch))
			}
			if _, err := h.planLimits.AddDailyUsage(c.Request().Context(), tenantID, repository.KindGroupSearch, 1); err != nil {
				return echo.NewHTTPError(http.StatusServiceUnavailable, "用量记录失败, 请稍后重试")
			}
		}
	}
	ingested, err := h.lib.SearchAndIngest(c.Request().Context(), tenantID, accID, req.Keyword)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"list": ingested, "count": len(ingested)})
}

// List handles GET /group-library?keyword=&gtype=&status=&source=&page=&page_size=
func (h *GroupLibraryHandler) List(c echo.Context) error {
	if _, err := middleware.GetTenantID(c); err != nil {
		return err
	}
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(c.QueryParam("page_size"))
	f := repository.SystemGroupFilter{
		Keyword:  c.QueryParam("keyword"),
		GType:    c.QueryParam("gtype"),
		Status:   c.QueryParam("status"),
		Source:   c.QueryParam("source"),
		Page:     page,
		PageSize: size,
	}
	list, total, err := h.repo.List(c.Request().Context(), f)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": size,
	})
}

// Import handles POST /group-library/import
// body: {"account_id": "可选", "text": "每行一个 t.me 链接 / @用户名 / 邀请链接, 支持粘贴整段文本"}
// 公开群/频道自动解析群名、成员数、类型; 私密邀请链接占位入库, 后台解析器自动补名。
func (h *GroupLibraryHandler) Import(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}
	const maxImportBytes = 20 << 20
	var payload, fileName string
	if strings.HasPrefix(strings.ToLower(c.Request().Header.Get(echo.HeaderContentType)), "multipart/form-data") {
		if err := c.Request().ParseMultipartForm(maxImportBytes); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "无法读取导入文件")
		}
		file, header, err := c.Request().FormFile("file")
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "请选择 txt/csv 文件")
		}
		defer file.Close()
		if header.Size > maxImportBytes {
			return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "文件不能超过 20MB")
		}
		body, err := io.ReadAll(io.LimitReader(file, maxImportBytes+1))
		if err != nil || len(body) > maxImportBytes {
			return echo.NewHTTPError(http.StatusBadRequest, "文件读取失败或超过 20MB")
		}
		payload, fileName = string(body), header.Filename
	} else {
		var req struct {
			Text     string `json:"text"`
			FileName string `json:"file_name"`
		}
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "导入内容格式错误")
		}
		payload, fileName = req.Text, req.FileName
		if len(payload) > maxImportBytes {
			return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "导入内容不能超过 20MB")
		}
	}
	if strings.TrimSpace(payload) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "导入内容不能为空")
	}
	job, err := h.lib.CreateImportPreview(c.Request().Context(), tenantID, fileName, payload)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusAccepted, job)
}

// ConfirmImportJob handles POST /group-library/import-jobs/:id/confirm.
// 只有用户确认预筛选结果后，任务才会进入 Telegram 群组解析队列。
func (h *GroupLibraryHandler) ConfirmImportJob(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
	}
	job, err := h.lib.ConfirmImportJob(c.Request().Context(), tenantID, jobID)
	if err != nil {
		return echo.NewHTTPError(http.StatusConflict, "导入预览不存在或已经确认")
	}
	return c.JSON(http.StatusOK, job)
}

// RetryFailedImportJob handles POST /group-library/import-jobs/:id/retry.
func (h *GroupLibraryHandler) RetryFailedImportJob(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
	}
	job, err := h.lib.RetryFailedImportJob(c.Request().Context(), tenantID, jobID)
	if err != nil {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}
	return c.JSON(http.StatusAccepted, job)
}

// CancelImportJob handles POST /group-library/import-jobs/:id/cancel.
func (h *GroupLibraryHandler) CancelImportJob(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
	}
	job, err := h.lib.CancelImportJob(c.Request().Context(), tenantID, jobID)
	if err != nil {
		return echo.NewHTTPError(http.StatusConflict, "导入任务不存在、已结束或无法取消")
	}
	return c.JSON(http.StatusOK, job)
}

// ListImportJobs handles GET /group-library/import-jobs.
func (h *GroupLibraryHandler) ListImportJobs(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}
	jobs, err := h.lib.ListImportJobs(c.Request().Context(), tenantID, 30)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "读取导入任务失败")
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"list": jobs})
}

// GetImportJob handles GET /group-library/import-jobs/:id.
func (h *GroupLibraryHandler) GetImportJob(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
	}
	job, err := h.lib.GetImportJob(c.Request().Context(), tenantID, jobID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "导入任务不存在")
	}
	return c.JSON(http.StatusOK, job)
}

// JoinSelected handles POST /group-library/join
// body: {"account_ids": ["..."], "sg_ids": ["..."]}
// 对每个勾选的账号创建一个加群任务(链接集 = 勾选群的去重 t.me 链接)。
func (h *GroupLibraryHandler) JoinSelected(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}
	var req struct {
		AccountIDs []uuid.UUID `json:"account_ids"`
		SGIDs      []uuid.UUID `json:"sg_ids"`
		// 分批加入: per_account_count>0 时把链接按账号轮流切块,
		// 每个账号只加自己那一份。0 = 兼容旧行为: 全部群给每个账号都加一遍。
		PerAccountCount int `json:"per_account_count"`
		// 加群间隔(分钟): 复用任务层 CreateJoinGroupsWithInterval,
		// 区间随机取值; 引擎侧 FloodWait 自动冷却逻辑不变。
		IntervalMinMinutes int `json:"interval_min_minutes"`
		IntervalMaxMinutes int `json:"interval_max_minutes"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}
	if len(req.AccountIDs) == 0 || len(req.AccountIDs) > 10 {
		return echo.NewHTTPError(http.StatusBadRequest, "account_ids 数量须 1-10")
	}
	if len(req.SGIDs) == 0 || len(req.SGIDs) > 100 {
		return echo.NewHTTPError(http.StatusBadRequest, "sg_ids 数量须 1-100")
	}
	if req.IntervalMaxMinutes > 0 && req.IntervalMaxMinutes < req.IntervalMinMinutes {
		return echo.NewHTTPError(http.StatusBadRequest, "间隔上限不能小于下限")
	}

	sgs, err := h.repo.GetByIDs(c.Request().Context(), req.SGIDs)
	if err != nil || len(sgs) == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "未找到勾选的群组")
	}
	links := make([]string, 0, len(sgs))
	seen := map[string]bool{}
	for _, sg := range sgs {
		link := sg.TmeLink
		if link == "" && sg.Username != "" {
			link = "https://t.me/" + sg.Username
		}
		if link == "" || seen[link] {
			continue
		}
		seen[link] = true
		links = append(links, link)
	}

	created := 0
	var lastErr error
	if req.PerAccountCount > 0 {
		// 分批模式: 链接按账号轮流切块(round-robin), 每个账号只加自己那一份。
		chunks := make([][]string, len(req.AccountIDs))
		for i, link := range links {
			bucket := (i / req.PerAccountCount) % len(req.AccountIDs)
			chunks[bucket] = append(chunks[bucket], link)
		}
		for i, accID := range req.AccountIDs {
			if len(chunks[i]) == 0 {
				continue
			}
			if _, err := h.taskSvc.CreateJoinGroupsWithInterval(c.Request().Context(), tenantID, accID, chunks[i], req.IntervalMinMinutes, req.IntervalMaxMinutes); err != nil {
				lastErr = err
				continue
			}
			created++
		}
	} else {
		// 兼容旧行为: 全部群给每个账号都加一遍
		for _, accID := range req.AccountIDs {
			if _, err := h.taskSvc.CreateJoinGroupsWithInterval(c.Request().Context(), tenantID, accID, links, req.IntervalMinMinutes, req.IntervalMaxMinutes); err != nil {
				lastErr = err
				continue
			}
			created++
		}
	}
	if created == 0 && lastErr != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, lastErr.Error())
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"created":    created,
		"links":      len(links),
		"failed_err": errString(lastErr),
	})
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

var _ = domain.SystemGroup{} // 保留 domain 引用(响应由 repo 结构直接序列化)
