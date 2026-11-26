package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
	"golang.org/x/crypto/acme/autocert"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Auth     AuthConfig
	Remote   RemoteConfig
	Agent    AgentConfig
	Monitoring MonitoringConfig
	Automation AutomationConfig
	Notification NotificationConfig
	Logging  LoggingConfig
	TLS      TLSConfig
	License  LicenseConfig
	Audit    AuditConfig
}

type ServerConfig struct {
	Host         string
	Port         int
	Mode         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxConnections  int32
	MaxIdleConns    int32
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	Host           string
	Port           int
	Password       string
	DB             int
	PoolSize       int
	MinIdleConns   int
}

type AuthConfig struct {
	JWTSecret        string
	JWTIssuer        string
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
	MFAIssuer        string
	BcryptCost       int
	MaxLoginAttempts int
	LockoutDuration  time.Duration
	PasswordMinLength int
	RequireMFA       bool
}

type RemoteConfig struct {
	GatewayHost           string
	GatewayPort           int
	WebSocketPath         string
	TurnServers           []string
	StunServers           []string
	ICEServers            []string
	MaxSessionDuration    time.Duration
	SessionCleanupInterval time.Duration
	RecordingEnabled      bool
	RecordingPath         string
}

type AgentConfig struct {
	HeartbeatInterval       time.Duration
	MissedHeartbeatsThreshold int
	EnrollmentTokenTTL      time.Duration
	MaxEnrollmentUses       int
	AutoUpdateEnabled       bool
	UpdateCheckInterval     time.Duration
	CommandTimeout          time.Duration
	FileTransferChunkSize   int
	MaxConcurrentTasks      int
}

type MonitoringConfig struct {
	MetricsInterval          time.Duration
	RetentionPeriod          time.Duration
	AlertEvaluationInterval  time.Duration
	NotificationBatchSize    int
	MaxAlertsPerDevice       int
}

type AutomationConfig struct {
	MaxConcurrentJobs int
	JobTimeout        time.Duration
	RetryAttempts     int
	RetryDelay        time.Duration
	SchedulePrecision time.Duration
}

type NotificationConfig struct {
	Email    EmailConfig
	Webhook  WebhookConfig
	Telegram TelegramConfig
}

type EmailConfig struct {
	Enabled      bool
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	FromAddress  string
	FromName     string
	UseTLS       bool
}

type WebhookConfig struct {
	Enabled        bool
	DefaultTimeout time.Duration
	MaxRetries     int
}

type TelegramConfig struct {
	Enabled  bool
	BotToken string
	ChatID   string
}

type LoggingConfig struct {
	Level      string
	Format     string
	Output     string
	FilePath   string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
}

type TLSConfig struct {
	CertFile        string
	KeyFile         string
	CAFile          string
	AutoCert        bool
	AutoCertDomains []string
	AutoCertEmail   string
	AutoCertCache   string
}

type LicenseConfig struct {
	Enabled             bool
	LicenseServerURL    string
	PublicKey           string
	GracePeriod         time.Duration
	CheckInterval       time.Duration
}

