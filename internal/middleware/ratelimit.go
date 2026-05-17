package middleware

import (
	"fmt"
	"net/http"
	"time"

	"jetwash/internal/cache"

	"github.com/gin-gonic/gin"
)

// RateLimit 返回一个基于 Redis 的每租户限流中间件
// 使用固定时间窗口算法，每分钟重置计数
func RateLimit(redisClient *cache.RedisClient, requestsPerMinute int) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, exists := GetTenantID(c)
		if !exists {
			// 没有 tenant_id 时跳过限流（不应发生在已认证路由上）
			c.Next()
			return
		}

		// 按租户 + 当前分钟生成 key
		key := fmt.Sprintf("ratelimit:%s:%d", tenantID, time.Now().Unix()/60)

		count, err := redisClient.Incr(c.Request.Context(), key)
		if err != nil {
			// Redis 故障时放行，不影响正常业务
			c.Next()
			return
		}

		// 首次请求时设置 2 分钟过期（比窗口长，确保窗口内不会被清除）
		if count == 1 {
			redisClient.Expire(c.Request.Context(), key, 2*time.Minute)
		}

		if count > int64(requestsPerMinute) {
			c.Header("Retry-After", "60")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    1006,
				"message": "Rate limit exceeded",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
