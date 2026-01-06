---
Title: Go-Based startdev Replacement Architecture
Ticket: MO-005-IMPROVE-STARTDEV
Status: active
Topics:
    - dev-tooling
    - scripts
    - bash
    - dev-environment
DocType: design-doc
Intent: long-term
Owners:
    - team
RelatedFiles: []
ExternalSources: []
Summary: "Architecture design for a Go-based replacement of startdev.sh with proper separation of concerns: configuration, building, environment preparation, validation, launching/monitoring, log exposure, and command management."
LastUpdated: 2026-01-06T13:04:45.123456789-05:00
WhatFor: "Designing a maintainable, testable Go program to replace the bash script with better error handling, process management, and extensibility."
WhenToUse: "When implementing the Go replacement for startdev.sh or reviewing architectural decisions."
---

# Go-Based startdev Replacement Architecture

## Executive Summary

This document designs a Go-based replacement for `moments/scripts/startdev.sh` that maintains all existing functionality while providing better structure, testability, and extensibility. The architecture separates concerns into seven distinct domains: **Configuration**, **Building**, **Environment Preparation**, **Validation**, **Launching & Monitoring**, **Log Exposure**, and **Command Management**. Each domain has its own interfaces, implementations, and affordances, enabling independent testing, evolution, and replacement.

The design leverages Go's strengths (structured error handling, concurrency, cross-platform support) while maintaining compatibility with existing tooling (moments-config CLI, Makefile targets, pnpm). The program will be structured as a Cobra CLI application with subcommands for different operations, supporting both interactive and programmatic usage.

## Problem Statement

The current `startdev.sh` bash script, while functional, has several limitations:

1. **Monolithic structure**: All logic intertwined, difficult to test individual components
2. **Limited error handling**: Bash error handling is fragile, difficult to provide actionable errors
3. **Process management**: Background process management is basic, no graceful shutdown
4. **No programmatic API**: Cannot be used as a library or integrated into other tools
5. **Platform dependencies**: Relies on platform-specific tools (`lsof`, shell features)
6. **Limited observability**: Basic logging, no structured output or metrics
7. **Hard to extend**: Adding new commands or features requires modifying the script

A Go-based replacement addresses these issues while maintaining feature parity and improving developer experience.

## Architecture Overview

### High-Level Structure

```
moments-dev (Go CLI)
├── cmd/moments-dev/
│   └── main.go (Cobra root command)
├── pkg/
│   ├── config/          # Configuration management
│   ├── builder/         # Build orchestration
│   ├── env/             # Environment preparation
│   ├── validator/       # Setup validation
│   ├── launcher/        # Service launching & monitoring
│   ├── logs/            # Log aggregation & exposure
│   └── commands/        # Command implementations (start, restart, rebuild, etc.)
└── internal/
    ├── process/         # Process management utilities
    └── ports/           # Port management utilities
```

### Core Principles

1. **Separation of Concerns**: Each domain is independent with clear interfaces
2. **Dependency Injection**: Components receive dependencies, enabling testing
3. **Error Handling**: Structured errors with context, no silent failures
4. **Observability**: Structured logging, metrics, and status reporting
5. **Extensibility**: Plugin-like architecture for adding new commands
6. **Backward Compatibility**: Maintains compatibility with existing tooling

## Component Design

### 1. Configuration (`pkg/config/`)

**Purpose**: Load, resolve, and provide access to configuration values from multiple sources (YAML files, environment variables, CLI flags).

**Key Interfaces**:

```go
// ConfigProvider loads and provides configuration values
type ConfigProvider interface {
    // Get retrieves a configuration value by key (dot-notation)
    Get(ctx context.Context, key string) (string, error)
    
    // GetWithDefault retrieves a value or returns default
    GetWithDefault(ctx context.Context, key, defaultValue string) string
    
    // GetAll returns all configuration as a map
    GetAll(ctx context.Context) (map[string]string, error)
    
    // GetViteEnv returns VITE_* prefixed environment variables for frontend
    GetViteEnv(ctx context.Context) (map[string]string, error)
}

// ConfigLoader loads configuration from various sources
type ConfigLoader interface {
    // Load loads configuration from sources (YAML, env vars, etc.)
    Load(ctx context.Context) (*Config, error)
    
    // Reload reloads configuration (useful for hot-reload)
    Reload(ctx context.Context) (*Config, error)
}

// Config represents the complete configuration
type Config struct {
    RepoRoot           string
    BackendPort        int
    FrontendPort       int
    BackendURL         string
    IdentityURL        string
    StytchPublicToken  string
    StytchHostedURL    string
    DatabaseURL        string
    KeysDir            string
    // ... other config values
}
```

**Implementations**:

1. **AppConfigProvider**: Uses `moments-config` CLI (maintains compatibility)
   ```go
   type AppConfigProvider struct {
       configCLIPath string
       repoRoot      string
       cache         *sync.Map // Cache for Get calls
   }
   ```

2. **YAMLConfigProvider**: Direct YAML parsing (future optimization)
   ```go
   type YAMLConfigProvider struct {
       configPaths []string
       envPrefix   string
   }
   ```

