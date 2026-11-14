package handler

import (
	"net/http"
	"tg-sender/internal/middleware"

	"github.com/labstack/echo/v4"
)

// RaffleResults 抽奖参与记录: GET /raffle/results
func (h *TaskHandler) RaffleResults(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}
	list, err := h.svc.RaffleResultList(c.Request().Context(), tenantID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "查询失败")
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"list": list})
}

// RaffleResultsClear R128: 清空全部抽奖参与记录
func (h *TaskHandler) RaffleResultsClear(c echo.Context) error {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}
	if err := h.svc.RaffleResultClear(c.Request().Context(), tenantID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "清空失败")
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"message": "已清空全部参与记录"})
}
