package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nexus-rmm/backend/internal/storage"
)

type RemoteHandler struct {
	db *storage.Database
}

func NewRemoteHandler(db *storage.Database) *RemoteHandler {
	return &RemoteHandler{db: db}
}

type CreateSessionRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
	Type     string `json:"type" binding:"required,oneof=desktop terminal file_manager"`
	Protocol string `json:"protocol,omitempty"`
	Quality  string `json:"quality,omitempty"`
}

type SessionResponse struct {
	ID             uuid.UUID  `json:"id"`
	SessionID      string     `json:"session_id"`
	DeviceID       uuid.UUID  `json:"device_id"`
	DeviceName     string     `json:"device_name"`
	UserID         uuid.UUID  `json:"user_id"`
	Type           string     `json:"type"`
	Status         string     `json:"status"`
	Protocol       string     `json:"protocol"`
	Quality        string     `json:"quality"`
	Resolution     string     `json:"resolution"`
	StartedAt      time.Time  `json:"started_at"`
	EndedAt        *time.Time `json:"ended_at"`
	Duration       int64      `json:"duration"`
	RecordingPath  string     `json:"recording_path"`
	Participants   []ParticipantResponse `json:"participants"`
}

type ParticipantResponse struct {
	UserID     uuid.UUID `json:"user_id"`
	UserName   string    `json:"user_name"`
	Role       string    `json:"role"`
	JoinedAt   time.Time `json:"joined_at"`
	LeftAt     *time.Time `json:"left_at"`
	Permissions []string `json:"permissions"`
}

type JoinSessionRequest struct {
	Role        string   `json:"role" binding:"required,oneof=viewer controller"`
	Permissions []string `json:"permissions"`
}

type SessionEventRequest struct {
	Type string                 `json:"type" binding:"required"`
	Data map[string]interface{} `json:"data"`
}

func (h *RemoteHandler) CreateSession(c *gin.Context) {
	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deviceID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device ID"})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)
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

	if !device.IsOnline {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device is offline"})
		return
	}

	session := storage.RemoteSession{
		SessionID:   uuid.New().String(),
		DeviceID:     deviceID,
		UserID:       userID,
		Type:         req.Type,
		Status:       "pending",
		Protocol:     req.Protocol,
		Quality:      req.Quality,
		StartedAt:    time.Now(),
		IPAddress:    c.ClientIP(),
	}

	_, err = h.db.NewInsert().Model(&session).Exec(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	participant := storage.SessionParticipant{
		SessionID:  session.ID,
		UserID:     userID,
		Role:       "host",
		JoinedAt:   time.Now(),
		Permissions: []string{"control", "view", "chat", "file_transfer"},
	}

	_, err = h.db.NewInsert().Model(&participant).Exec(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add participant"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"session_id": session.SessionID,
		"device_id":  device.ID,
		"device_name": device.Name,
		"type":       session.Type,
		"status":     session.Status,
	})
}