3. **EnvConfigProvider**: Environment variable overrides
   ```go
   type EnvConfigProvider struct {
       prefix string
   }
   ```

**Features**:
- Caching of configuration values (avoid repeated CLI calls)
- Layered configuration (base.yaml → local.yaml → env vars → CLI flags)
- Validation of required configuration values
- Type-safe accessors (GetInt, GetBool, etc.)

**Affordances**:
- `config.Load()` - Load configuration from all sources
- `config.Get(key)` - Get single value
- `config.GetViteEnv()` - Get frontend environment variables
- `config.Validate()` - Validate required values are present

### 2. Building (`pkg/builder/`)

**Purpose**: Orchestrate building of backend binaries and frontend dependencies.

**Key Interfaces**:

```go
// Builder builds backend binaries
type Builder interface {
    // Build builds the moments-config CLI
    BuildConfigCLI(ctx context.Context) error
    
    // BuildServer builds the moments-server binary
    BuildServer(ctx context.Context) error
    
    // BuildAll builds all required binaries
    BuildAll(ctx context.Context) error
    
    // NeedsBuild checks if binaries need rebuilding
    NeedsBuild(ctx context.Context, target string) (bool, error)
}

// FrontendBuilder builds/manages frontend dependencies
type FrontendBuilder interface {
    // InstallDependencies installs npm dependencies
    InstallDependencies(ctx context.Context) error
    
    // GenerateVersion generates version metadata
    GenerateVersion(ctx context.Context) error
    
    // Prepare prepares frontend for dev server
    Prepare(ctx context.Context) error
}
```

**Implementations**:

1. **MakefileBuilder**: Uses Makefile targets (maintains compatibility)
   ```go
   type MakefileBuilder struct {
       repoRoot string
       makeCmd  string
   }
   ```

2. **GoBuilder**: Direct `go build` commands (future optimization)
   ```go
   type GoBuilder struct {
       goCmd    string
       workDir  string
       ldflags  string
   }
   ```

3. **PnpmBuilder**: Uses pnpm for frontend
   ```go
   type PnpmBuilder struct {
       pnpmCmd  string
       workDir  string
   }
   ```

**Features**:
- Parallel building when possible
- Build caching (skip if already built)
- Build progress reporting
- Build artifact verification

**Affordances**:
- `builder.BuildConfigCLI()` - Build moments-config
- `builder.BuildServer()` - Build moments-server
- `builder.BuildAll()` - Build everything
- `builder.NeedsBuild()` - Check if rebuild needed

### 3. Environment Preparation (`pkg/env/`)

**Purpose**: Prepare the development environment (database, keys, migrations, dependencies).

**Key Interfaces**:

```go
// EnvironmentPreparer prepares the dev environment
type EnvironmentPreparer interface {
    // Prepare prepares the entire environment
    Prepare(ctx context.Context) error
    
    // PrepareDatabase ensures database exists and migrations are run
    PrepareDatabase(ctx context.Context) error
    
    // PrepareKeys ensures JWT keys exist
    PrepareKeys(ctx context.Context) error
    
    // PrepareDependencies ensures all dependencies are installed
    PrepareDependencies(ctx context.Context) error
}

// BootstrapRunner runs the bootstrap script
type BootstrapRunner interface {
    // Run runs the bootstrap script
    Run(ctx context.Context) error
    
    // Check checks if bootstrap is needed
    Check(ctx context.Context) (bool, error)
}
```

**Implementations**:

1. **BootstrapPreparer**: Uses existing bootstrap-startup.sh (maintains compatibility)
   ```go
   type BootstrapPreparer struct {
       scriptPath string
       config     *config.Config
   }
   ```

2. **NativePreparer**: Native Go implementation (future optimization)
   ```go
   type NativePreparer struct {
       dbClient   database.Client
       keyManager key.Manager
       migrator   migration.Runner
   }
   ```

**Features**:
- Idempotent operations (safe to run multiple times)
- Progress reporting for long operations
- Rollback support for failed operations
- Dependency ordering (database before migrations, etc.)

**Affordances**:
- `env.Prepare()` - Prepare entire environment
- `env.PrepareDatabase()` - Just database setup
- `env.PrepareKeys()` - Just key generation
- `env.Check()` - Check if preparation needed

### 4. Validation (`pkg/validator/`)

**Purpose**: Validate that the development environment is correctly set up and ready to run.

**Key Interfaces**:

```go
// Validator validates the setup
type Validator interface {
    // Validate performs all validation checks
    Validate(ctx context.Context) (*ValidationResult, error)
    
    // ValidatePort checks if a port is available
    ValidatePort(ctx context.Context, port int) error
    
    // ValidateDatabase checks database connectivity
    ValidateDatabase(ctx context.Context) error
    
    // ValidateKeys checks if keys exist and are valid
    ValidateKeys(ctx context.Context) error
    
    // ValidateBinaries checks if required binaries exist
    ValidateBinaries(ctx context.Context) error
}

// ValidationResult contains validation results
type ValidationResult struct {
    Valid   bool
    Errors  []ValidationError
    Warnings []ValidationWarning
}

// ValidationError represents a validation failure
type ValidationError struct {
    Component string
    Message   string
    Fix       string // Suggested fix
}
```

