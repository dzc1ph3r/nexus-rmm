package storage

import (
	"time"

	"github.com/google/uuid"
)

type BaseModel struct {
	ID        uuid.UUID `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp"`
	DeletedAt *time.Time `bun:"deleted_at,soft_delete,nullzero"`
}

type User struct {
	BaseModel
	Email          string    `bun:"email,unique,notnull"`
	PasswordHash   string    `bun:"password_hash,notnull"`
	Name           string    `bun:"name,notnull"`
	Role           string    `bun:"role,notnull,default:'technician'"`
	MFAEnabled     bool      `bun:"mfa_enabled,notnull,default:false"`
	MFASecret      string    `bun:"mfa_secret"`
	LastLoginAt    *time.Time `bun:"last_login_at"`
	FailedAttempts int       `bun:"failed_attempts,notnull,default:0"`
	LockedUntil    *time.Time `bun:"locked_until"`
	IsActive       bool      `bun:"is_active,notnull,default:true"`
	OrganizationID uuid.UUID `bun:"organization_id,type:uuid,notnull"`
	Organization   *Organization `bun:"rel:belongs-to,join:organization_id=id"`
	// Has-many relations removed to avoid bun v1.3 schema creation issues
}

type Organization struct {
	BaseModel
	Name        string `bun:"name,notnull"`
	Slug        string `bun:"slug,unique,notnull"`
	Description string `bun:"description"`
	Domain      string `bun:"domain"`
	LogoURL     string `bun:"logo_url"`
	PrimaryColor string `bun:"primary_color"`
	SecondaryColor string `bun:"secondary_color"`
	IsActive    bool   `bun:"is_active,notnull,default:true"`
	Settings    map[string]interface{} `bun:"settings,type:jsonb,default:'{}'"`
	// Has-many relations removed to avoid bun v1.3 schema creation issues
	// The belongs-to relations on the "many" side handle foreign keys
}

type Site struct {
	BaseModel
	Name          string `bun:"name,notnull"`
	Description   string `bun:"description"`
	Address       string `bun:"address"`
	City          string `bun:"city"`
	State         string `bun:"state"`
	Country       string `bun:"country"`
	PostalCode    string `bun:"postal_code"`
	Timezone      string `bun:"timezone"`
	OrganizationID uuid.UUID `bun:"organization_id,type:uuid,notnull"`
	Organization  *Organization `bun:"rel:belongs-to,join:organization_id=id"`
	// Has-many relations removed to avoid bun v1.3 schema creation issues
	// The belongs-to relations on the "many" side handle foreign keys
}

type Device struct {
	BaseModel
	AgentID          string    `bun:"agent_id,unique,notnull"`
	Name             string    `bun:"name,notnull"`
	Description      string    `bun:"description"`
	OS               string    `bun:"os,notnull"`
	OSVersion        string    `bun:"os_version"`
	Architecture     string    `bun:"architecture"`
	Hostname         string    `bun:"hostname"`
	IPAddress        string    `bun:"ip_address"`
	MACAddress       string    `bun:"mac_address"`
	LastSeenAt       *time.Time `bun:"last_seen_at"`
	Status           string    `bun:"status,notnull,default:'offline'"`
	Version          string    `bun:"version"`
	IsOnline         bool      `bun:"is_online,notnull,default:false"`
	IsManaged        bool      `bun:"is_managed,notnull,default:true"`
	OrganizationID   uuid.UUID `bun:"organization_id,type:uuid,notnull"`
	Organization     *Organization `bun:"rel:belongs-to,join:organization_id=id"`
	SiteID           *uuid.UUID `bun:"site_id,type:uuid"`
	Site             *Site      `bun:"rel:belongs-to,join:site_id=id"`
	GroupID          *uuid.UUID `bun:"group_id,type:uuid"`
	Group            *DeviceGroup `bun:"rel:belongs-to,join:group_id=id"`

	HardwareInfo     *HardwareInfo     `bun:"rel:has-one,join:id=device_id"`
	// Has-many relations removed to avoid bun v1.3 schema creation issues
	// The belongs-to relations on the "many" side handle foreign keys
}

