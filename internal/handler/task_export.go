package handler

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"tg-sender/internal/middleware"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// GetLogs godoc
// @Summary 获取任务日志
// @Description 获取指定任务的执行日志
// @Tags 任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {array} domain.TaskLog
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// ListScrapedLinks handles GET /scrape-links?account_id=xxx&since=RFC3339&page=1&page_size=50
// 返回链接采集结果。account_id 必填; since(任务创建时间)可选, 用于圈出单次任务窗口。
func (h *TaskHandler) ListScrapedLinks(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}
	accountIDStr := c.QueryParam("account_id")
	if accountIDStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "account_id is required")
	}
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid account_id")
	}
	var since time.Time
	if sinceStr := c.QueryParam("since"); sinceStr != "" {
		since, err = time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid since (RFC3339)")
		}
	}
	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	links, total, err := h.svc.ListScrapedLinks(c.Request().Context(), tenantID, accountID, since, page, pageSize)
	if err != nil {
		log.Printf("[handler] 内部错误已脱敏: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "操作失败，请稍后重试")
	}
	return c.JSON(http.StatusOK, map[string]any{"items": links, "total": total})
}

// ToggleLinkScraping handles POST /scrape-links/toggle  body: {account_id, enabled}
// 开启/关闭当前账号的常驻链接监听(开启=监听所有群组的邀请链接)。
func (h *TaskHandler) ToggleLinkScraping(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}
	var req struct {
		AccountID string `json:"account_id"`
		Enabled   bool   `json:"enabled"`
		Mode      string `json:"mode"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}
	accountID, err := uuid.Parse(req.AccountID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid account_id")
	}
	if req.Mode == "bio" {
		h.svc.SetBioLinkScraping(accountID, req.Enabled)
		return c.JSON(http.StatusOK, map[string]any{"enabled": h.svc.BioLinkScrapingEnabled(accountID), "mode": "bio"})
	}
	h.svc.SetLinkScraping(c.Request().Context(), tenantID, accountID, req.Enabled)
	return c.JSON(http.StatusOK, map[string]any{"enabled": req.Enabled, "mode": "links"})
}

// LinkScrapingStatus handles GET /scrape-links/status?account_id=xxx
func (h *TaskHandler) LinkScrapingStatus(c echo.Context) error {
	if _, err := middleware.GetTenantID(c); err != nil {
		return err
	}
	accountID, err := uuid.Parse(c.QueryParam("account_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid account_id")
	}
	if c.QueryParam("mode") == "bio" {
		return c.JSON(http.StatusOK, map[string]any{"enabled": h.svc.BioLinkScrapingEnabled(accountID), "mode": "bio"})
	}
	return c.JSON(http.StatusOK, map[string]any{"enabled": h.svc.LinkScrapingEnabled(accountID), "mode": "links"})
}

// ExportScrapedLinks handles GET /scrape-links/export?account_id=xxx&since=RFC3339
// 导出纯文本, 每行一个链接 —— 与前端"自动加群-导入链接"的格式一致, 可直接导入。
func (h *TaskHandler) ExportScrapedLinks(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}
	accountIDStr := c.QueryParam("account_id")
	if accountIDStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "account_id is required")
	}
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid account_id")
	}
	var since time.Time
	if sinceStr := c.QueryParam("since"); sinceStr != "" {
		since, err = time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid since (RFC3339)")
		}
	}
	// 导出上限 5000 条, 足够单次任务使用
	links, _, err := h.svc.ListScrapedLinks(c.Request().Context(), tenantID, accountID, since, 1, 5000)
	if err != nil {
		log.Printf("[handler] 内部错误已脱敏: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "操作失败，请稍后重试")
	}
	// format=csv 时输出带表头的 CSV, 默认 txt(每行一个链接, 与导入器两种格式都兼容)
	if c.QueryParam("format") == "csv" {
		var b strings.Builder
		b.WriteString("link,title,member_count,source_chat\n")
		for _, link := range links {
			b.WriteString(fmt.Sprintf("%s,%s,%d,%s\n",
				csvField(link.FullURL), csvField(link.ResolvedTitle), link.MemberCount, csvField(link.SourceChatTitle)))
		}
		filename := fmt.Sprintf("scrape-links-%s.csv", time.Now().Format("20060102-150405"))
		c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%s", filename))
		return c.Blob(http.StatusOK, "text/csv; charset=utf-8", []byte(b.String()))
	}
	var b strings.Builder
	for _, link := range links {
		b.WriteString(link.FullURL)
		b.WriteString("\n")
	}
	filename := fmt.Sprintf("scrape-links-%s.txt", time.Now().Format("20060102-150405"))
	c.Response().Header().Set(echo.HeaderContentType, "text/plain; charset=utf-8")
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%s", filename))
	return c.String(http.StatusOK, b.String())
}

// csvField 同时处理 CSV 结构转义与电子表格公式注入。
func csvField(s string) string {
	// Excel/LibreOffice 可能忽略公式前的空白和控制字符；检查首个有效字符。
	trimmed := strings.TrimLeftFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r) || unicode.Is(unicode.Cf, r)
	})
	if len(trimmed) < len(s) && len(trimmed) > 0 && strings.ContainsRune("=+-@", rune(trimmed[0])) {
		s = "\t" + s
	} else if len(trimmed) == len(s) && len(s) > 0 && strings.ContainsRune("=+-@", rune(s[0])) {
		s = "\t" + s
	}
	if strings.ContainsAny(s, ",\"\r\n") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}