**Implementations**:

1. **PortValidator**: Checks port availability
   ```go
   type PortValidator struct {
       portChecker ports.Checker
   }
   ```

2. **DatabaseValidator**: Checks database connectivity
   ```go
   type DatabaseValidator struct {
       dbClient database.Client
   }
   ```

3. **HealthValidator**: Checks service health endpoints
   ```go
   type HealthValidator struct {
       backendURL  string
       frontendURL string
       httpClient  *http.Client
   }
   ```

**Features**:
- Comprehensive validation before startup
- Actionable error messages with fix suggestions
- Health endpoint checking (not just port listening)
- Validation caching (avoid repeated checks)

**Affordances**:
- `validator.Validate()` - Run all validations
- `validator.ValidatePort()` - Check specific port
- `validator.ValidateHealth()` - Check service health
- `validator.GetResult()` - Get detailed results

### 5. Launching & Monitoring (`pkg/launcher/`)

**Purpose**: Launch services, monitor their health, and manage their lifecycle.

**Key Interfaces**:

```go
// Service represents a service to be launched
type Service struct {
    Name        string
    Command     []string
    WorkDir     string
    Env         map[string]string
    Port        int
    HealthCheck HealthCheck
    LogFile     string
}

// HealthCheck defines how to check service health
type HealthCheck struct {
    Type        string // "port", "http", "tcp"
    Endpoint    string // For HTTP health checks
    Timeout     time.Duration
    Interval    time.Duration
}

// Launcher launches and manages services
type Launcher interface {
    // Start starts a service
    Start(ctx context.Context, service *Service) (*Process, error)
    
    // Stop stops a service gracefully
    Stop(ctx context.Context, process *Process) error
    
    // Restart restarts a service
    Restart(ctx context.Context, process *Process) error
    
    // Status gets the status of a service
    Status(ctx context.Context, process *Process) (*ServiceStatus, error)
    
    // Wait waits for a service to be ready
    WaitForReady(ctx context.Context, process *Process) error
}

// Process represents a running service
type Process struct {
    ID          string
    Service     *Service
    Cmd         *exec.Cmd
    StartedAt   time.Time
    Status      ProcessStatus
    Health      HealthStatus
}

// ServiceStatus represents current service status
type ServiceStatus struct {
    Running   bool
    Healthy   bool
    Uptime    time.Duration
    Restarts  int
    LastError error
}
```

**Implementations**:

1. **ProcessLauncher**: Uses os/exec for process management
   ```go
   type ProcessLauncher struct {
       processes map[string]*Process
       mu        sync.RWMutex
       logger    log.Logger
   }
   ```

2. **DockerLauncher**: Docker-based launching (future option)
   ```go
   type DockerLauncher struct {
       client *docker.Client
   }
   ```

**Features**:
- Graceful shutdown (SIGTERM → wait → SIGKILL)
- Health monitoring with configurable checks
- Automatic restart on failure (optional)
- Process state management
- Resource limits (CPU, memory)

**Affordances**:
- `launcher.Start()` - Start a service
- `launcher.Stop()` - Stop a service
- `launcher.Restart()` - Restart a service
- `launcher.Status()` - Get service status
- `launcher.WaitForReady()` - Wait for service to be healthy

### 6. Log Exposure (`pkg/logs/`)

**Purpose**: Aggregate, filter, and expose logs from multiple services in a unified interface.

**Key Interfaces**:

```go
// LogAggregator aggregates logs from multiple sources
type LogAggregator interface {
    // AddSource adds a log source
    AddSource(ctx context.Context, source LogSource) error
    
    // RemoveSource removes a log source
    RemoveSource(ctx context.Context, sourceID string) error
    
    // Stream streams logs matching the filter
    Stream(ctx context.Context, filter LogFilter) (<-chan LogEntry, error)
    
    // Tail returns the last N log entries
    Tail(ctx context.Context, filter LogFilter, n int) ([]LogEntry, error)
    
    // Search searches logs matching the query
    Search(ctx context.Context, query string, filter LogFilter) ([]LogEntry, error)
}

// LogSource represents a source of logs
type LogSource struct {
    ID       string
    Name     string
    Type     string // "file", "stdout", "stderr"
    Path     string // For file sources
    Process  *Process // For process sources
}

// LogEntry represents a single log entry
type LogEntry struct {
    Timestamp time.Time
    Source    string
    Level     string
    Message   string
    Fields    map[string]interface{}
}

// LogFilter filters log entries
type LogFilter struct {
    Sources   []string
    Levels    []string
    Since     time.Time
    Until     time.Time
    Pattern   string // Regex pattern
}
```

**Implementations**:

1. **FileLogAggregator**: Aggregates from log files
   ```go
   type FileLogAggregator struct {
       sources map[string]*fileSource
       mu      sync.RWMutex
   }
   ```

2. **StreamLogAggregator**: Streams from process stdout/stderr
   ```go
   type StreamLogAggregator struct {
       sources map[string]*streamSource
       mu      sync.RWMutex
   }
   ```