type DeviceGroup struct {
	BaseModel
	Name           string `bun:"name,notnull"`
	Description    string `bun:"description"`
	OrganizationID uuid.UUID `bun:"organization_id,type:uuid,notnull"`
	Organization   *Organization `bun:"rel:belongs-to,join:organization_id=id"`
	ParentID       *uuid.UUID `bun:"parent_id,type:uuid"`
	Parent         *DeviceGroup `bun:"rel:belongs-to,join:parent_id=id"`
	// Has-many relations removed to avoid bun v1.3 schema creation issues
}

type HardwareInfo struct {
	BaseModel
	DeviceID     uuid.UUID `bun:"device_id,type:uuid,notnull,unique"`
	Device       *Device   `bun:"rel:belongs-to,join:device_id=id"`
	CPUModel     string    `bun:"cpu_model"`
	CPUCores     int       `bun:"cpu_cores"`
	CPUThreads   int       `bun:"cpu_threads"`
	CPUFrequency string    `bun:"cpu_frequency"`
	RAMTotal     uint64    `bun:"ram_total"`
	RAMType      string    `bun:"ram_type"`
	RAMSpeed     string    `bun:"ram_speed"`
	Motherboard  string    `bun:"motherboard"`
	BIOSVersion  string    `bun:"bios_version"`
	BIOSDate     string    `bun:"bios_date"`
	ChassisType  string    `bun:"chassis_type"`
	Manufacturer string    `bun:"manufacturer"`
	Model        string    `bun:"model"`
	SerialNumber string    `bun:"serial_number"`
	GPUs         []GPUInfo `bun:"-"`
	Disks        []DiskInfo `bun:"-"`
	NetworkInterfaces []NetworkInterfaceInfo `bun:"-"`
}

type GPUInfo struct {
	BaseModel
	HardwareInfoID uuid.UUID `bun:"hardware_info_id,type:uuid,notnull"`
	HardwareInfo   *HardwareInfo `bun:"rel:belongs-to,join:hardware_info_id=id"`
	Name           string    `bun:"name"`
	Vendor         string    `bun:"vendor"`
	VRAM           uint64    `bun:"vram"`
	DriverVersion  string    `bun:"driver_version"`
}

type DiskInfo struct {
	BaseModel
	HardwareInfoID uuid.UUID `bun:"hardware_info_id,type:uuid,notnull"`
	HardwareInfo   *HardwareInfo `bun:"rel:belongs-to,join:hardware_info_id=id"`
	Name           string    `bun:"name"`
	Model          string    `bun:"model"`
	SerialNumber   string    `bun:"serial_number"`
	Size           uint64    `bun:"size"`
	Type           string    `bun:"type"` // SSD, HDD, NVMe
	Interface      string    `bun:"interface"` // SATA, NVMe, etc.
	Health         string    `bun:"health"`
	Temperature    int       `bun:"temperature"`
	Partitions     []PartitionInfo `bun:"-"`
}

type PartitionInfo struct {
	BaseModel
	DiskInfoID   uuid.UUID `bun:"disk_info_id,type:uuid,notnull"`
	DiskInfo     *DiskInfo `bun:"rel:belongs-to,join:disk_info_id=id"`
	MountPoint   string    `bun:"mount_point"`
	Filesystem   string    `bun:"filesystem"`
	Size         uint64    `bun:"size"`
	Used         uint64    `bun:"used"`
	Free         uint64    `bun:"free"`
}

type NetworkInterfaceInfo struct {
	BaseModel
	HardwareInfoID uuid.UUID `bun:"hardware_info_id,type:uuid,notnull"`
	HardwareInfo   *HardwareInfo `bun:"rel:belongs-to,join:hardware_info_id=id"`
	Name           string    `bun:"name"`
	MACAddress     string    `bun:"mac_address"`
	IPAddresses    []string  `bun:"ip_addresses,type:jsonb"`
	Speed          uint64    `bun:"speed"`
	Type           string    `bun:"type"`
	Status         string    `bun:"status"`
}