func (h *RemoteHandler) GetSession(c *gin.Context) {
	id := c.Param("id")
	sessionID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	orgID := c.MustGet("organization_id").(uuid.UUID)

	var session storage.RemoteSession
	err = h.db.NewSelect().
		Model(&session).
		Where("id = ?", sessionID).
		Relation("Device").
		Relation("User").
		Scan(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	if session.Device.OrganizationID != orgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Query participants separately
	var participants []storage.SessionParticipant
	err = h.db.NewSelect().
		Model(&participants).
		Where("session_id = ?", session.ID).
		Relation("User").
		Scan(c.Request.Context())
	if err != nil {
		participants = []storage.SessionParticipant{}
	}

	participantResponses := make([]ParticipantResponse, len(participants))
	for i, p := range participants {
		participantResponses[i] = ParticipantResponse{
			UserID:     p.UserID,
			UserName:   p.User.Name,
			Role:       p.Role,
			JoinedAt:   p.JoinedAt,
			LeftAt:     p.LeftAt,
			Permissions: p.Permissions,
		}
	}

	c.JSON(http.StatusOK, SessionResponse{
		ID:            session.ID,
		SessionID:     session.SessionID,
		DeviceID:      session.DeviceID,
		DeviceName:    session.Device.Name,
		UserID:        session.UserID,
		Type:          session.Type,
		Status:        session.Status,
		Protocol:      session.Protocol,
		Quality:       session.Quality,
		Resolution:    session.Resolution,
		StartedAt:     session.StartedAt,
		EndedAt:       session.EndedAt,
		Duration:      session.Duration,
		RecordingPath: session.RecordingPath,
		Participants:  participantResponses,
	})
}

func (h *RemoteHandler) ListSessions(c *gin.Context) {
	orgID := c.MustGet("organization_id").(uuid.UUID)
	userID := c.MustGet("user_id").(uuid.UUID)
	userRole := c.MustGet("user_role").(string)

	status := c.Query("status")
	deviceID := c.Query("device_id")

	query := h.db.NewSelect().
		Model((*storage.RemoteSession)(nil)).
		Relation("Device").
		Relation("User").
		Where("device.organization_id = ?", orgID)

	if userRole != "admin" && userRole != "super_admin" {
		query = query.Where("user_id = ?", userID)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if deviceID != "" {
		if dID, err := uuid.Parse(deviceID); err == nil {
			query = query.Where("device_id = ?", dID)
		}
	}

	var sessions []storage.RemoteSession
	err := query.Order("started_at DESC").Limit(50).Scan(c.Request.Context(), &sessions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch sessions"})
		return
	}

	type SessionListResponse struct {
		ID          uuid.UUID  `json:"id"`
		SessionID   string     `json:"session_id"`
		DeviceID    uuid.UUID  `json:"device_id"`
		DeviceName  string     `json:"device_name"`
		UserID      uuid.UUID  `json:"user_id"`
		UserName    string     `json:"user_name"`
		Type        string     `json:"type"`
		Status      string     `json:"status"`
		StartedAt   time.Time  `json:"started_at"`
		EndedAt     *time.Time `json:"ended_at"`
		Duration    int64      `json:"duration"`
	}

	response := make([]SessionListResponse, len(sessions))
	for i, s := range sessions {
		response[i] = SessionListResponse{
			ID:         s.ID,
			SessionID:  s.SessionID,
			DeviceID:   s.DeviceID,
			DeviceName: s.Device.Name,
			UserID:     s.UserID,
			UserName:   s.User.Name,
			Type:       s.Type,
			Status:     s.Status,
			StartedAt:  s.StartedAt,
			EndedAt:    s.EndedAt,
			Duration:   s.Duration,
		}
	}

	c.JSON(http.StatusOK, response)
}

func (h *RemoteHandler) JoinSession(c *gin.Context) {
	id := c.Param("id")
	sessionID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	var req JoinSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)
	orgID := c.MustGet("organization_id").(uuid.UUID)

	var session storage.RemoteSession
	err = h.db.NewSelect().
		Model(&session).
		Where("id = ?", sessionID).
		Relation("Device").
		Scan(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	if session.Device.OrganizationID != orgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if session.Status != "active" && session.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session is not active"})
		return
	}

	var existing storage.SessionParticipant
	err = h.db.NewSelect().
		Model(&existing).
		Where("session_id = ? AND user_id = ?", session.ID, userID).
		Scan(c.Request.Context())

	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "already joined this session"})
		return
	}

	permissions := req.Permissions
	if len(permissions) == 0 {
		switch req.Role {
		case "controller":
			permissions = []string{"control", "view", "chat", "file_transfer"}
		case "viewer":
			permissions = []string{"view", "chat"}
		}
	}

	participant := storage.SessionParticipant{
		SessionID:  session.ID,
		UserID:     userID,
		Role:       req.Role,
		JoinedAt:   time.Now(),
		Permissions: permissions,
	}

	_, err = h.db.NewInsert().Model(&participant).Exec(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to join session"})
		return
	}

	if session.Status == "pending" {
		h.db.NewUpdate().
			Model(&session).
			Set("status = ?", "active").
			WherePK().
			Exec(c.Request.Context())
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "joined session",
		"session_id": session.SessionID,
	})
}

func (h *RemoteHandler) LeaveSession(c *gin.Context) {
	id := c.Param("id")
	sessionID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	_, err = h.db.NewUpdate().
		Model((*storage.SessionParticipant)(nil)).
		Set("left_at = ?", time.Now()).
		Where("session_id = ? AND user_id = ?", sessionID, userID).
		Exec(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to leave session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "left session"})
}

func (h *RemoteHandler) EndSession(c *gin.Context) {
	id := c.Param("id")
	sessionID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)
	orgID := c.MustGet("organization_id").(uuid.UUID)

	var session storage.RemoteSession
	err = h.db.NewSelect().
		Model(&session).
		Where("id = ?", sessionID).
		Relation("Device").
		Scan(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	if session.Device.OrganizationID != orgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if session.UserID != userID {
		userRole := c.MustGet("user_role").(string)
		if userRole != "admin" && userRole != "super_admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "only session host can end session"})
			return
		}
	}

	now := time.Now()
	duration := int64(now.Sub(session.StartedAt).Seconds())

	_, err = h.db.NewUpdate().
		Model(&session).
		Set("status = ?", "ended").
		Set("ended_at = ?", now).
		Set("duration = ?", duration).
		WherePK().
		Exec(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to end session"})
		return
	}

	h.db.NewUpdate().
		Model((*storage.SessionParticipant)(nil)).
		Set("left_at = ?", now).
		Where("session_id = ? AND left_at IS NULL", sessionID).
		Exec(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{"message": "session ended"})
}

func (h *RemoteHandler) AddSessionEvent(c *gin.Context) {
	id := c.Param("id")
	sessionID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	var req SessionEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)
	orgID := c.MustGet("organization_id").(uuid.UUID)

	var session storage.RemoteSession
	err = h.db.NewSelect().
		Model(&session).
		Where("id = ?", sessionID).
		Relation("Device").
		Scan(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	if session.Device.OrganizationID != orgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	event := storage.SessionEvent{
		SessionID: session.ID,
		UserID:    userID,
		Type:      req.Type,
		Data:      req.Data,
		Timestamp: time.Now(),
	}

	_, err = h.db.NewInsert().Model(&event).Exec(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add event"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "event recorded"})
}

func (h *RemoteHandler) RegisterRoutes(r *gin.RouterGroup) {
	remote := r.Group("/remote")
	{
		remote.POST("/sessions", h.CreateSession)
		remote.GET("/sessions", h.ListSessions)
		remote.GET("/sessions/:id", h.GetSession)
		remote.POST("/sessions/:id/join", h.JoinSession)
		remote.POST("/sessions/:id/leave", h.LeaveSession)
		remote.POST("/sessions/:id/end", h.EndSession)
		remote.POST("/sessions/:id/events", h.AddSessionEvent)
	}
}