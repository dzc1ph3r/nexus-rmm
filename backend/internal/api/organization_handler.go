package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nexus-rmm/backend/internal/storage"
)

type OrganizationHandler struct {
	db *storage.Database
}

func NewOrganizationHandler(db *storage.Database) *OrganizationHandler {
	return &OrganizationHandler{db: db}
}

type OrganizationResponse struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	Description   string    `json:"description"`
	Domain        string    `json:"domain"`
	LogoURL       string    `json:"logo_url"`
	PrimaryColor  string    `json:"primary_color"`
	SecondaryColor string   `json:"secondary_color"`
	IsActive      bool      `json:"is_active"`
	Settings      map[string]interface{} `json:"settings"`
	UserCount     int64     `json:"user_count"`
	DeviceCount   int64     `json:"device_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateOrganizationRequest struct {
	Name        string `json:"name" binding:"required"`
	Slug        string `json:"slug" binding:"required"`
	Description string `json:"description"`
	Domain      string `json:"domain"`
}

type UpdateOrganizationRequest struct {
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Domain         string                 `json:"domain"`
	LogoURL        string                 `json:"logo_url"`
	PrimaryColor   string                 `json:"primary_color"`
	SecondaryColor string                 `json:"secondary_color"`
	IsActive       *bool                  `json:"is_active"`
	Settings       map[string]interface{} `json:"settings"`
}

func (h *OrganizationHandler) List(c *gin.Context) {
	userRole := c.MustGet("user_role").(string)
	userID := c.MustGet("user_id").(uuid.UUID)

	var orgs []storage.Organization
	query := h.db.NewSelect().Model(&orgs)

	if userRole != "admin" && userRole != "super_admin" {
		query = query.Where("id IN (SELECT organization_id FROM users WHERE id = ?)", userID)
	}

	err := query.Order("name ASC").Scan(c.Request.Context(), &orgs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch organizations"})
		return
	}

	response := make([]OrganizationResponse, len(orgs))
	for i, org := range orgs {
		userCount, _ := h.db.NewSelect().
			Model((*storage.User)(nil)).
			Where("organization_id = ?", org.ID).
			Count(c.Request.Context())

		deviceCount, _ := h.db.NewSelect().
			Model((*storage.Device)(nil)).
			Where("organization_id = ?", org.ID).
			Count(c.Request.Context())

		response[i] = OrganizationResponse{
			ID:             org.ID,
			Name:           org.Name,
			Slug:           org.Slug,
			Description:    org.Description,
			Domain:         org.Domain,
			LogoURL:        org.LogoURL,
			PrimaryColor:   org.PrimaryColor,
			SecondaryColor: org.SecondaryColor,
			IsActive:       org.IsActive,
			Settings:       org.Settings,
			UserCount:      userCount,
			DeviceCount:    deviceCount,
			CreatedAt:      org.CreatedAt,
			UpdatedAt:      org.UpdatedAt,
		}
	}

	c.JSON(http.StatusOK, response)
}

func (h *OrganizationHandler) Get(c *gin.Context) {
	id := c.Param("id")
	orgID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	userRole := c.MustGet("user_role").(string)
	userOrgID := c.MustGet("organization_id").(uuid.UUID)

	if userRole != "admin" && userRole != "super_admin" && userOrgID != orgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	var org storage.Organization
	err = h.db.NewSelect().
		Model(&org).
		Where("id = ?", orgID).
		Scan(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		return
	}

	userCount, _ := h.db.NewSelect().
		Model((*storage.User)(nil)).
		Where("organization_id = ?", org.ID).
		Count(c.Request.Context())

	deviceCount, _ := h.db.NewSelect().
		Model((*storage.Device)(nil)).
		Where("organization_id = ?", org.ID).
		Count(c.Request.Context())

	c.JSON(http.StatusOK, OrganizationResponse{
		ID:             org.ID,
		Name:           org.Name,
		Slug:           org.Slug,
		Description:    org.Description,
		Domain:         org.Domain,
		LogoURL:        org.LogoURL,
		PrimaryColor:   org.PrimaryColor,
		SecondaryColor: org.SecondaryColor,
		IsActive:       org.IsActive,
		Settings:       org.Settings,
		UserCount:      userCount,
		DeviceCount:    deviceCount,
		CreatedAt:      org.CreatedAt,
		UpdatedAt:      org.UpdatedAt,
	})
}

func (h *OrganizationHandler) Create(c *gin.Context) {
	userRole := c.MustGet("user_role").(string)
	if userRole != "admin" && userRole != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	var req CreateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	org := storage.Organization{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Domain:      req.Domain,
		IsActive:    true,
		Settings:    map[string]interface{}{},
	}

	_, err := h.db.NewInsert().Model(&org).Exec(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create organization"})
		return
	}

	c.JSON(http.StatusCreated, org)
}

func (h *OrganizationHandler) Update(c *gin.Context) {
	id := c.Param("id")
	orgID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	userRole := c.MustGet("user_role").(string)
	userOrgID := c.MustGet("organization_id").(uuid.UUID)

	if userRole != "admin" && userRole != "super_admin" && userOrgID != orgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	var req UpdateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updateQuery := h.db.NewUpdate().
		Model((*storage.Organization)(nil)).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", orgID)

	if req.Name != "" {
		updateQuery = updateQuery.Set("name = ?", req.Name)
	}
	if req.Description != "" {
		updateQuery = updateQuery.Set("description = ?", req.Description)
	}
	if req.Domain != "" {
		updateQuery = updateQuery.Set("domain = ?", req.Domain)
	}
	if req.LogoURL != "" {
		updateQuery = updateQuery.Set("logo_url = ?", req.LogoURL)
	}
	if req.PrimaryColor != "" {
		updateQuery = updateQuery.Set("primary_color = ?", req.PrimaryColor)
	}
	if req.SecondaryColor != "" {
		updateQuery = updateQuery.Set("secondary_color = ?", req.SecondaryColor)
	}
	if req.IsActive != nil {
		updateQuery = updateQuery.Set("is_active = ?", *req.IsActive)
	}
	if req.Settings != nil {
		updateQuery = updateQuery.Set("settings = ?", req.Settings)
	}

	_, err = updateQuery.Exec(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update organization"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "organization updated"})
}

func (h *OrganizationHandler) Delete(c *gin.Context) {
	userRole := c.MustGet("user_role").(string)
	if userRole != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	id := c.Param("id")
	orgID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	_, err = h.db.NewDelete().
		Model((*storage.Organization)(nil)).
		Where("id = ?", orgID).
		Exec(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete organization"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "organization deleted"})
}

func (h *OrganizationHandler) GetUsers(c *gin.Context) {
	id := c.Param("id")
	orgID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	userRole := c.MustGet("user_role").(string)
	userOrgID := c.MustGet("organization_id").(uuid.UUID)

	if userRole != "admin" && userRole != "super_admin" && userOrgID != orgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	var users []storage.User
	err = h.db.NewSelect().
		Model(&users).
		Where("organization_id = ?", orgID).
		Order("name ASC").
		Scan(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch users"})
		return
	}

	type UserListResponse struct {
		ID             uuid.UUID `json:"id"`
		Email          string    `json:"email"`
		Name           string    `json:"name"`
		Role           string    `json:"role"`
		MFAEnabled     bool      `json:"mfa_enabled"`
		LastLoginAt    *time.Time `json:"last_login_at"`
		IsActive       bool      `json:"is_active"`
		CreatedAt      time.Time `json:"created_at"`
	}

	response := make([]UserListResponse, len(users))
	for i, u := range users {
		response[i] = UserListResponse{
			ID:         u.ID,
			Email:      u.Email,
			Name:       u.Name,
			Role:       u.Role,
			MFAEnabled: u.MFAEnabled,
			LastLoginAt: u.LastLoginAt,
			IsActive:   u.IsActive,
			CreatedAt:  u.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, response)
}

func (h *OrganizationHandler) GetSites(c *gin.Context) {
	id := c.Param("id")
	orgID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	userRole := c.MustGet("user_role").(string)
	userOrgID := c.MustGet("organization_id").(uuid.UUID)

	if userRole != "admin" && userRole != "super_admin" && userOrgID != orgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	var sites []storage.Site
	err = h.db.NewSelect().
		Model(&sites).
		Where("organization_id = ?", orgID).
		Order("name ASC").
		Scan(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch sites"})
		return
	}

	c.JSON(http.StatusOK, sites)
}

func (h *OrganizationHandler) CreateSite(c *gin.Context) {
	id := c.Param("id")
	orgID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	userRole := c.MustGet("user_role").(string)
	userOrgID := c.MustGet("organization_id").(uuid.UUID)

	if userRole != "admin" && userRole != "super_admin" && userOrgID != orgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Address     string `json:"address"`
		City        string `json:"city"`
		State       string `json:"state"`
		Country     string `json:"country"`
		PostalCode  string `json:"postal_code"`
		Timezone    string `json:"timezone"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	site := storage.Site{
		Name:           req.Name,
		Description:    req.Description,
		Address:        req.Address,
		City:           req.City,
		State:          req.State,
		Country:        req.Country,
		PostalCode:     req.PostalCode,
		Timezone:       req.Timezone,
		OrganizationID: orgID,
	}

	_, err = h.db.NewInsert().Model(&site).Exec(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create site"})
		return
	}

	c.JSON(http.StatusCreated, site)
}

func (h *OrganizationHandler) RegisterRoutes(r *gin.RouterGroup) {
	orgs := r.Group("/organizations")
	{
		orgs.GET("", h.List)
		orgs.POST("", h.Create)
		orgs.GET("/:id", h.Get)
		orgs.PUT("/:id", h.Update)
		orgs.DELETE("/:id", h.Delete)
		orgs.GET("/:id/users", h.GetUsers)
		orgs.GET("/:id/sites", h.GetSites)
		orgs.POST("/:id/sites", h.CreateSite)
	}
}