package response

import (
	"net/http"

	"jetwash/pkg/ecode"

	"github.com/gin-gonic/gin"
)

// Response 统一 API 响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// OK 返回成功响应
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    ecode.Success,
		Message: "success",
		Data:    data,
	})
}

// OKWithMessage 返回带自定义消息的成功响应
func OKWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    ecode.Success,
		Message: message,
		Data:    data,
	})
}

// Created 返回创建成功响应
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Code:    ecode.Success,
		Message: "created",
		Data:    data,
	})
}

// Error 返回错误响应，使用 ecode.Ecode
func Error(c *gin.Context, e ecode.Ecode) {
	status := mapCodeToHTTPStatus(e.Code())
	c.JSON(status, Response{
		Code:    e.Code(),
		Message: e.Message(),
	})
}

// ErrorWithMessage 返回带自定义消息的错误响应
func ErrorWithMessage(c *gin.Context, e ecode.Ecode, message string) {
	status := mapCodeToHTTPStatus(e.Code())
	c.JSON(status, Response{
		Code:    e.Code(),
		Message: message,
	})
}

// mapCodeToHTTPStatus 将业务错误码映射为 HTTP 状态码
func mapCodeToHTTPStatus(code int) int {
	switch {
	case code == ecode.Success:
		return http.StatusOK
	case code == ecode.Unauthorized || code == ecode.InvalidAPIKey:
		return http.StatusUnauthorized
	case code == ecode.Forbidden || code == ecode.TenantInactive || code == ecode.TenantSuspended:
		return http.StatusForbidden
	case code == ecode.NotFound || code == ecode.TenantNotFound || code == ecode.WordNotFound || code == ecode.HistoryNotFound:
		return http.StatusNotFound
	case code == ecode.InvalidParams || code == ecode.WordAlreadyExists || code == ecode.TextTooLong:
		return http.StatusBadRequest
	case code == ecode.TooManyRequests:
		return http.StatusTooManyRequests
	case code == ecode.RequestTimeout:
		return http.StatusRequestTimeout
	default:
		return http.StatusInternalServerError
	}
}
