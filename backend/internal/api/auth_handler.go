package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nexus-rmm/backend/internal/auth"
	"github.com/nexus-rmm/backend/internal/storage"
)

type AuthHandler struct {
	db           *storage.Database
	authService  *auth.AuthService
}

func NewAuthHandler(db *storage.Database, authService *auth.AuthService) *AuthHandler {
	return &AuthHandler{db: db, authService: authService}
}

func (h *AuthHandler) RegisterRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.RefreshToken)
		auth.POST("/logout", h.AuthMiddleware(), h.Logout)
		auth.GET("/me", h.AuthMiddleware(), h.Me)
		auth.POST("/change-password", h.AuthMiddleware(), h.ChangePassword)

		mfa := auth.Group("/mfa")
		mfa.Use(h.AuthMiddleware())
		{
			mfa.POST("/setup", h.MFASetup)
			mfa.POST("/enable", h.MFAEnable)
			mfa.POST("/disable", h.MFADisable)
			mfa.GET("/status", h.MFAStatus)
		}

		apiKeys := auth.Group("/api-keys")
		apiKeys.Use(h.AuthMiddleware())
		{
			apiKeys.POST("", h.CreateAPIKey)
			apiKeys.GET("", h.ListAPIKeys)
			apiKeys.DELETE("/:id", h.DeleteAPIKey)
		}
	}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	MFA      string `json:"mfa_code"`
	Remember bool   `json:"remember"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	User         UserResponse `json:"user"`
}

type UserResponse struct {
	ID             uuid.UUID `json:"id"`
	Email          string    `json:"email"`
	Name           string    `json:"name"`
	Role           string    `json:"role"`
	OrganizationID uuid.UUID `json:"organization_id"`
	MFAEnabled     bool      `json:"mfa_enabled"`
	LastLoginAt    *time.Time `json:"last_login_at"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user storage.User
	err := h.db.NewSelect().
		Model(&user).
		Where("email = ? AND is_active = true", req.Email).
		Relation("Organization").
		Scan(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if err := h.authService.VerifyPassword(req.Password, user.PasswordHash); err != nil {
		h.db.NewUpdate().
			Model(&user).
			Set("failed_attempts = failed_attempts + 1").
			Set("locked_until = CASE WHEN failed_attempts + 1 >= ? THEN ? ELSE locked_until END",
				h.authService.Config().MaxLoginAttempts,
				time.Now().Add(h.authService.Config().LockoutDuration)).
			WherePK().
			Exec(c.Request.Context())

		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		c.JSON(http.StatusLocked, gin.H{"error": "account locked"})
		return
	}

	if user.MFAEnabled {
		if req.MFA == "" {
			c.JSON(http.StatusOK, gin.H{
				"mfa_required": true,
				"message":      "MFA code required",
			})
			return
		}
		if !h.authService.VerifyMFACode(user.MFASecret, req.MFA) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid MFA code"})
			return
		}
	}

	accessTTL := h.authService.Config().AccessTokenTTL
	if req.Remember {
		accessTTL = 24 * time.Hour
	}

	claims := &auth.UserClaims{
		UserID:         user.ID,
		Email:          user.Email,
		Role:           user.Role,
		OrganizationID: user.OrganizationID,
	}

	accessToken, err := h.authService.GenerateAccessToken(claims)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}

	refreshToken, err := h.authService.GenerateRefreshToken(claims)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}

	refreshHash, _ := h.authService.HashPassword(refreshToken)
	refreshTokenModel := storage.RefreshToken{
		UserID:    user.ID,
		Token:     refreshHash,
		ExpiresAt: time.Now().Add(h.authService.Config().RefreshTokenTTL),
	}
	h.db.NewInsert().Model(&refreshTokenModel).Exec(c.Request.Context())

	now := time.Now()
	h.db.NewUpdate().
		Model(&user).
		Set("last_login_at = ?", now).
		Set("failed_login_attempts = 0").
		Set("locked_until = NULL").
		WherePK().
		Exec(c.Request.Context())

	c.JSON(http.StatusOK, LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(accessTTL.Seconds()),
		TokenType:    "Bearer",
		User: UserResponse{
			ID:             user.ID,
			Email:          user.Email,
			Name:           user.Name,
			Role:           user.Role,
			OrganizationID: user.OrganizationID,
			MFAEnabled:     user.MFAEnabled,
			LastLoginAt:    user.LastLoginAt,
		},
	})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	type RefreshRequest struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	claims, err := h.authService.ParseRefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	var storedToken storage.RefreshToken
	err = h.db.NewSelect().
		Model(&storedToken).
		Where("user_id = ? AND expires_at > ?", claims.UserID, time.Now()).
		Order("created_at DESC").
		Scan(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token not found"})
		return
	}

	if err := h.authService.VerifyPassword(req.RefreshToken, storedToken.Token); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	h.db.NewDelete().
		Model(&storedToken).
		WherePK().
		Exec(c.Request.Context())

	newClaims := &auth.UserClaims{
		UserID:         claims.UserID,
		Email:          claims.Email,
		Role:           claims.Role,
		OrganizationID: claims.OrganizationID,
	}

	accessToken, err := h.authService.GenerateAccessToken(newClaims)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}

	newRefreshToken, err := h.authService.GenerateRefreshToken(newClaims)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}

	newRefreshHash, _ := h.authService.HashPassword(newRefreshToken)
	newRefreshModel := storage.RefreshToken{
		UserID:    claims.UserID,
		Token:     newRefreshHash,
		ExpiresAt: time.Now().Add(h.authService.Config().RefreshTokenTTL),
	}
	h.db.NewInsert().Model(&newRefreshModel).Exec(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
		"expires_in":    int64(h.authService.Config().AccessTokenTTL.Seconds()),
		"token_type":    "Bearer",
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	h.db.NewDelete().
		Model((*storage.RefreshToken)(nil)).
		Where("user_id = ?", userID).
		Exec(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var user storage.User
	err := h.db.NewSelect().
		Model(&user).
		Where("id = ?", userID).
		Relation("Organization").
		Scan(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, UserResponse{
		ID:             user.ID,
		Email:          user.Email,
		Name:           user.Name,
		Role:           user.Role,
		OrganizationID: user.OrganizationID,
		MFAEnabled:     user.MFAEnabled,
		LastLoginAt:    user.LastLoginAt,
	})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	type ChangePasswordRequest struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required"`
	}
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	var user storage.User
	err := h.db.NewSelect().
		Model(&user).
		Where("id = ?", userID).
		Scan(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if err := h.authService.VerifyPassword(req.CurrentPassword, user.PasswordHash); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "current password is incorrect"})
		return
	}

	if err := h.authService.ValidatePasswordStrength(req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newHash, err := h.authService.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	_, err = h.db.NewUpdate().
		Model(&user).
		Set("password_hash = ?", newHash).
		Set("updated_at = ?", time.Now()).
		WherePK().
		Exec(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update password"})
		return
	}

	h.db.NewDelete().
		Model((*storage.RefreshToken)(nil)).
		Where("user_id = ?", userID).
		Exec(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{"message": "password changed"})
}

