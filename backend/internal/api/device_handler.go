package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nexus-rmm/backend/internal/storage"
)

type DeviceHandler struct {
	db *storage.Database
}

func NewDeviceHandler(db *storage.Database) *DeviceHandler {
	return &DeviceHandler{db: db}
}

type DeviceResponse struct {
	ID             uuid.UUID              `json:"id"`
	AgentID        string                 `json:"agent_id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	OS             string                 `json:"os"`
	OSVersion      string                 `json:"os_version"`
	Architecture   string                 `json:"architecture"`
	Hostname       string                 `json:"hostname"`
	IPAddress      string                 `json:"ip_address"`
	MACAddress     string                 `json:"mac_address"`
	Status         string                 `json:"status"`
	Version        string                 `json:"version"`
	IsOnline       bool                   `json:"is_online"`
	LastSeenAt     *time.Time             `json:"last_seen_at"`
	SiteID         *uuid.UUID             `json:"site_id,omitempty"`
	Site           *SiteResponse          `json:"site,omitempty"`
	GroupID        *uuid.UUID             `json:"group_id,omitempty"`
	Group          *DeviceGroupResponse   `json:"group,omitempty"`
	HardwareInfo   *HardwareInfoResponse  `json:"hardware_info,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

type SiteResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type DeviceGroupResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type HardwareInfoResponse struct {
	CPUModel     string  `json:"cpu_model"`
	CPUCores     int     `json:"cpu_cores"`
	CPUThreads   int     `json:"cpu_threads"`
	RAMTotal     uint64  `json:"ram_total"`
	Manufacturer string  `json:"manufacturer"`
	Model        string  `json:"model"`
	SerialNumber string  `json:"serial_number"`
}

type DeviceListRequest struct {
	Page       int      `form:"page,default=1"`
	Limit      int      `form:"limit,default=20"`
	Search     string   `form:"search"`
	Status     string   `form:"status"`
	OS         string   `form:"os"`
	SiteID     string   `form:"site_id"`
	GroupID    string   `form:"group_id"`
	SortBy     string   `form:"sort_by,default=created_at"`
	SortOrder  string   `form:"sort_order,default=desc"`
}

type DeviceListResponse struct {
	Data       []DeviceResponse `json:"data"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	Limit      int              `json:"limit"`
	TotalPages int              `json:"total_pages"`
}

func (h *DeviceHandler) List(c *gin.Context) {
	var req DeviceListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orgID := c.MustGet("organization_id").(uuid.UUID)

	query := h.db.NewSelect().
		Model((*storage.Device)(nil)).
		Where("organization_id = ?", orgID).
		Relation("Site").
		Relation("Group").
		Relation("HardwareInfo")

	if req.Search != "" {
		search := "%" + strings.ToLower(req.Search) + "%"
		query = query.Where("(LOWER(name) LIKE ? OR LOWER(hostname) LIKE ? OR LOWER(ip_address) LIKE ?)", search, search, search)
	}

	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	if req.OS != "" {
		query = query.Where("os = ?", req.OS)
	}

	if req.SiteID != "" {
		if siteID, err := uuid.Parse(req.SiteID); err == nil {
			query = query.Where("site_id = ?", siteID)
		}
	}

	if req.GroupID != "" {
		if groupID, err := uuid.Parse(req.GroupID); err == nil {
			query = query.Where("group_id = ?", groupID)
		}
	}

	sortOrder := "DESC"
	if strings.ToLower(req.SortOrder) == "asc" {
		sortOrder = "ASC"
	}
	validSortFields := map[string]bool{
		"created_at": true, "updated_at": true, "name": true, "hostname": true,
		"status": true, "last_seen_at": true, "os": true,
	}
	sortField := "created_at"
	if validSortFields[req.SortBy] {
		sortField = req.SortBy
	}
	query = query.Order(sortField + " " + sortOrder)

	total, err := query.Count(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count devices"})
		return
	}

	offset := (req.Page - 1) * req.Limit
	var devices []storage.Device
	err = query.Limit(int64(req.Limit)).Offset(int64(offset)).Scan(c.Request.Context(), &devices)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch devices"})
		return
	}

	response := make([]DeviceResponse, len(devices))
	for i, d := range devices {
		var siteResp *SiteResponse
		if d.Site != nil {
			siteResp = &SiteResponse{ID: d.Site.ID, Name: d.Site.Name}
		}

		var groupResp *DeviceGroupResponse
		if d.Group != nil {
			groupResp = &DeviceGroupResponse{ID: d.Group.ID, Name: d.Group.Name}
		}

		var hwResp *HardwareInfoResponse
		if d.HardwareInfo != nil {
			hwResp = &HardwareInfoResponse{
				CPUModel:     d.HardwareInfo.CPUModel,
				CPUCores:     d.HardwareInfo.CPUCores,
				CPUThreads:   d.HardwareInfo.CPUThreads,
				RAMTotal:     d.HardwareInfo.RAMTotal,
				Manufacturer: d.HardwareInfo.Manufacturer,
				Model:        d.HardwareInfo.Model,
				SerialNumber: d.HardwareInfo.SerialNumber,
			}
		}

		response[i] = DeviceResponse{
			ID:           d.ID,
			AgentID:      d.AgentID,
			Name:         d.Name,
			Description:  d.Description,
			OS:           d.OS,
			OSVersion:    d.OSVersion,
			Architecture: d.Architecture,
			Hostname:     d.Hostname,
			IPAddress:    d.IPAddress,
			MACAddress:   d.MACAddress,
			Status:       d.Status,
			Version:      d.Version,
			IsOnline:     d.IsOnline,
			LastSeenAt:   d.LastSeenAt,
			SiteID:       d.SiteID,
			Site:         siteResp,
			GroupID:      d.GroupID,
			Group:        groupResp,
			HardwareInfo: hwResp,
			CreatedAt:    d.CreatedAt,
			UpdatedAt:    d.UpdatedAt,
		}
	}

	totalPages := int((total + int64(req.Limit) - 1) / int64(req.Limit))

	c.JSON(http.StatusOK, DeviceListResponse{
		Data:       response,
		Total:      total,
		Page:       req.Page,
		Limit:      req.Limit,
		TotalPages: totalPages,
	})
}

func (h *DeviceHandler) Get(c *gin.Context) {
	id := c.Param("id")
	deviceID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device ID"})
		return
	}

	orgID := c.MustGet("organization_id").(uuid.UUID)

	var device storage.Device
	err = h.db.NewSelect().
		Model(&device).
		Where("id = ? AND organization_id = ?", deviceID, orgID).
		Relation("Site").
		Relation("Group").
		Relation("HardwareInfo").
		Relation("HardwareInfo.GPUs").
		Relation("HardwareInfo.Disks").
		Relation("HardwareInfo.Disks.Partitions").
		Relation("HardwareInfo.NetworkInterfaces").
		Scan(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}

	c.JSON(http.StatusOK, device)
}

func (h *DeviceHandler) Create(c *gin.Context) {
	type CreateDeviceRequest struct {
		Name        string     `json:"name" binding:"required"`
		Description string     `json:"description"`
		SiteID      *uuid.UUID `json:"site_id"`
		GroupID     *uuid.UUID `json:"group_id"`
	}

	var req CreateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orgID := c.MustGet("organization_id").(uuid.UUID)

	device := storage.Device{
		Name:           req.Name,
		Description:    req.Description,
		OrganizationID: orgID,
		SiteID:         req.SiteID,
		GroupID:        req.GroupID,
		Status:         "offline",
		IsManaged:      true,
	}

	_, err := h.db.NewInsert().Model(&device).Exec(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create device"})
		return
	}

	c.JSON(http.StatusCreated, device)
}

func (h *DeviceHandler) Update(c *gin.Context) {
	id := c.Param("id")
	deviceID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device ID"})
		return
	}

	orgID := c.MustGet("organization_id").(uuid.UUID)

	var req struct {
		Name        string     `json:"name"`
		Description string     `json:"description"`
		SiteID      *uuid.UUID `json:"site_id"`
		GroupID     *uuid.UUID `json:"group_id"`
		IsManaged   *bool      `json:"is_managed"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err = h.db.NewUpdate().
		Model((*storage.Device)(nil)).
		Set("name = COALESCE(?, name)", req.Name).
		Set("description = COALESCE(?, description)", req.Description).
		Set("site_id = COALESCE(?, site_id)", req.SiteID).
		Set("group_id = COALESCE(?, group_id)", req.GroupID).
		Set("is_managed = COALESCE(?, is_managed)", req.IsManaged).
		Set("updated_at = ?", time.Now()).
		Where("id = ? AND organization_id = ?", deviceID, orgID).
		Exec(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update device"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "device updated"})
}

func (h *DeviceHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	deviceID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device ID"})
		return
	}

	orgID := c.MustGet("organization_id").(uuid.UUID)

	_, err = h.db.NewDelete().
		Model((*storage.Device)(nil)).
		Where("id = ? AND organization_id = ?", deviceID, orgID).
		Exec(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete device"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "device deleted"})
}

func (h *DeviceHandler) GetMetrics(c *gin.Context) {
	id := c.Param("id")
	deviceID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device ID"})
		return
	}

	orgID := c.MustGet("organization_id").(uuid.UUID)

	var device storage.Device
	err = h.db.NewSelect().
		Model(&device).
		Where("id = ? AND organization_id = ?", deviceID, orgID).
		Scan(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}

	since := time.Now().Add(-24 * time.Hour)
	if s := c.Query("since"); s != "" {
		if d, err := time.ParseDuration(s); err == nil {
			since = time.Now().Add(-d)
		}
	}

	var metrics []storage.Metric
	err = h.db.NewSelect().
		Model(&metrics).
		Where("device_id = ? AND timestamp >= ?", deviceID, since).
		Order("timestamp ASC").
		Scan(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch metrics"})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *DeviceHandler) GetSoftwareInventory(c *gin.Context) {
	id := c.Param("id")
	deviceID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device ID"})
		return
	}

	orgID := c.MustGet("organization_id").(uuid.UUID)

	var device storage.Device
	err = h.db.NewSelect().
		Model(&device).
		Where("id = ? AND organization_id = ?", deviceID, orgID).
		Scan(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}

	var software []storage.SoftwareInventory
	err = h.db.NewSelect().
		Model(&software).
		Where("device_id = ?", deviceID).
		Order("name ASC").
		Scan(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch software inventory"})
		return
	}

	c.JSON(http.StatusOK, software)
}

func (h *DeviceHandler) GetAlerts(c *gin.Context) {
	id := c.Param("id")
	deviceID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device ID"})
		return
	}

	orgID := c.MustGet("organization_id").(uuid.UUID)

	var device storage.Device
	err = h.db.NewSelect().
		Model(&device).
		Where("id = ? AND organization_id = ?", deviceID, orgID).
		Scan(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}

	status := c.Query("status")
	query := h.db.NewSelect().
		Model((*storage.Alert)(nil)).
		Where("device_id = ?", deviceID).
		Relation("Policy").
		Order("created_at DESC")

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var alerts []storage.Alert
	err = query.Scan(c.Request.Context(), &alerts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch alerts"})
		return
	}

	c.JSON(http.StatusOK, alerts)
}

func (h *DeviceHandler) RegisterRoutes(r *gin.RouterGroup) {
	devices := r.Group("/devices")
	{
		devices.GET("", h.List)
		devices.POST("", h.Create)
		devices.GET("/:id", h.Get)
		devices.PUT("/:id", h.Update)
		devices.DELETE("/:id", h.Delete)
		devices.GET("/:id/metrics", h.GetMetrics)
		devices.GET("/:id/software", h.GetSoftwareInventory)
		devices.GET("/:id/alerts", h.GetAlerts)
	}
}