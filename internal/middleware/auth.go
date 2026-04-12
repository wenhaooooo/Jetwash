package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"jetwash/internal/config"
	"jetwash/internal/repository"
	"jetwash/internal/util"
	"jetwash/pkg/ecode"
)

const (
	// TenantIDKey Context 中存储 TenantID 的键
	TenantIDKey = "tenant_id"
	// APIKeyHeader HTTP Header 中 API Key 的键名
	APIKeyHeader = "X-API-Key"
	// AuthorizationHeader HTTP Header 中 Authorization 的键名
	AuthorizationHeader = "Authorization"
)

// AuthMiddleware 鉴权中间件
// 支持 API Key 和 JWT Token 两种认证方式
func AuthMiddleware(tenantRepo repository.TenantRepository, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tenantID string

		// 优先尝试从 Authorization Header 获取 JWT Token
		authHeader := c.GetHeader(AuthorizationHeader)
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			// 使用 JWT Token 验证
			token := strings.TrimPrefix(authHeader, "Bearer ")
			claims, parseErr := util.ParseToken(token, cfg)
			if parseErr != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"code":    ecode.Unauthorized,
					"message": "Invalid or expired JWT token",
				})
				c.Abort()
				return
			}
			tenantID = claims.TenantID
		} else {
			// 使用 API Key 验证
			apiKey := c.GetHeader(APIKeyHeader)
			if apiKey == "" {
				// 尝试从 Query 参数中获取
				apiKey = c.Query("api_key")
			}

			// 如果 API Key 为空，返回未授权错误
			if apiKey == "" {
				c.JSON(http.StatusUnauthorized, gin.H{
					"code":    ecode.Unauthorized,
					"message": "API Key or JWT token is required",
				})
				c.Abort()
				return
			}

			// 去除 API Key 前后的空格
			apiKey = strings.TrimSpace(apiKey)

			// 根据 API Key 查询租户
			tenant, err := tenantRepo.GetTenantByAPIKey(apiKey)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"code":    ecode.InvalidAPIKey,
					"message": "Invalid API Key",
				})
				c.Abort()
				return
			}

			// 检查租户状态
			if tenant.Status != 1 {
				var message string
				switch tenant.Status {
				case 2:
					message = "Tenant is inactive"
				case 3:
					message = "Tenant is suspended"
				default:
					message = "Tenant is not active"
				}

				c.JSON(http.StatusForbidden, gin.H{
					"code":    ecode.Forbidden,
					"message": message,
				})
				c.Abort()
				return
			}

			tenantID = tenant.ID.String()
		}

		// 将 TenantID 注入到 Context 中
		c.Set(TenantIDKey, tenantID)

		// 继续处理请求
		c.Next()
	}
}

// GetTenantID 从 Context 中获取 TenantID
func GetTenantID(c *gin.Context) (string, bool) {
	tenantID, exists := c.Get(TenantIDKey)
	if !exists {
		return "", false
	}
	return tenantID.(string), true
}

// GetTenantUUID 从 Context 中获取 TenantID 并解析为 UUID
func GetTenantUUID(c *gin.Context) (uuid.UUID, error) {
	tenantIDStr, exists := GetTenantID(c)
	if !exists {
		return uuid.Nil, fmt.Errorf("tenant_id not found in context")
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid tenant_id: %w", err)
	}

	return tenantID, nil
}
