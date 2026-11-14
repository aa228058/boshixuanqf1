package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
)

// APIErrorHandler 统一错误出口。
//
// 背景: handler 里有 60 多处 `echo.NewHTTPError(500, err.Error())`，Echo 默认会把
// 这段文案原样写回响应体。SQL 报错里通常带表名、列名、约束名，连接错误里可能带
// DSN 片段——这些都是内部实现细节，不该出现在客户端响应里（前端 utils/request.ts
// 已经在按 SQLSTATE/constraint 关键字二次过滤，说明这个泄漏确实发生过）。
//
// 规则: 4xx 保留业务文案（前端依赖这些中文提示），5xx 一律只回通用提示，完整错误
// 走服务端日志。
func APIErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	code := http.StatusInternalServerError
	var message any = "internal server error"

	var he *echo.HTTPError
	if errors.As(err, &he) {
		code = he.Code
		if code < http.StatusInternalServerError {
			message = he.Message
			if m, ok := message.(error); ok {
				message = m.Error()
			}
		}
	}

	if code >= http.StatusInternalServerError {
		log.Printf("[api] %s %s -> %d: %v", c.Request().Method, c.Request().URL.Path, code, err)
	}

	if c.Request().Method == http.MethodHead {
		if noContentErr := c.NoContent(code); noContentErr != nil {
			log.Printf("[api] write %d response failed: %v", code, noContentErr)
		}
		return
	}

	if writeErr := c.JSON(code, map[string]any{
		"code":    code,
		"message": message,
	}); writeErr != nil {
		log.Printf("[api] write %d response failed: %v", code, writeErr)
	}
}