type SoftwareInventory struct {
	BaseModel
	DeviceID       uuid.UUID `bun:"device_id,type:uuid,notnull"`
	Device         *Device   `bun:"rel:belongs-to,join:device_id=id"`
	Name           string    `bun:"name,notnull"`
	Version        string    `bun:"version"`
	Publisher      string    `bun:"publisher"`
	InstallDate    *time.Time `bun:"install_date"`
	InstallPath    string    `bun:"install_path"`
	UninstallString string   `bun:"uninstall_string"`
	Size           uint64    `bun:"size"`
	IsSystem       bool      `bun:"is_system,notnull,default:false"`
	Architecture   string    `bun:"architecture"`
}

type Metric struct {
	BaseModel
	DeviceID    uuid.UUID `bun:"device_id,type:uuid,notnull"`
	Device      *Device   `bun:"rel:belongs-to,join:device_id=id"`
	Name        string    `bun:"name,notnull"`
	Value       float64   `bun:"value,notnull"`
	Unit        string    `bun:"unit"`
	Tags        map[string]string `bun:"tags,type:jsonb"`
	Timestamp   time.Time `bun:"timestamp,notnull"`
}

type MonitoringPolicy struct {
	BaseModel
	Name           string `bun:"name,notnull"`
	Description    string `bun:"description"`
	IsEnabled      bool   `bun:"is_enabled,notnull,default:true"`
	Severity       string `bun:"severity,notnull,default:'warning'"`
	Condition      string `bun:"condition,notnull"` // JSON condition expression
	Threshold      float64 `bun:"threshold"`
	Duration       time.Duration `bun:"duration"`
	OrganizationID uuid.UUID `bun:"organization_id,type:uuid,notnull"`
	Organization   *Organization `bun:"rel:belongs-to,join:organization_id=id"`
	// Has-many relations removed to avoid bun v1.3 schema creation issues
}

type Alert struct {
	BaseModel
	DeviceID       uuid.UUID `bun:"device_id,type:uuid,notnull"`
	Device         *Device   `bun:"rel:belongs-to,join:device_id=id"`
	PolicyID       *uuid.UUID `bun:"policy_id,type:uuid"`
	Policy         *MonitoringPolicy `bun:"rel:belongs-to,join:policy_id=id"`
	Title          string    `bun:"title,notnull"`
	Message        string    `bun:"message"`
	Severity       string    `bun:"severity,notnull"`
	Status         string    `bun:"status,notnull,default:'open'"`
	Value          float64   `bun:"value"`
	Threshold      float64   `bun:"threshold"`
	AcknowledgedAt *time.Time `bun:"acknowledged_at"`
	AcknowledgedBy *uuid.UUID `bun:"acknowledged_by,type:uuid"`
	AcknowledgedByUser *User `bun:"rel:belongs-to,join:acknowledged_by=id"`
	ResolvedAt     *time.Time `bun:"resolved_at"`
	ResolvedBy     *uuid.UUID `bun:"resolved_by,type:uuid"`
	ResolvedByUser *User `bun:"rel:belongs-to,join:resolved_by=id"`
}

type RefreshToken struct {
	BaseModel
	Token        string    `bun:"token,unique,notnull"`
	UserID       uuid.UUID `bun:"user_id,type:uuid,notnull"`
	User         *User     `bun:"rel:belongs-to,join:user_id=id"`
	ExpiresAt    time.Time `bun:"expires_at,notnull"`
	RevokedAt    *time.Time `bun:"revoked_at"`
	DeviceInfo   string    `bun:"device_info"`
	IPAddress    string    `bun:"ip_address"`
}

type APIKey struct {
	BaseModel
	Name           string    `bun:"name,notnull"`
	KeyHash        string    `bun:"key_hash,unique,notnull"`
	Prefix         string    `bun:"prefix,notnull"`
	UserID         uuid.UUID `bun:"user_id,type:uuid,notnull"`
	User           *User     `bun:"rel:belongs-to,join:user_id=id"`
	OrganizationID uuid.UUID `bun:"organization_id,type:uuid,notnull"`
	Organization   *Organization `bun:"rel:belongs-to,join:organization_id=id"`
	ExpiresAt      *time.Time `bun:"expires_at"`
	LastUsedAt     *time.Time `bun:"last_used_at"`
	Scopes         []string  `bun:"scopes,type:jsonb"`
	IsActive       bool      `bun:"is_active,notnull,default:true"`
}