func (h *AuthHandler) MFASetup(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var user storage.User
	err := h.db.NewSelect().
		Model(&user).
		Where("id = ?", userID).
		Scan(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if user.MFAEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MFA already enabled"})
		return
	}

	secret, err := h.authService.GenerateMFSecret()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate MFA secret"})
		return
	}

	uri := h.authService.GenerateMFAURI(user.Email, secret)

	_, err = h.db.NewUpdate().
		Model(&user).
		Set("mfa_secret = ?", secret).
		WherePK().
		Exec(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save MFA secret"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"secret": secret,
		"uri":    uri,
	})
}

func (h *AuthHandler) MFAEnable(c *gin.Context) {
	type MFAEnableRequest struct {
		Code  string `json:"code" binding:"required"`
		Secret string `json:"secret" binding:"required"`
	}
	var req MFAEnableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	var user storage.User
	err := h.db.NewSelect().
		Model(&user).
		Where("id = ?", userID).
		Scan(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if user.MFASecret != req.Secret {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid secret"})
		return
	}

	if !h.authService.VerifyMFACode(req.Secret, req.Code) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid MFA code"})
		return
	}

	_, err = h.db.NewUpdate().
		Model(&user).
		Set("mfa_enabled = true").
		Set("updated_at = ?", time.Now()).
		WherePK().
		Exec(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enable MFA"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "MFA enabled"})
}

func (h *AuthHandler) MFADisable(c *gin.Context) {
	type MFADisableRequest struct {
		Password string `json:"password" binding:"required"`
		Code     string `json:"code" binding:"required"`
	}
	var req MFADisableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	var user storage.User
	err := h.db.NewSelect().
		Model(&user).
		Where("id = ?", userID).
		Scan(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if err := h.authService.VerifyPassword(req.Password, user.PasswordHash); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password"})
		return
	}

	if !h.authService.VerifyMFACode(user.MFASecret, req.Code) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid MFA code"})
		return
	}

	_, err = h.db.NewUpdate().
		Model(&user).
		Set("mfa_enabled = false").
		Set("mfa_secret = ''").
		Set("updated_at = ?", time.Now()).
		WherePK().
		Exec(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable MFA"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "MFA disabled"})
}

func (h *AuthHandler) MFAStatus(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var user storage.User
	err := h.db.NewSelect().
		Model(&user).
		Where("id = ?", userID).
		Scan(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"mfa_enabled": user.MFAEnabled,
	})
}