**Features**:
- Real-time log streaming
- Log filtering and search
- Structured log parsing (JSON, key-value)
- Log rotation support
- Multiple output formats (text, JSON, colored)

**Affordances**:
- `logs.AddSource()` - Add a log source
- `logs.Stream()` - Stream logs in real-time
- `logs.Tail()` - Get recent logs
- `logs.Search()` - Search logs
- `logs.Follow()` - Follow logs (like `tail -f`)

### 7. Command Management (`pkg/commands/`)

**Purpose**: Implement high-level commands (start, stop, restart, rebuild, status, logs) that compose the lower-level components.

**Key Interfaces**:

```go
// Command represents a high-level command
type Command interface {
    // Name returns the command name
    Name() string
    
    // Description returns the command description
    Description() string
    
    // Run executes the command
    Run(ctx context.Context, args []string) error
    
    // Flags returns command-specific flags
    Flags() []Flag
}

// CommandRunner runs commands
type CommandRunner interface {
    // Register registers a command
    Register(cmd Command) error
    
    // Run runs a command by name
    Run(ctx context.Context, name string, args []string) error
    
    // List lists all registered commands
    List() []Command
}
```

**Command Implementations**:

1. **StartCommand**: Starts all services
   ```go
   type StartCommand struct {
       config     config.ConfigProvider
       builder    builder.Builder
       preparer   env.EnvironmentPreparer
       validator  validator.Validator
       launcher   launcher.Launcher
       logAgg     logs.LogAggregator
   }
   ```

2. **StopCommand**: Stops all services
   ```go
   type StopCommand struct {
       launcher launcher.Launcher
   }
   ```

3. **RestartCommand**: Restarts services
   ```go
   type RestartCommand struct {
       launcher launcher.Launcher
   }
   ```

4. **RebuildCommand**: Rebuilds binaries
   ```go
   type RebuildCommand struct {
       builder builder.Builder
   }
   ```

5. **StatusCommand**: Shows service status
   ```go
   type StatusCommand struct {
       launcher launcher.Launcher
   }
   ```

6. **LogsCommand**: Shows logs
   ```go
   type LogsCommand struct {
       logAgg logs.LogAggregator
   }
   ```

**Features**:
- Composable commands (start = build + prepare + validate + launch)
- Command chaining (rebuild && restart)
- Dry-run mode
- Command history/audit log

**Affordances**:
- `moments-dev start` - Start all services
- `moments-dev stop` - Stop all services
- `moments-dev restart` - Restart services
- `moments-dev rebuild` - Rebuild binaries
- `moments-dev status` - Show status
- `moments-dev logs` - Show logs
- `moments-dev logs --follow` - Follow logs
- `moments-dev logs --search "error"` - Search logs

## Data Flow

### Start Command Flow

```
User: moments-dev start
  ↓
1. Load Configuration (config.Load)
   ├─ Read YAML files
   ├─ Read environment variables
   └─ Apply CLI flags
  ↓
2. Build Binaries (builder.BuildAll)
   ├─ Build moments-config CLI
   └─ Check if moments-server needs build
  ↓
3. Prepare Environment (env.Prepare)
   ├─ Run bootstrap script
   ├─ Ensure database exists
   ├─ Run migrations
   └─ Generate keys if needed
  ↓
4. Validate Setup (validator.Validate)
   ├─ Check ports available
   ├─ Check database connectivity
   ├─ Check keys exist
   └─ Check binaries exist
  ↓
5. Launch Services (launcher.Start)
   ├─ Start backend service
   │  ├─ Set environment variables
   │  ├─ Start process
   │  └─ Wait for health check
   └─ Start frontend service
      ├─ Set VITE_* env vars
      ├─ Start process
      └─ Wait for health check
  ↓
6. Setup Log Aggregation (logs.AddSource)
   ├─ Add backend log source
   └─ Add frontend log source
  ↓
7. Display Status
   ├─ Show service statuses
   ├─ Show log file locations
   └─ Show quick commands
```

### Restart Command Flow

```
User: moments-dev restart backend
  ↓
1. Find Process (launcher.FindProcess)
   └─ Lookup by service name
  ↓
2. Stop Service (launcher.Stop)
   ├─ Send SIGTERM
   ├─ Wait for graceful shutdown
   └─ Send SIGKILL if needed
  ↓
3. Start Service (launcher.Start)
   ├─ Reuse existing Service config
   ├─ Start process
   └─ Wait for health check
```

## Implementation Plan

### Phase 1: Core Infrastructure (Week 1-2)

1. **Setup project structure**
   - Create `cmd/moments-dev/` with Cobra root
   - Create `pkg/` directories for each domain
   - Setup basic logging and error handling

2. **Implement Configuration (`pkg/config/`)**
   - AppConfigProvider using moments-config CLI
   - Config struct with all fields
   - Caching layer
   - Tests

3. **Implement Building (`pkg/builder/`)**
   - MakefileBuilder
   - PnpmBuilder
   - Build verification
   - Tests

### Phase 2: Environment & Validation (Week 3)

4. **Implement Environment Preparation (`pkg/env/`)**
   - BootstrapPreparer wrapping bootstrap-startup.sh
   - Progress reporting
   - Tests