type RemoteSession struct {
	BaseModel
	SessionID      string    `bun:"session_id,unique,notnull"`
	DeviceID       uuid.UUID `bun:"device_id,type:uuid,notnull"`
	Device         *Device   `bun:"rel:belongs-to,join:device_id=id"`
	UserID         uuid.UUID `bun:"user_id,type:uuid,notnull"`
	User           *User     `bun:"rel:belongs-to,join:user_id=id"`
	Type           string    `bun:"type,notnull"` // desktop, terminal, file_manager
	Status         string    `bun:"status,notnull,default:'active'"`
	Protocol       string    `bun:"protocol"` // webrtc, rdp, vnc, ssh
	StartedAt      time.Time `bun:"started_at,notnull"`
	EndedAt        *time.Time `bun:"ended_at"`
	Duration       int64     `bun:"duration"` // seconds
	RecordingPath  string    `bun:"recording_path"`
	Quality        string    `bun:"quality"`
	Resolution     string    `bun:"resolution"`
	IPAddress      string    `bun:"ip_address"`
	Location       string    `bun:"location"`
	Participants   []SessionParticipant `bun:"-"`
	Events         []SessionEvent       `bun:"-"`
}

type SessionParticipant struct {
	BaseModel
	SessionID   uuid.UUID `bun:"session_id,type:uuid,notnull"`
	Session     *RemoteSession `bun:"rel:belongs-to,join:session_id=id"`
	UserID      uuid.UUID `bun:"user_id,type:uuid,notnull"`
	User        *User     `bun:"rel:belongs-to,join:user_id=id"`
	Role        string    `bun:"role,notnull"` // host, viewer, controller
	JoinedAt    time.Time `bun:"joined_at,notnull"`
	LeftAt      *time.Time `bun:"left_at"`
	Permissions []string  `bun:"permissions,type:jsonb"`
}

type SessionEvent struct {
	BaseModel
	SessionID  uuid.UUID `bun:"session_id,type:uuid,notnull"`
	Session    *RemoteSession `bun:"rel:belongs-to,join:session_id=id"`
	Type       string    `bun:"type,notnull"` // chat, file_transfer, command, permission_change
	UserID     uuid.UUID `bun:"user_id,type:uuid,notnull"`
	User       *User     `bun:"rel:belongs-to,join:user_id=id"`
	Data       map[string]interface{} `bun:"data,type:jsonb"`
	Timestamp  time.Time `bun:"timestamp,notnull"`
}

type Script struct {
	BaseModel
	Name           string `bun:"name,notnull"`
	Description    string `bun:"description"`
	Category       string `bun:"category"`
	Tags           []string `bun:"tags,type:jsonb"`
	Language       string `bun:"language,notnull"` // powershell, bash, shell, python
	Content        string `bun:"content,notnull"`
	Parameters     map[string]interface{} `bun:"parameters,type:jsonb"`
	Timeout        time.Duration `bun:"timeout"`
	RequiresElevation bool   `bun:"requires_elevation,notnull,default:false"`
	IsActive       bool   `bun:"is_active,notnull,default:true"`
	Version        string `bun:"version"`
	OrganizationID uuid.UUID `bun:"organization_id,type:uuid,notnull"`
	Organization   *Organization `bun:"rel:belongs-to,join:organization_id=id"`
	CreatedByID    uuid.UUID `bun:"created_by,type:uuid,notnull"`
	CreatedBy      *User     `bun:"rel:belongs-to,join:created_by=id"`
	// Has-many relations removed to avoid bun v1.3 schema creation issues
}

