package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nexus-rmm/backend/internal/auth"
)

func AuthMiddleware(authService *auth.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			return
		}

		claims, err := authService.ParseToken(parts[1])
		if err != nil {
			if errors.Is(err, auth.ErrTokenExpired) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token expired", "code": "TOKEN_EXPIRED"})
			} else {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			}
			return
		}

		if claims.TokenType != "access" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token type"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)
		c.Set("organization_id", claims.OrganizationID)
		c.Set("claims", claims)

		c.Next()
	}
}

func OptionalAuthMiddleware(authService *auth.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		claims, err := authService.ParseToken(parts[1])
		if err != nil {
			c.Next()
			return
		}

		if claims.TokenType == "access" {
			c.Set("user_id", claims.UserID)
			c.Set("user_email", claims.Email)
			c.Set("user_role", claims.Role)
			c.Set("organization_id", claims.OrganizationID)
			c.Set("claims", claims)
		}

		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("user_role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "role not found"})
			return
		}

		userRole := role.(string)
		allowed := false
		for _, r := range roles {
			if userRole == r {
				allowed = true
				break
			}
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}

		c.Next()
	}
}

func RequireOrganizationAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, exists := c.Get("organization_id")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "organization context required"})
			return
		}

		resourceOrgID := c.Param("organization_id")
		if resourceOrgID != "" {
			parsedOrgID, err := uuid.Parse(resourceOrgID)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
				return
			}
			if orgID.(uuid.UUID) != parsedOrgID {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied to this organization"})
				return
			}
		}

		c.Next()
	}
}

func TenantIsolationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, exists := c.Get("organization_id")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "organization context required"})
			return
		}

		c.Set("tenant_id", orgID)
		c.Next()
	}
}

type RateLimiter struct {
	requests map[string]int
	window   time.Duration
	limit    int
	mu       sync.Mutex
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]int),
		window:   window,
		limit:    limit,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	count := rl.requests[key]
	if count >= rl.limit {
		return false
	}
	rl.requests[key] = count + 1
	return true
}

func (rl *RateLimiter) RetryAfter(key string) time.Duration {
	return rl.window
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		rl.requests = make(map[string]int)
		rl.mu.Unlock()
	}
}

func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()
		if userID, exists := c.Get("user_id"); exists {
			key = "user:" + userID.(uuid.UUID).String()
		}

		if !limiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": limiter.RetryAfter(key).String(),
			})
			return
		}

		c.Next()
	}
}

func AuditMiddleware(auditService AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		userID, _ := c.Get("user_id")
		orgID, _ := c.Get("organization_id")

		var uid, oid uuid.UUID
		if userID != nil {
			uid = userID.(uuid.UUID)
		}
		if orgID != nil {
			oid = orgID.(uuid.UUID)
		}

		auditService.Log(c.Request.Context(), AuditLogEntry{
			UserID:         uid,
			OrganizationID: oid,
			Action:         method + " " + path,
			Resource:       extractResource(path),
			ResourceID:     c.Param("id"),
			IPAddress:      c.ClientIP(),
			UserAgent:      c.GetHeader("User-Agent"),
			Status:         getStatus(c.Writer.Status()),
			Duration:       time.Since(start).Milliseconds(),
		})
	}
}

func extractResource(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}

func getStatus(code int) string {
	if code >= 200 && code < 300 {
		return "success"
	}
	return "failure"
}

type AuditService interface {
	Log(ctx context.Context, entry AuditLogEntry)
}

type AuditLogEntry struct {
	UserID         uuid.UUID
	OrganizationID uuid.UUID
	Action         string
	Resource       string
	ResourceID     string
	IPAddress      string
	UserAgent      string
	Status         string
	Duration       int64
}