5. **Implement Validation (`pkg/validator/`)**
   - PortValidator (cross-platform port checking)
   - DatabaseValidator
   - BinaryValidator
   - HealthValidator (HTTP health checks)
   - Tests

### Phase 3: Launching & Monitoring (Week 4)

6. **Implement Launching (`pkg/launcher/`)**
   - ProcessLauncher with os/exec
   - Process state management
   - Health monitoring
   - Graceful shutdown
   - Tests

7. **Implement Log Aggregation (`pkg/logs/`)**
   - FileLogAggregator
   - StreamLogAggregator
   - Log filtering and search
   - Tests

### Phase 4: Commands (Week 5)

8. **Implement Commands (`pkg/commands/`)**
   - StartCommand
   - StopCommand
   - RestartCommand
   - RebuildCommand
   - StatusCommand
   - LogsCommand
   - Tests

9. **CLI Integration**
   - Wire up commands to Cobra
   - Add flags and help text
   - Error handling and user feedback

### Phase 5: Polish & Testing (Week 6)

10. **Testing & Documentation**
    - Integration tests
    - End-to-end tests
    - User documentation
    - Migration guide from bash script

11. **Performance & Reliability**
    - Optimize configuration caching
    - Add retry logic
    - Improve error messages
    - Add metrics/observability

## Key Design Decisions

### Decision 1: Maintain Compatibility with Existing Tooling

**Rationale**: The bash script works and is relied upon. A complete rewrite would be risky. By maintaining compatibility with moments-config CLI, Makefile targets, and bootstrap-startup.sh, we can migrate incrementally.

**Trade-offs**:
- ✅ Lower risk, incremental migration
- ✅ Can reuse existing scripts during transition
- ❌ Some inefficiency (CLI calls vs direct YAML parsing)
- ❌ Still depends on external tools

**Future**: Can optimize by implementing native YAML parsing and direct database/key management.

### Decision 2: Use Interfaces for All Components

**Rationale**: Enables testing, allows swapping implementations, and provides clear contracts.

**Trade-offs**:
- ✅ Highly testable
- ✅ Flexible, can swap implementations
- ✅ Clear contracts
- ❌ More code (interfaces + implementations)
- ❌ Slight performance overhead

### Decision 3: Structured Logging Throughout

**Rationale**: Better observability, easier debugging, enables log aggregation tools.

**Trade-offs**:
- ✅ Better debugging experience
- ✅ Can integrate with log tools
- ✅ Structured data for analysis
- ❌ More verbose than simple prints

### Decision 4: Health Checks Instead of Just Port Checks

**Rationale**: Port listening doesn't mean service is ready. HTTP health checks provide actual readiness.

**Trade-offs**:
- ✅ More accurate readiness detection
- ✅ Catches initialization errors
- ❌ Requires services to have health endpoints
- ❌ Slightly slower startup detection

### Decision 5: Process Management Instead of Background Jobs

**Rationale**: Better control, graceful shutdown, monitoring capabilities.

**Trade-offs**:
- ✅ Graceful shutdown
- ✅ Better monitoring
- ✅ Can restart on failure
- ❌ More complex than background jobs
- ❌ Requires process state management

## Alternatives Considered

### Alternative 1: Keep Bash Script, Add Go Helper Libraries

**Approach**: Keep startdev.sh but create Go libraries for complex operations (port checking, process management).

**Pros**:
- Minimal changes to existing script
- Can reuse Go libraries elsewhere
- Lower risk

**Cons**:
- Still have bash script complexity
- Mix of bash and Go is awkward
- Harder to test end-to-end

**Decision**: Rejected - doesn't solve core problems (monolithic structure, limited error handling).

### Alternative 2: Use Existing Tools (Docker Compose, systemd, etc.)

**Approach**: Use Docker Compose or systemd to manage services instead of custom tooling.

**Pros**:
- Leverage existing, battle-tested tools
- Less code to maintain
- Industry standard

**Cons**:
- Docker adds overhead for dev environment
- systemd is Linux-specific
- Less control over startup flow
- Harder to integrate with existing tooling

**Decision**: Rejected - adds complexity and doesn't integrate well with existing Makefile/bootstrap workflow.

### Alternative 3: Python Instead of Go

**Approach**: Write replacement in Python.

**Pros**:
- Rich ecosystem for process management
- Easier scripting for some operations
- Good cross-platform support

**Cons**:
- Requires Python runtime (not always available)
- Slower than Go
- Less type safety
- Doesn't match codebase language (Go)

**Decision**: Rejected - Go matches codebase, better performance, single binary distribution.

## Plugin Protocol Design

### Overview

To enable extensibility and allow colleagues to easily extend the startup script, each phase supports a plugin protocol based on stdio communication. Plugins are external scripts or programs that communicate with the main tool via JSON-RPC-like protocol over stdin/stdout. This design allows:

- **Language-agnostic plugins**: Plugins can be written in any language (bash, Python, Go, etc.)
- **Easy extension**: Colleagues can add custom logic without modifying core code
- **Generic tooling**: The main tool becomes a generic orchestrator that can be extended for different projects
- **Isolation**: Plugin failures don't crash the main tool
- **Composability**: Multiple plugins can extend the same phase

