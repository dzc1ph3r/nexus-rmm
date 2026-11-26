# Nexus RMM

Enterprise-grade Remote Monitoring and Management (RMM) platform built with Go backend, React/TypeScript frontend, and Go endpoint agents.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            NEXUS RMM ARCHITECTURE                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────────┐     ┌──────────────────┐     ┌──────────────────────┐    │
│  │   Web UI     │◄───►│   API Server     │◄───►│  PostgreSQL Database │    │
│  │  (React/TS)  │     │    (Go/Gin)      │     │     + Redis Cache    │    │
│  └──────────────┘     └────────┬─────────┘     └──────────────────────┘    │
│                                │                                           │
│                                │ mTLS WebSocket + Custom Binary Protocol    │
│                                ▼                                           │
│                       ┌──────────────────┐                                 │
│                       │  Remote Gateway  │                                 │
│                       │  (WebRTC Signal) │                                 │
│                       └────────┬─────────┘                                 │
│                                │                                           │
│                    ┌───────────┼───────────┐                               │
│                    ▼           ▼           ▼                               │
│             ┌──────────┐ ┌──────────┐ ┌──────────┐                         │
│             │ Agent 1  │ │ Agent 2  │ │ Agent N  │                         │
│             │ (Go)     │ │ (Go)     │ │ (Go)     │                         │
│             └──────────┘ └──────────┘ └──────────┘                         │
│                    ▲           ▲           ▲                               │
│                    │           │           │                               │
│              Windows      Linux        macOS                              │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Core Components

| Component | Technology | Description |
|-----------|------------|-------------|
| **API Server** | Go 1.22+, Gin, Bun ORM | REST API, authentication, device management, automation |
| **Remote Gateway** | Go, WebRTC, WebSocket | Signaling server for remote desktop sessions |
| **Endpoint Agent** | Go, mTLS, Protobuf | Cross-platform agent (Windows/Linux/macOS) |
| **Web Console** | React 18, TypeScript, Vite, Tailwind CSS | Admin dashboard with real-time updates |
| **Database** | PostgreSQL 15+ | Primary data store with multi-tenant isolation |
| **Cache/Messaging** | Redis 7+ | Session storage, pub/sub, caching |

## Features

### Core RMM Capabilities
- **Multi-tenant Architecture** - Strict organization isolation with row-level security
- **Device Management** - Inventory, monitoring, grouping, tagging, custom fields
- **Remote Desktop** - WebRTC-based browser access with clipboard, file transfer
- **Remote Shell** - Encrypted command execution with audit logging
- **Script Engine** - PowerShell/Bash/Python with parameter validation, scheduling
- **Task Automation** - Cron-like scheduler with retry logic, chaining, approvals

### Security
- **mTLS Authentication** - Mutual TLS for all agent-server communication
- **Ed25519/X25519 Cryptography** - Enrollment tokens and session encryption
- **JWT with Refresh Rotation** - Short-lived access tokens, secure refresh flow
- **Argon2id Password Hashing** - Configurable cost factor
- **TOTP MFA** - RFC 6238 compliant with backup codes
- **Audit Logging** - Immutable event log with tamper-evident chaining

### Enterprise Features
- **Role-Based Access Control** - Granular permissions per resource type
- **Patch Management** - WSUS/APT/YUM/DNF integration with compliance reporting
- **Software Inventory** - Automated collection with vulnerability correlation
- **Alerting Engine** - Threshold, anomaly, and correlation rules
- **Reporting** - Scheduled PDF/CSV/HTML reports with custom templates
- **License Management** - Offline-capable license validation with feature flags

## Quick Start

### Prerequisites

- **Go** 1.22+
- **Node.js** 20+ (for frontend)
- **PostgreSQL** 15+
- **Redis** 7+
- **Docker** 24+ (for containerized deployment)

### Development Setup

#### Backend

```bash
cd backend
cp config.yaml.example config.yaml  # Edit with your settings
go mod download
go run main.go
```

Server starts on `http://localhost:8080`

#### Frontend

```bash
cd frontend
npm install
npm run dev
```

Frontend runs on `http://localhost:5173` (proxies API to backend)

### Docker Deployment