type ScriptExecution struct {
	BaseModel
	ScriptID       uuid.UUID `bun:"script_id,type:uuid,notnull"`
	Script         *Script   `bun:"rel:belongs-to,join:script_id=id"`
	DeviceID       uuid.UUID `bun:"device_id,type:uuid,notnull"`
	Device         *Device   `bun:"rel:belongs-to,join:device_id=id"`
	UserID         *uuid.UUID `bun:"user_id,type:uuid"`
	User           *User     `bun:"rel:belongs-to,join:user_id=id"`
	TaskID         *uuid.UUID `bun:"task_id,type:uuid"`
	Task           *Task     `bun:"rel:belongs-to,join:task_id=id"`
	Parameters     map[string]interface{} `bun:"parameters,type:jsonb"`
	Status         string    `bun:"status,notnull"` // pending, running, completed, failed, timeout, cancelled
	ExitCode       *int      `bun:"exit_code"`
	Output         string    `bun:"output"`
	Error          string    `bun:"error"`
	StartedAt      time.Time `bun:"started_at,notnull"`
	CompletedAt    *time.Time `bun:"completed_at"`
	Duration       int64     `bun:"duration"` // milliseconds
}

type Schedule struct {
	BaseModel
	Name        string `bun:"name,notnull"`
	Description string `bun:"description"`
	CronExpr    string `bun:"cron_expr,notnull"`
	Timezone    string `bun:"timezone,notnull,default:'UTC'"`
	IsEnabled   bool   `bun:"is_enabled,notnull,default:true"`
	ScriptID    uuid.UUID `bun:"script_id,type:uuid,notnull"`
	Script      *Script   `bun:"rel:belongs-to,join:script_id=id"`
	DeviceIDs   []uuid.UUID `bun:"device_ids,type:uuid[],default:'{}'"`
	GroupIDs    []uuid.UUID `bun:"group_ids,type:uuid[],default:'{}'"`
	Parameters  map[string]interface{} `bun:"parameters,type:jsonb"`
	LastRunAt   *time.Time `bun:"last_run_at"`
	NextRunAt   *time.Time `bun:"next_run_at"`
	RunCount    int64      `bun:"run_count,notnull,default:0"`

	Task *Task `bun:"rel:has-one,join:id=schedule_id"`
}

type Task struct {
	BaseModel
	Name           string `bun:"name,notnull"`
	Description    string `bun:"description"`
	Type           string `bun:"type,notnull"` // script, command, patch, custom
	Status         string `bun:"status,notnull,default:'pending'"` // pending, running, completed, failed, cancelled
	Priority       int    `bun:"priority,notnull,default:0"`
	DeviceID       uuid.UUID `bun:"device_id,type:uuid,notnull"`
	Device         *Device   `bun:"rel:belongs-to,join:device_id=id"`
	UserID         *uuid.UUID `bun:"user_id,type:uuid"`
	User           *User     `bun:"rel:belongs-to,join:user_id=id"`
	ScriptID       *uuid.UUID `bun:"script_id,type:uuid"`
	Script         *Script   `bun:"rel:belongs-to,join:script_id=id"`
	ScheduleID     *uuid.UUID `bun:"schedule_id,type:uuid"`
	Schedule       *Schedule `bun:"rel:belongs-to,join:schedule_id=id"`
	Parameters     map[string]interface{} `bun:"parameters,type:jsonb"`
	Result         map[string]interface{} `bun:"result,type:jsonb"`
	Error          string    `bun:"error"`
	RetryCount     int       `bun:"retry_count,notnull,default:0"`
	MaxRetries     int       `bun:"max_retries,notnull,default:3"`
	ScheduledAt    *time.Time `bun:"scheduled_at"`
	StartedAt      *time.Time `bun:"started_at"`
	CompletedAt    *time.Time `bun:"completed_at"`
	Timeout        time.Duration `bun:"timeout"`
}

type Notification struct {
	BaseModel
	UserID     uuid.UUID `bun:"user_id,type:uuid,notnull"`
	User       *User     `bun:"rel:belongs-to,join:user_id=id"`
	Type       string    `bun:"type,notnull"` // alert, info, warning, success
	Title      string    `bun:"title,notnull"`
	Message    string    `bun:"message"`
	Data       map[string]interface{} `bun:"data,type:jsonb"`
	IsRead     bool      `bun:"is_read,notnull,default:false"`
	ReadAt     *time.Time `bun:"read_at"`
	ActionURL  string    `bun:"action_url"`
	Priority   string    `bun:"priority,notnull,default:'normal'"` // low, normal, high, urgent
}