### Protocol Specification

#### Communication Format

Plugins communicate using JSON messages over stdin/stdout:

```json
{
  "jsonrpc": "2.0",
  "method": "phase_name",
  "params": {
    "config": {...},
    "context": {...}
  },
  "id": 1
}
```

**Response Format**:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "success": true,
    "data": {...},
    "messages": ["info", "warn", "error"]
  },
  "id": 1
}
```

**Error Format**:
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32000,
    "message": "Plugin error message",
    "data": {...}
  },
  "id": 1
}
```

#### Phase-Specific Protocols

Each phase defines its own protocol with specific request/response structures.

### 1. Configuration Phase Plugin Protocol

**Purpose**: Allow plugins to modify or extend configuration resolution.

**Request**:
```json
{
  "jsonrpc": "2.0",
  "method": "configure",
  "params": {
    "phase": "config",
    "config": {
      "repo_root": "/path/to/repo",
      "backend_port": 8083,
      "frontend_port": 5173,
      "backend_url": "http://localhost:8083",
      "identity_url": "http://localhost:8083"
    },
    "raw_config": {
      "platform": {...},
      "integrations": {...}
    }
  },
  "id": 1
}
```

**Response**:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "success": true,
    "config": {
      "backend_port": 8084,  // Modified value
      "custom_field": "value"  // Added field
    },
    "vite_env": {
      "VITE_CUSTOM_VAR": "value"  // Additional VITE vars
    },
    "messages": [
      {"level": "info", "message": "Configuration modified"}
    ]
  },
  "id": 1
}
```

**Plugin Interface**:
```go
type ConfigPlugin interface {
    Configure(ctx context.Context, req ConfigRequest) (*ConfigResponse, error)
}

type ConfigRequest struct {
    Config     map[string]interface{}
    RawConfig  map[string]interface{}
    RepoRoot   string
}

type ConfigResponse struct {
    Config     map[string]interface{}
    ViteEnv    map[string]string
    Messages   []PluginMessage
}
```

**Example Plugin** (bash):
```bash
#!/bin/bash
# config-plugin.sh - Adds custom VITE variable

while read -r line; do
    request=$(echo "$line" | jq -r '.')
    
    # Extract config
    config=$(echo "$request" | jq -r '.params.config')
    backend_port=$(echo "$config" | jq -r '.backend_port')
    
    # Modify config
    new_config=$(echo "$config" | jq ".backend_port = $((backend_port + 1))")
    
    # Add VITE variable
    vite_env=$(echo '{}' | jq ".VITE_CUSTOM_PORT = $backend_port")
    
    # Build response
    response=$(jq -n \
        --argjson config "$new_config" \
        --argjson vite_env "$vite_env" \
        '{
            "jsonrpc": "2.0",
            "result": {
                "success": true,
                "config": $config,
                "vite_env": $vite_env,
                "messages": [{"level": "info", "message": "Port incremented"}]
            },
            "id": 1
        }')
    
    echo "$response"
done
```

### 2. Building Phase Plugin Protocol

**Purpose**: Allow plugins to customize build process or add build steps.

**Request**:
```json
{
  "jsonrpc": "2.0",
  "method": "build",
  "params": {
    "phase": "build",
    "targets": ["config_cli", "server", "frontend"],
    "config": {...},
    "force": false
  },
  "id": 1
}
```

**Response**:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "success": true,
    "built": ["config_cli", "server"],
    "skipped": ["frontend"],
    "artifacts": {
      "config_cli": "/path/to/moments-config",
      "server": "/path/to/moments-server"
    },
    "messages": [
      {"level": "info", "message": "Built config CLI"}
    ]
  },
  "id": 1
}
```

**Plugin Interface**:
```go
type BuildPlugin interface {
    Build(ctx context.Context, req BuildRequest) (*BuildResponse, error)
}

type BuildRequest struct {
    Targets []string
    Config  map[string]interface{}
    Force   bool
}

type BuildResponse struct {
    Built     []string
    Skipped   []string
    Artifacts map[string]string
    Messages  []PluginMessage
}
```

### 3. Environment Preparation Phase Plugin Protocol

**Purpose**: Allow plugins to add custom environment setup steps.

**Request**:
```json
{
  "jsonrpc": "2.0",
  "method": "prepare",
  "params": {
    "phase": "env",
    "config": {
      "database_url": "...",
      "keys_dir": "..."
    },
    "steps": ["database", "keys", "migrations"]
  },
  "id": 1
}
```

**Response**:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "success": true,
    "prepared": ["database", "keys", "migrations", "custom_setup"],
    "skipped": [],
    "messages": [
      {"level": "info", "message": "Custom setup completed"}
    ]
  },
  "id": 1
}
```

**Plugin Interface**:
```go
type EnvPlugin interface {
    Prepare(ctx context.Context, req EnvRequest) (*EnvResponse, error)
}

type EnvRequest struct {
    Config map[string]interface{}
    Steps  []string
}