```bash
# Build and start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

### Production Deployment (Windows RDP)

For production deployment on Windows RDP with domain `rmm.mezfashion.pro`:

1. **Prepare the Server**
   - Windows Server 2022+ with RDP enabled
   - Domain configured: `rmm.mezfashion.pro`
   - SSL certificate (Let's Encrypt or commercial)

2. **Deploy using Docker**
   ```bash
   # On the Windows RDP server
   git clone <repository>
   cd nexus-rmm
   docker-compose -f docker-compose.prod.yml up -d
   ```

3. **Configure Reverse Proxy** (Caddy/NGINX)
   - Terminate TLS at proxy
   - Route `/api/*` to backend:8080
   - Route `/ws/*` to gateway:8443
   - Serve frontend static files

4. **Agent Installation**
   - Windows: Run `install-agent.ps1` as Administrator
   - Linux: `curl -sSL <url>/install-agent.sh | sudo bash`
   - macOS: `curl -sSL <url>/install-agent.sh | sudo bash`

## Configuration

### Environment Variables

Key configuration options (see `backend/config.yaml`):

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  mode: "release"

database:
  host: "postgres"
  port: 5432
  user: "nexus_rmm"
  password: "secure-password"
  name: "nexus_rmm"

auth:
  jwt_secret: "your-256-bit-secret"
  access_token_ttl: 15m
  refresh_token_ttl: 168h
  require_mfa: true
```

### Default Credentials

After first migration, a default admin user is created:

| Field | Value |
|-------|-------|
| Email | `admin@nexus-rmm.local` |
| Password | `admin123` (change immediately!) |

**⚠️ Change the default password on first login!**

## API Documentation

### Authentication

```bash
# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@nexus-rmm.local", "password": "admin123"}'

# Refresh token
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Authorization: Bearer <refresh_token>"
```

### Device Management

```bash
# List devices
curl -X GET http://localhost:8080/api/v1/devices \
  -H "Authorization: Bearer <access_token>"

# Get device details
curl -X GET http://localhost:8080/api/v1/devices/{id} \
  -H "Authorization: Bearer <access_token>"

# Create device (enrollment)
curl -X POST http://localhost:8080/api/v1/devices \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "WORKSTATION-001", "organization_id": "..."}'
```

### Remote Session

```bash
# Create remote session
curl -X POST http://localhost:8080/api/v1/remote/sessions \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"device_id": "...", "protocol": "webrtc"}'

# Join session (WebSocket)
wscat -c "ws://localhost:8080/api/v1/remote/sessions/{id}/join?token=<access_token>"
```

## Project Structure

```
nexus-rmm/
├── backend/                 # Go API Server
│   ├── cmd/                 # Entry points
│   ├── internal/
│   │   ├── api/             # HTTP handlers
│   │   ├── auth/            # Authentication logic
│   │   ├── config/          # Configuration management
│   │   └── storage/         # Database layer (Bun ORM)
│   ├── config.yaml          # Configuration file
│   ├── go.mod
│   └── main.go
│
├── frontend/                # React + TypeScript Web Console
│   ├── public/              # Static assets
│   ├── src/
│   │   ├── components/      # Reusable UI components (shadcn/ui)
│   │   ├── contexts/        # React contexts (Auth, etc.)
│   │   ├── hooks/           # Custom hooks
│   │   ├── lib/             # Utilities (API client, query client)
│   │   ├── pages/           # Page components
│   │   ├── App.tsx          # Main app with routing
│   │   └── main.tsx         # Entry point
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
│
├── agent/                   # Go Endpoint Agent (to be implemented)
│   ├── cmd/
│   ├── internal/
│   └── pkg/
│
├── docker/                  # Docker configurations
│   ├── Dockerfile.backend
│   ├── Dockerfile.frontend
│   ├── Dockerfile.agent
│   └── docker-compose.yml
│
├── docs/                    # Documentation
│   ├── api.md
│   ├── deployment.md
│   ├── development.md
│   └── agent-installation.md
│
└── scripts/                 # Build and deployment scripts
    ├── build-all.sh
    ├── deploy.sh
    └── install-agent.ps1
```

## Development

### Running Tests

```bash
# Backend tests
cd backend
go test ./...

# Frontend tests
cd frontend
npm test
```

### Code Quality

```bash
# Backend linting
cd backend
golangci-lint run

# Frontend linting
cd frontend
npm run lint
```

### Database Migrations

```bash
# Create new migration
cd backend
go run -tags migrate main.go migrate create "migration_name"

# Run migrations
go run main.go migrate up
```

## Agent Protocol

The agent communicates with the server via **mTLS WebSocket** using a custom binary protocol:

```
┌────────────────────────────────────────────────────┐
│                    FRAME FORMAT                     │
├──────────┬──────────┬──────────┬──────────┬────────┤
│  MAGIC   │ VERSION  │  FLAGS   │ PAYLOAD  │  CRC32 │
│  (4B)    │  (1B)    │  (1B)    │  (var)   │  (4B)  │
└──────────┴──────────┴──────────┴──────────┴────────┘
```

### Message Types

| Type | Value | Direction | Description |
|------|-------|-----------|-------------|
| HELLO | 0x01 | Agent → Server | Initial handshake |
| AUTH | 0x02 | Both | Authentication challenge/response |
| HEARTBEAT | 0x03 | Agent → Server | Keep-alive with metrics |
| COMMAND | 0x04 | Server → Agent | Execute command |
| COMMAND_RESULT | 0x05 | Agent → Server | Command output |
| FILE_TRANSFER | 0x06 | Both | File upload/download |
| REMOTE_DESKTOP | 0x07 | Both | WebRTC signaling |
| DISCONNECT | 0xFF | Both | Graceful shutdown |

## License

This project is licensed under the **Apache License 2.0** - see the [LICENSE](LICENSE) file for details.

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Support

- **Documentation**: `/docs` directory
- **Issues**: GitHub Issues
- **Security**: Report to security@nexus-rmm.local

## Roadmap

- [x] Core API Server (Go/Gin)
- [x] Authentication & Authorization
- [x] Device Management API
- [x] Web Console (React/TypeScript)
- [ ] Remote Access Gateway (WebRTC)
- [ ] Endpoint Agent (Go)
- [ ] Script Engine & Automation
- [ ] Patch Management
- [ ] Reporting & Analytics
- [ ] License Management
- [ ] Docker Deployment
- [ ] Agent Installers (MSI, DEB, RPM, PKG)
- [ ] Complete Documentation

---

**Nexus RMM** - Enterprise Remote Monitoring & Management