type AuditLog struct {
	BaseModel
	UserID       *uuid.UUID `bun:"user_id,type:uuid"`
	User         *User      `bun:"rel:belongs-to,join:user_id=id"`
	OrganizationID *uuid.UUID `bun:"organization_id,type:uuid"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id"`
	Action       string    `bun:"action,notnull"`
	Resource     string    `bun:"resource,notnull"`
	ResourceID   string    `bun:"resource_id"`
	OldValues    map[string]interface{} `bun:"old_values,type:jsonb"`
	NewValues    map[string]interface{} `bun:"new_values,type:jsonb"`
	IPAddress    string    `bun:"ip_address"`
	UserAgent    string    `bun:"user_agent"`
	Status       string    `bun:"status,notnull"` // success, failure
	Error        string    `bun:"error"`
}

type License struct {
	BaseModel
	LicenseKey     string    `bun:"license_key,unique,notnull"`
	OrganizationID *uuid.UUID `bun:"organization_id,type:uuid"`
	Organization   *Organization `bun:"rel:belongs-to,join:organization_id=id"`
	Plan           string    `bun:"plan,notnull"`
	MaxDevices     int       `bun:"max_devices,notnull,default:0"`
	MaxUsers       int       `bun:"max_users,notnull,default:0"`
	Features       []string  `bun:"features,type:jsonb"`
	Status         string    `bun:"status,notnull"` // active, expired, trial, suspended
	IssuedAt       time.Time `bun:"issued_at,notnull"`
	ExpiresAt      *time.Time `bun:"expires_at"`
	LastCheckedAt  *time.Time `bun:"last_checked_at"`
	CheckInterval  time.Duration `bun:"check_interval"`
	Metadata       map[string]interface{} `bun:"metadata,type:jsonb"`
}

type Integration struct {
	BaseModel
	Name        string `bun:"name,notnull"`
	Type        string `bun:"type,notnull"` // webhook, telegram, email, slack, teams, ldap, oidc
	Config      map[string]interface{} `bun:"config,type:jsonb"`
	IsEnabled   bool   `bun:"is_enabled,notnull,default:true"`
	OrganizationID uuid.UUID `bun:"organization_id,type:uuid,notnull"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id"`
	CreatedByID uuid.UUID `bun:"created_by,type:uuid,notnull"`
	CreatedBy   *User     `bun:"rel:belongs-to,join:created_by=id"`
	// Has-many relations removed to avoid bun v1.3 schema creation issues
}

type IntegrationTrigger struct {
	BaseModel
	IntegrationID uuid.UUID `bun:"integration_id,type:uuid,notnull"`
	Integration   *Integration `bun:"rel:belongs-to,join:integration_id=id"`
	Event         string    `bun:"event,notnull"`
	Conditions    map[string]interface{} `bun:"conditions,type:jsonb"`
	IsEnabled     bool      `bun:"is_enabled,notnull,default:true"`
}

var Models = []interface{}{
	(*User)(nil),
	(*Organization)(nil),
	(*Site)(nil),
	(*Device)(nil),
	(*DeviceGroup)(nil),
	(*HardwareInfo)(nil),
	(*GPUInfo)(nil),
	(*DiskInfo)(nil),
	(*PartitionInfo)(nil),
	(*NetworkInterfaceInfo)(nil),
	(*SoftwareInventory)(nil),
	(*Metric)(nil),
	(*MonitoringPolicy)(nil),
	(*Alert)(nil),
	(*RefreshToken)(nil),
	(*APIKey)(nil),
	(*RemoteSession)(nil),
	(*SessionParticipant)(nil),
	(*SessionEvent)(nil),
	(*Script)(nil),
	(*ScriptExecution)(nil),
	(*Schedule)(nil),
	(*Task)(nil),
	(*Notification)(nil),
	(*AuditLog)(nil),
	(*License)(nil),
	(*Integration)(nil),
	(*IntegrationTrigger)(nil),
}