type EnvResponse struct {
    Prepared []string
    Skipped  []string
    Messages []PluginMessage
}
```

### 4. Validation Phase Plugin Protocol

**Purpose**: Allow plugins to add custom validation checks.

**Request**:
```json
{
  "jsonrpc": "2.0",
  "method": "validate",
  "params": {
    "phase": "validate",
    "config": {...},
    "checks": ["ports", "database", "keys"]
  },
  "id": 1
}
```

**Response**:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "success": true,
    "valid": true,
    "checks": {
      "ports": {"valid": true},
      "database": {"valid": true},
      "custom_check": {"valid": false, "error": "Custom check failed"}
    },
    "errors": [
      {"component": "custom_check", "message": "Custom check failed", "fix": "Run setup-custom"}
    ],
    "warnings": []
  },
  "id": 1
}
```

**Plugin Interface**:
```go
type ValidationPlugin interface {
    Validate(ctx context.Context, req ValidationRequest) (*ValidationResponse, error)
}

type ValidationRequest struct {
    Config map[string]interface{}
    Checks []string
}

type ValidationResponse struct {
    Valid    bool
    Checks   map[string]CheckResult
    Errors   []ValidationError
    Warnings []ValidationWarning
}
```

### 5. Launching Phase Plugin Protocol

**Purpose**: Allow plugins to customize service launching or add services.

**Request**:
```json
{
  "jsonrpc": "2.0",
  "method": "launch",
  "params": {
    "phase": "launch",
    "services": [
      {
        "name": "backend",
        "command": ["go", "run", "./cmd/moments-server", "serve"],
        "port": 8083,
        "health_check": {"type": "http", "endpoint": "/rpc/v1/health"}
      }
    ],
    "config": {...}
  },
  "id": 1
}
```

**Response**:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "success": true,
    "services": [
      {
        "name": "backend",
        "modified": true,
        "command": ["go", "run", "./cmd/moments-server", "serve", "--custom-flag"],
        "env": {"CUSTOM_VAR": "value"}
      },
      {
        "name": "custom_service",
        "added": true,
        "command": ["./custom-service"],
        "port": 9000
      }
    ],
    "messages": [
      {"level": "info", "message": "Added custom service"}
    ]
  },
  "id": 1
}
```

**Plugin Interface**:
```go
type LaunchPlugin interface {
    Launch(ctx context.Context, req LaunchRequest) (*LaunchResponse, error)
}

type LaunchRequest struct {
    Services []Service
    Config   map[string]interface{}
}

type LaunchResponse struct {
    Services []Service
    Messages []PluginMessage
}
```

### 6. Logging Phase Plugin Protocol

**Purpose**: Allow plugins to add log sources or modify log handling.

**Request**:
```json
{
  "jsonrpc": "2.0",
  "method": "logs",
  "params": {
    "phase": "logs",
    "sources": [
      {"id": "backend", "type": "file", "path": "/tmp/backend.log"},
      {"id": "frontend", "type": "file", "path": "/tmp/frontend.log"}
    ]
  },
  "id": 1
}
```

**Response**:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "success": true,
    "sources": [
      {"id": "backend", "added": true},
      {"id": "custom", "added": true, "type": "file", "path": "/tmp/custom.log"}
    ],
    "filters": [
      {"source": "backend", "pattern": "ERROR", "action": "highlight"}
    ],
    "messages": [
      {"level": "info", "message": "Added custom log source"}
    ]
  },
  "id": 1
}
```

### Plugin Discovery and Registration

#### Plugin Discovery

Plugins are discovered via configuration file or directory scanning:

**Configuration File** (`moments-dev.yaml`):
```yaml
plugins:
  config:
    - name: "custom-config"
      path: "./scripts/config-plugin.sh"
      enabled: true
  build:
    - name: "custom-build"
      path: "./scripts/build-plugin.py"
      enabled: true
  env:
    - name: "custom-env"
      path: "./scripts/env-plugin.sh"
      enabled: false
```

**Directory Scanning** (`plugins/` directory):
```
plugins/
  config/
    custom-config.sh
  build/
    custom-build.py
  validate/
    custom-validate.go
```

#### Plugin Execution

**Plugin Runner Interface**:
```go
type PluginRunner interface {
    RunPlugin(ctx context.Context, plugin Plugin, req interface{}) (interface{}, error)
}

type Plugin struct {
    Name     string
    Path     string
    Phase    string
    Enabled  bool
    Timeout  time.Duration
}
```

**Execution Flow**:
1. Discover plugins for phase
2. Filter enabled plugins
3. For each plugin:
   - Spawn process (script or binary)
   - Send JSON-RPC request via stdin
   - Read JSON-RPC response from stdout
   - Handle errors/timeouts
   - Merge results with previous plugins
4. Return aggregated result

**Code Example**:
```go
func (r *StdioPluginRunner) RunPlugin(ctx context.Context, plugin Plugin, req interface{}) (interface{}, error) {
    // Spawn plugin process
    cmd := exec.CommandContext(ctx, plugin.Path)
    cmd.Stdin = strings.NewReader(jsonRequest)
    
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        return nil, err
    }
    
    if err := cmd.Start(); err != nil {
        return nil, err
    }
    
    // Read response with timeout
    responseCh := make(chan []byte, 1)
    errCh := make(chan error, 1)
    
    go func() {
        data, err := io.ReadAll(stdout)
        responseCh <- data
        errCh <- err
    }()
    
    select {
    case <-ctx.Done():
        cmd.Process.Kill()
        return nil, ctx.Err()
    case data := <-responseCh:
        if err := <-errCh; err != nil {
            return nil, err
        }
        return parseResponse(data)
    }
}
```