func (h *AuthHandler) CreateAPIKey(c *gin.Context) {
	type CreateAPIKeyRequest struct {
		Name        string `json:"name" binding:"required"`
		ExpiresDays *int   `json:"expires_days"`
	}
	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)
	orgID := c.MustGet("organization_id").(uuid.UUID)

	key, hash, err := h.authService.GenerateAPIKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate API key"})
		return
	}

	var expiresAt *time.Time
	if req.ExpiresDays != nil {
		t := time.Now().AddDate(0, 0, *req.ExpiresDays)
		expiresAt = &t
	}

	apiKey := storage.APIKey{
		OrganizationID: orgID,
		UserID:         userID,
		Name:           req.Name,
		KeyHash:        hash,
		ExpiresAt:      expiresAt,
	}

	_, err = h.db.NewInsert().Model(&apiKey).Exec(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save API key"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         apiKey.ID,
		"name":       apiKey.Name,
		"key":        key,
		"expires_at": apiKey.ExpiresAt,
		"created_at": apiKey.CreatedAt,
	})
}

func (h *AuthHandler) ListAPIKeys(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var keys []storage.APIKey
	err := h.db.NewSelect().
		Model(&keys).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Scan(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch API keys"})
		return
	}

	type APIKeyResponse struct {
		ID        uuid.UUID  `json:"id"`
		Name      string     `json:"name"`
		LastUsed  *time.Time `json:"last_used"`
		ExpiresAt *time.Time `json:"expires_at"`
		CreatedAt time.Time  `json:"created_at"`
		IsActive  bool       `json:"is_active"`
	}

	response := make([]APIKeyResponse, len(keys))
	for i, k := range keys {
		response[i] = APIKeyResponse{
			ID:        k.ID,
			Name:      k.Name,
			LastUsed:  k.LastUsedAt,
			ExpiresAt: k.ExpiresAt,
			CreatedAt: k.CreatedAt,
			IsActive:  k.IsActive,
		}
	}

	c.JSON(http.StatusOK, response)
}

func (h *AuthHandler) DeleteAPIKey(c *gin.Context) {
	id := c.Param("id")
	keyID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid API key ID"})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	_, err = h.db.NewDelete().
		Model((*storage.APIKey)(nil)).
		Where("id = ? AND user_id = ?", keyID, userID).
		Exec(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete API key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key deleted"})
}

func (h *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			return
		}

		claims, err := h.authService.ParseToken(authHeader)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)
		c.Set("organization_id", claims.OrganizationID)

		c.Next()
	}
}