type AuditConfig struct {
	Enabled         bool
	RetentionPeriod time.Duration
	BatchSize       int
	FlushInterval   time.Duration
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/nexus-rmm")
	viper.AutomaticEnv()
	viper.SetEnvPrefix("NEXUS")

	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func setDefaults() {
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("server.idle_timeout", "120s")

	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "nexus_rmm")
	viper.SetDefault("database.password", "changeme")
	viper.SetDefault("database.name", "nexus_rmm")
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_connections", 25)
	viper.SetDefault("database.max_idle_connections", 5)
	viper.SetDefault("database.conn_max_lifetime", "300s")

	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.pool_size", 10)
	viper.SetDefault("redis.min_idle_conns", 2)

	viper.SetDefault("auth.jwt_secret", "changeme")
	viper.SetDefault("auth.jwt_issuer", "nexus-rmm")
	viper.SetDefault("auth.access_token_ttl", "15m")
	viper.SetDefault("auth.refresh_token_ttl", "168h")
	viper.SetDefault("auth.mfa_issuer", "Nexus RMM")
	viper.SetDefault("auth.bcrypt_cost", 12)
	viper.SetDefault("auth.max_login_attempts", 5)
	viper.SetDefault("auth.lockout_duration", "15m")
	viper.SetDefault("auth.password_min_length", 12)
	viper.SetDefault("auth.require_mfa", false)

	viper.SetDefault("remote.gateway_host", "0.0.0.0")
	viper.SetDefault("remote.gateway_port", 8443)
	viper.SetDefault("remote.websocket_path", "/ws")
	viper.SetDefault("remote.turn_servers", []string{})
	viper.SetDefault("remote.stun_servers", []string{"stun:stun.l.google.com:19302"})
	viper.SetDefault("remote.ice_servers", []string{})
	viper.SetDefault("remote.max_session_duration", "8h")
	viper.SetDefault("remote.session_cleanup_interval", "1h")
	viper.SetDefault("remote.recording_enabled", false)
	viper.SetDefault("remote.recording_path", "/var/lib/nexus-rmm/recordings")

	viper.SetDefault("agent.heartbeat_interval", "30s")
	viper.SetDefault("agent.missed_heartbeats_threshold", 3)
	viper.SetDefault("agent.enrollment_token_ttl", "24h")
	viper.SetDefault("agent.max_enrollment_uses", 10)
	viper.SetDefault("agent.auto_update_enabled", true)
	viper.SetDefault("agent.update_check_interval", "1h")
	viper.SetDefault("agent.command_timeout", "5m")
	viper.SetDefault("agent.file_transfer_chunk_size", 1048576)
	viper.SetDefault("agent.max_concurrent_tasks", 5)

	viper.SetDefault("monitoring.metrics_interval", "60s")
	viper.SetDefault("monitoring.retention_period", "720h")
	viper.SetDefault("monitoring.alert_evaluation_interval", "30s")
	viper.SetDefault("monitoring.notification_batch_size", 100)
	viper.SetDefault("monitoring.max_alerts_per_device", 1000)

	viper.SetDefault("automation.max_concurrent_jobs", 10)
	viper.SetDefault("automation.job_timeout", "30m")
	viper.SetDefault("automation.retry_attempts", 3)
	viper.SetDefault("automation.retry_delay", "5m")
	viper.SetDefault("automation.schedule_precision", "1m")

	viper.SetDefault("notification.email.enabled", false)
	viper.SetDefault("notification.email.smtp_host", "")
	viper.SetDefault("notification.email.smtp_port", 587)
	viper.SetDefault("notification.email.smtp_user", "")
	viper.SetDefault("notification.email.smtp_password", "")
	viper.SetDefault("notification.email.from_address", "noreply@nexus-rmm.local")
	viper.SetDefault("notification.email.from_name", "Nexus RMM")
	viper.SetDefault("notification.email.use_tls", true)

	viper.SetDefault("notification.webhook.enabled", false)
	viper.SetDefault("notification.webhook.default_timeout", "10s")
	viper.SetDefault("notification.webhook.max_retries", 3)

	viper.SetDefault("notification.telegram.enabled", false)
	viper.SetDefault("notification.telegram.bot_token", "")
	viper.SetDefault("notification.telegram.chat_id", "")

	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output", "stdout")
	viper.SetDefault("logging.file_path", "/var/log/nexus-rmm/server.log")
	viper.SetDefault("logging.max_size", 100)
	viper.SetDefault("logging.max_backups", 7)
	viper.SetDefault("logging.max_age", 30)
	viper.SetDefault("logging.compress", true)

	viper.SetDefault("tls.cert_file", "")
	viper.SetDefault("tls.key_file", "")
	viper.SetDefault("tls.ca_file", "")
	viper.SetDefault("tls.auto_cert", false)
	viper.SetDefault("tls.auto_cert_domains", []string{})
	viper.SetDefault("tls.auto_cert_email", "")
	viper.SetDefault("tls.auto_cert_cache", "/var/lib/nexus-rmm/certs")

	viper.SetDefault("license.enabled", true)
	viper.SetDefault("license.license_server_url", "")
	viper.SetDefault("license.public_key", "")
	viper.SetDefault("license.grace_period", "168h")
	viper.SetDefault("license.check_interval", "24h")

	viper.SetDefault("audit.enabled", true)
	viper.SetDefault("audit.retention_period", "8760h")
	viper.SetDefault("audit.batch_size", 100)
	viper.SetDefault("audit.flush_interval", "5s")
}

func (c *Config) ServerAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

func (c *Config) DatabaseDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}

func (c *Config) TLSConfig() (*tls.Config, error) {
	if c.TLS.AutoCert && len(c.TLS.AutoCertDomains) > 0 {
		m := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(c.TLS.AutoCertDomains...),
			Cache:      autocert.DirCache(c.TLS.AutoCertCache),
			Email:      c.TLS.AutoCertEmail,
		}
		return &tls.Config{
			GetCertificate: m.GetCertificate,
			NextProtos:     []string{"h2", "http/1.1"},
			MinVersion:     tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_AES_256_GCM_SHA384,
				tls.TLS_CHACHA20_POLY1305_SHA256,
				tls.TLS_AES_128_GCM_SHA256,
			},
			CurvePreferences: []tls.CurveID{
				tls.X25519,
				tls.CurveP256,
			},
		}, nil
	}

	if c.TLS.CertFile != "" && c.TLS.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(c.TLS.CertFile, c.TLS.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS cert: %w", err)
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_AES_256_GCM_SHA384,
				tls.TLS_CHACHA20_POLY1305_SHA256,
				tls.TLS_AES_128_GCM_SHA256,
			},
			CurvePreferences: []tls.CurveID{
				tls.X25519,
				tls.CurveP256,
			},
			NextProtos: []string{"h2", "http/1.1"},
		}

		if c.TLS.CAFile != "" {
			caCert, err := os.ReadFile(c.TLS.CAFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read CA file: %w", err)
			}
			caPool := x509.NewCertPool()
			caPool.AppendCertsFromPEM(caCert)
			tlsConfig.ClientCAs = caPool
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		}

		return tlsConfig, nil
	}

	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_AES_128_GCM_SHA256,
		},
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
		},
		NextProtos: []string{"h2", "http/1.1"},
	}, nil
}

func GetConfigPath() string {
	if path := os.Getenv("NEXUS_CONFIG"); path != "" {
		return path
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nexus-rmm", "config.yaml")
}