### Plugin Result Aggregation

Results from multiple plugins are aggregated:

**Configuration Phase**:
- Config values: Later plugins override earlier ones
- VITE env: Merged (all plugins can add variables)
- Messages: Collected from all plugins

**Build Phase**:
- Built targets: Union of all plugins
- Artifacts: Merged maps
- Messages: Collected

**Validation Phase**:
- Checks: Merged maps
- Errors/Warnings: Collected from all plugins
- Valid: AND of all plugin results (all must be valid)

### Error Handling

**Plugin Failures**:
- Plugin errors don't crash main tool
- Errors logged with plugin name
- Option to continue or abort on plugin failure
- Timeout handling (default: 30 seconds)

**Error Recovery**:
- Failed plugins are skipped
- Remaining plugins continue execution
- Final result includes plugin failure information

### Security Considerations

**Plugin Execution**:
- Plugins run with same privileges as main tool
- No sandboxing (plugins have full access)
- Path validation (prevent arbitrary command execution)
- Timeout limits prevent hanging plugins

**Recommendations**:
- Only enable trusted plugins
- Review plugin code before enabling
- Use absolute paths for plugin executables
- Consider sandboxing for untrusted plugins (future)

### Example: Complete Plugin Workflow

**Scenario**: Colleague wants to add a custom service that runs before backend.

**Step 1**: Create plugin script (`plugins/launch/custom-service.sh`):
```bash
#!/bin/bash
while read -r line; do
    request=$(echo "$line" | jq -r '.')
    services=$(echo "$request" | jq -r '.params.services')
    
    # Add custom service before backend
    custom_service='{"name": "custom", "command": ["./custom-service"], "port": 9000}'
    services=$(echo "$services" | jq ". = [$custom_service] + .")
    
    response=$(jq -n \
        --argjson services "$services" \
        '{
            "jsonrpc": "2.0",
            "result": {"success": true, "services": $services, "messages": []},
            "id": 1
        }')
    
    echo "$response"
done
```

**Step 2**: Enable plugin in `moments-dev.yaml`:
```yaml
plugins:
  launch:
    - name: "custom-service"
      path: "./plugins/launch/custom-service.sh"
      enabled: true
```

**Step 3**: Tool automatically discovers and executes plugin during launch phase.

### Benefits of Plugin Protocol

1. **Extensibility**: Easy to add custom logic without modifying core code
2. **Language Agnostic**: Plugins can be written in any language
3. **Composability**: Multiple plugins can extend the same phase
4. **Isolation**: Plugin failures don't crash main tool
5. **Generic Tooling**: Tool becomes generic orchestrator usable for different projects
6. **Team Collaboration**: Colleagues can share plugins via git

### Implementation Considerations

**Phase 1**: Basic plugin support
- Stdio communication
- JSON-RPC protocol
- Single plugin per phase
- Basic error handling

**Phase 2**: Advanced features
- Multiple plugins per phase
- Plugin result aggregation
- Plugin discovery (config file + directory)
- Plugin timeout and retry

**Phase 3**: Enhanced features
- Plugin marketplace/sharing
- Plugin validation
- Plugin sandboxing (optional)
- Plugin metrics/observability

## Open Questions

1. **State Persistence**: Should we persist process state (PIDs, status) to disk for recovery after crashes?
   - **Option A**: In-memory only (simpler, but lost on crash)
   - **Option B**: Persist to file (survives crashes, but needs cleanup)

2. **Configuration Hot Reload**: Should configuration be reloadable without restart?
   - **Option A**: Load once at startup (simpler)
   - **Option B**: Support reload command (more flexible)

3. **Multi-Project Support**: Should the tool support multiple projects/workspaces?
   - **Option A**: Single project only (simpler)
   - **Option B**: Multi-project support (more complex, but useful)

4. **Plugin Protocol Versioning**: How should we handle plugin protocol versioning?
   - **Option A**: Single version, breaking changes require plugin updates
   - **Option B**: Versioned protocol with backward compatibility

## Related Documents

- [startdev.sh Complete Step-by-Step Analysis](../analysis/01-startdev-sh-complete-step-by-step-analysis.md) - Detailed analysis of current bash script
- [moments-config and Configuration Phase Analysis](../analysis/02-moments-config-and-configuration-phase-analysis.md) - Detailed analysis of configuration system
- [Research Diary](../reference/01-diary.md) - Research process and findings

## Conclusion

This architecture provides a solid foundation for replacing the bash script with a maintainable, testable Go program. The separation of concerns into seven domains enables independent development, testing, and evolution. The design maintains compatibility with existing tooling while providing a path for future optimizations.

The phased implementation plan allows for incremental development and testing, reducing risk while delivering value early. Key decisions prioritize maintainability, testability, and developer experience while keeping the door open for future improvements.
