---
Title: startdev.sh Complete Step-by-Step Analysis
Ticket: MO-005-IMPROVE-STARTDEV
Status: active
Topics:
    - dev-tooling
    - scripts
    - bash
    - dev-environment
DocType: analysis
Intent: long-term
Owners:
    - team
RelatedFiles: []
ExternalSources: []
Summary: "Exhaustive step-by-step analysis of startdev.sh script, documenting every function, variable, command, and interaction with detailed pseudocode and code references."
LastUpdated: 2026-01-06T13:01:33.97797899-05:00
WhatFor: "Understanding the complete execution flow of startdev.sh for debugging, enhancement, and maintenance purposes."
WhenToUse: "When modifying startdev.sh, debugging startup issues, or understanding the dev environment initialization process."
---

# startdev.sh Complete Step-by-Step Analysis

## Executive Summary

`startdev.sh` is a comprehensive development environment orchestration script that coordinates the startup of both the Moments backend server and frontend Vite dev server. It handles configuration resolution, dependency management, port conflict resolution, process lifecycle management, and provides structured logging output. The script is designed to be idempotent and safe to run repeatedly, making it suitable for daily development workflows.

The script operates in three main phases: (1) **Preparation** - resolving configuration, ensuring CLI tools are built, and checking dependencies; (2) **Port Management** - killing existing processes on target ports to prevent conflicts; (3) **Service Startup** - launching backend and frontend servers in background processes with health checking and status reporting.

## Script Overview

**Location**: `moments/scripts/startdev.sh`  
**Purpose**: Unified dev environment starter for Moments web app  
**Dependencies**: 
- `moments-config` CLI (`backend/dist/moments-config`)
- `pnpm` package manager
- `lsof` command (for port checking)
- `make` (for backend build/run)
- Backend Makefile (`backend/Makefile`)
- Frontend package.json (`web/package.json`)

**Key Features**:
- Derives VITE_* environment variables from appconfig via `moments-config`
- Starts backend server via `make run` (which runs `bootstrap` then `go run`)
- Starts Vite dev server with proper proxy configuration
- Manages port conflicts automatically
- Provides structured logging and status reporting
- Handles graceful shutdown on SIGINT/SIGTERM

## Detailed Step-by-Step Analysis

### Phase 1: Script Initialization and Setup

#### Step 1.1: Shebang and Error Handling Setup

```bash
#!/usr/bin/env bash
set -Eeuo pipefail
```

**What happens**:
- `#!/usr/bin/env bash` - Uses the system's `bash` interpreter found in PATH
- `set -Eeuo pipefail` - Enables strict error handling:
  - `-E`: ERR trap is inherited by shell functions
  - `-e`: Exit immediately if any command exits with non-zero status
  - `-u`: Treat unset variables as errors
  - `-o pipefail`: Pipeline returns the exit status of the last command to exit with non-zero status

**Code Reference**: Lines 1-2 of `moments/scripts/startdev.sh`

#### Step 1.2: Variable Initialization

```bash
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MOMENTS_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="${MOMENTS_ROOT}"
CONFIG_CLI="${MOMENTS_ROOT}/backend/dist/moments-config"
```

**Pseudocode**:
```
SCRIPT_DIR = resolve absolute path of directory containing startdev.sh
MOMENTS_ROOT = resolve parent directory of SCRIPT_DIR (moments/)
REPO_ROOT = MOMENTS_ROOT (alias for clarity)
CONFIG_CLI = MOMENTS_ROOT + "/backend/dist/moments-config"
```

**What happens**:
- `SCRIPT_DIR` resolves to `moments/scripts/` (absolute path)
- `MOMENTS_ROOT` resolves to `moments/` (absolute path)
- `CONFIG_CLI` points to the expected location of the `moments-config` binary

**Code Reference**: Lines 9-14 of `moments/scripts/startdev.sh`

#### Step 1.3: Logging Function Definitions

```bash
info()  { echo -e "[startdev] $*"; }
warn()  { echo -e "[startdev][warn] $*" >&2; }
error() { echo -e "[startdev][error] $*" >&2; }
```

**What happens**:
- `info()` - Prints prefixed messages to stdout
- `warn()` - Prints prefixed messages to stderr
- `error()` - Prints prefixed messages to stderr

**Code Reference**: Lines 16-18 of `moments/scripts/startdev.sh`

#### Step 1.4: Cleanup Handler Registration

```bash
cleanup() {
	echo ""
	info "Shutting down servers..."
	exit 0
}

trap cleanup INT TERM
```

**What happens**:
- Defines `cleanup()` function that prints shutdown message and exits
- Registers `cleanup` as trap handler for SIGINT (Ctrl+C) and SIGTERM signals
- When user presses Ctrl+C or script receives SIGTERM, cleanup runs before exit

**Code Reference**: Lines 20-27 of `moments/scripts/startdev.sh`

**Note**: The cleanup function is minimal - it doesn't actually kill background processes. This is intentional as the script waits for processes to exit naturally.

### Phase 2: Configuration CLI Preparation

#### Step 2.1: Ensure moments-config CLI Exists

```bash
ensure_config_cli() {
	if [[ -x "${CONFIG_CLI}" ]]; then
		return 0
	fi
	info "Building moments-config CLI..."
	make -C "${MOMENTS_ROOT}/backend" build >/dev/null
	if [[ ! -x "${CONFIG_CLI}" ]]; then
		error "moments-config CLI not found at ${CONFIG_CLI}"
		exit 1
	fi
}
```

**Pseudocode**:
```
FUNCTION ensure_config_cli():
    IF CONFIG_CLI exists and is executable:
        RETURN success
    PRINT "Building moments-config CLI..."
    RUN make build in backend directory (suppress output)
    IF CONFIG_CLI still doesn't exist or isn't executable:
        PRINT error message
        EXIT with code 1
```

**What happens**:
- Checks if `moments-config` binary exists at expected path and is executable
- If missing, runs `make build` in `backend/` directory (which builds both `moments-server` and `moments-config`)
- Verifies binary was created successfully
- Exits with error if build fails

**Code Reference**: Lines 29-39 of `moments/scripts/startdev.sh`

**Related Code**: 
- `moments/backend/Makefile` lines 18-26 define the `build` target
- `moments/backend/cmd/moments-config/main.go` is the entry point for the CLI

#### Step 2.2: Configuration Value Retrieval Function

```bash
cfg_get() {
	local key="$1"
	"${CONFIG_CLI}" get "${key}" --repo-root "${REPO_ROOT}" 2>/dev/null || true
}
```

**Pseudocode**:
```
FUNCTION cfg_get(key):
    local key = first argument
    RUN moments-config get key --repo-root REPO_ROOT
    SUPPRESS stderr
    RETURN empty string if command fails (don't exit)
```

**What happens**:
- Calls `moments-config get <key>` with repo root flag
- Suppresses stderr to avoid noise from missing config keys
- Returns empty string (via `|| true`) if key doesn't exist or CLI fails
- This allows graceful fallback to defaults

**Code Reference**: Lines 41-44 of `moments/scripts/startdev.sh`

**Related Code**:
- `moments/backend/cmd/moments-config/get_cmd.go` implements the `get` command
- Supported keys: `repo_root`, `database_url`, `db_name`, `db_admin_url`, `keys_dir`, `private_key_filename`, `public_key_filename`, `openssl_bin`
- `moments/backend/cmd/moments-config/bootstrap_config.go` lines 102-178 compute bootstrap config from appconfig

### Phase 3: Port Conflict Detection

#### Step 3.1: Port Checking Function

```bash
check_port() {
	lsof -Pi :"$1" -sTCP:LISTEN -t >/dev/null 2>&1
}
```

**Pseudocode**:
```
FUNCTION check_port(port):
    RUN lsof -Pi :port -sTCP:LISTEN -t
    SUPPRESS stdout and stderr
    RETURN exit code (0 if port in use, non-zero if free)
```

**What happens**:
- Uses `lsof` to check if a port is listening
- `-Pi :PORT` - Check IPv4/IPv6 port
- `-sTCP:LISTEN` - Only TCP sockets in LISTEN state
- `-t` - Print only PIDs (suppressed here, only used for exit code)
- Returns 0 (success) if port is in use, non-zero if free

**Code Reference**: Lines 46-49 of `moments/scripts/startdev.sh`

**Dependencies**: Requires `lsof` command to be installed (standard on macOS, may need installation on Linux)

### Phase 4: Main Function - Configuration Resolution

#### Step 4.1: Initial Preparation

```bash
main() {
	info "Preparing dev environment"
	ensure_config_cli
```

**What happens**:
- Entry point for script execution
- Prints preparation message
- Ensures `moments-config` CLI is built and available

**Code Reference**: Lines 51-53 of `moments/scripts/startdev.sh`

#### Step 4.2: Configuration Value Resolution

```bash
	local backend_url identity_url stytch_public stytch_hosted
	backend_url="$(cfg_get 'platform.mento-service-public-base-url')"
	identity_url="$(cfg_get 'platform.mento-service-identity-base-url')"
	stytch_public="$(cfg_get 'integrations.stytch.stytch-public-token')"
	stytch_hosted="$(cfg_get 'integrations.stytch.stytch-hosted-url')"
```

**Pseudocode**:
```
DECLARE local variables: backend_url, identity_url, stytch_public, stytch_hosted
backend_url = cfg_get('platform.mento-service-public-base-url')
identity_url = cfg_get('platform.mento-service-identity-base-url')
stytch_public = cfg_get('integrations.stytch.stytch-public-token')
stytch_hosted = cfg_get('integrations.stytch.stytch-hosted-url')
```

**What happens**:
- Declares local variables for configuration values
- Calls `cfg_get()` for each configuration key
- Keys are dot-notation paths into the YAML config hierarchy
- Values may be empty if not configured

**Code Reference**: Lines 56-61 of `moments/scripts/startdev.sh`

**Related Code**:
- Configuration keys reference `moments/config/app/base.yaml` and `moments/config/app/local.yaml`
- `moments-config` uses `appconfig` package to read and resolve these values
- See `moments/backend/pkg/appconfig/` for config loading logic

#### Step 4.3: Fallback Defaults for Local Development

```bash
	if [[ -z "${backend_url}" ]]; then backend_url="http://localhost:8082"; fi
	if [[ -z "${identity_url}" ]]; then identity_url="http://localhost:8083"; fi
```

**Pseudocode**:
```
IF backend_url is empty:
    backend_url = "http://localhost:8082"
IF identity_url is empty:
    identity_url = "http://localhost:8083"
```

**What happens**:
- Sets default URLs for local development when config values are missing
- Backend defaults to port 8082 (though actual backend runs on 8083)
- Identity service defaults to port 8083

**Code Reference**: Lines 63-65 of `moments/scripts/startdev.sh`

**Note**: There's a potential inconsistency - `backend_url` defaults to 8082 but the backend actually runs on 8083. This may be intentional if there's a proxy or if the config should override this.

#### Step 4.4: Export VITE Environment Variables

```bash
	export VITE_BACKEND_URL="${backend_url}"
	export VITE_IDENTITY_BACKEND_URL="${identity_url}"
	export VITE_IDENTITY_SERVICE_URL="${identity_url}"
	if [[ -n "${stytch_public}" ]]; then
		export VITE_STYTCH_PUBLIC_TOKEN="${stytch_public}"
	fi
```

**Pseudocode**:
```
EXPORT VITE_BACKEND_URL = backend_url
EXPORT VITE_IDENTITY_BACKEND_URL = identity_url
EXPORT VITE_IDENTITY_SERVICE_URL = identity_url
IF stytch_public is not empty:
    EXPORT VITE_STYTCH_PUBLIC_TOKEN = stytch_public
```

**What happens**:
- Exports environment variables prefixed with `VITE_` for Vite dev server
- Vite only exposes variables starting with `VITE_` to the frontend code
- Sets backend and identity URLs
- Conditionally exports Stytch token if configured

**Code Reference**: Lines 67-73 of `moments/scripts/startdev.sh`

**Related Code**:
- `moments/web/vite.config.mts` lines 34-37 read these environment variables
- Vite's `loadEnv()` function loads `VITE_*` prefixed variables
- See `moments/web/vite.config.mts` lines 69-103 for proxy configuration using these URLs

#### Step 4.5: Display Configuration

```bash
	info "VITE_BACKEND_URL=${VITE_BACKEND_URL}"
	info "VITE_IDENTITY_BACKEND_URL=${VITE_IDENTITY_BACKEND_URL}"
	if [[ -n "${VITE_STYTCH_PUBLIC_TOKEN:-}" ]]; then
		info "VITE_STYTCH_PUBLIC_TOKEN=***"
	fi
	if [[ -n "${VITE_STYTCH_HOSTED_URL:-}" ]]; then
		info "VITE_STYTCH_HOSTED_URL=${VITE_STYTCH_HOSTED_URL}"
	fi
```

**What happens**:
- Prints resolved configuration values for debugging
- Masks Stytch token with `***` for security
- Uses `${VAR:-}` syntax to safely check if variable is set

**Code Reference**: Lines 78-85 of `moments/scripts/startdev.sh`

#### Step 4.6: Port Configuration

```bash
	local backend_port="${PORT:-8083}"
	local frontend_port="${VITE_PORT:-5173}"
```

**Pseudocode**:
```
backend_port = PORT environment variable OR default to 8083
frontend_port = VITE_PORT environment variable OR default to 5173
```

**What happens**:
- Reads port numbers from environment variables with defaults
- Backend defaults to 8083 (matches Makefile default)
- Frontend defaults to 5173 (Vite's default port)

**Code Reference**: Lines 87-89 of `moments/scripts/startdev.sh`

**Related Code**:
- `moments/backend/Makefile` line 14 defines `PORT ?= 8083`
- `moments/web/vite.config.mts` line 70 allows port override via CLI flag

### Phase 5: Port Conflict Resolution

#### Step 5.1: Kill Existing Processes on Ports

```bash
	info "Stopping existing services..."
	for port in "${backend_port}" "${frontend_port}"; do
		if lsof -Pi :"${port}" -sTCP:LISTEN -t >/dev/null 2>&1; then
			info "  - Killing process on port ${port}..."
			kill -9 $(lsof -Pi :"${port}" -sTCP:LISTEN -t) 2>/dev/null || true
		fi
	done
	sleep 1
```

**Pseudocode**:
```
PRINT "Stopping existing services..."
FOR each port in [backend_port, frontend_port]:
    IF port is in use (lsof returns success):
        PRINT "Killing process on port..."
        GET PID from lsof
        KILL process with SIGKILL (-9)
        SUPPRESS errors (process may have exited)
WAIT 1 second for ports to be released
```

**What happens**:
- Iterates over backend and frontend ports
- Checks if port is in use using `lsof`
- If in use, extracts PID and kills with `kill -9` (SIGKILL, cannot be caught)
- Suppresses errors (process may exit between check and kill)
- Waits 1 second for ports to be released

**Code Reference**: Lines 91-99 of `moments/scripts/startdev.sh`

**Safety Note**: Using `kill -9` is forceful and may cause data loss if processes aren't gracefully shutdown. However, for dev environment this is acceptable.

### Phase 6: Frontend Dependencies Preparation

#### Step 6.1: Ensure Web Dependencies

```bash
	pushd "${MOMENTS_ROOT}/web" >/dev/null
	if command -v pnpm >/dev/null 2>&1; then
		info "Ensuring web dependencies (pnpm install)..."
		pnpm install --prefer-offline
		info "Generating version file..."
		pnpm generate-version
	else
		error "pnpm not found. Install pnpm to run the web dev server."
		exit 1
	fi
	popd >/dev/null
```

**Pseudocode**:
```
CHANGE directory to web/
IF pnpm command exists:
    PRINT "Ensuring web dependencies..."
    RUN pnpm install --prefer-offline
    PRINT "Generating version file..."
    RUN pnpm generate-version
ELSE:
    PRINT error message
    EXIT with code 1
RESTORE previous directory
```

**What happens**:
- Changes to `web/` directory
- Checks if `pnpm` is available
- Runs `pnpm install --prefer-offline` to install/update dependencies (prefers cached packages)
- Runs `pnpm generate-version` to generate version metadata file
- Exits with error if `pnpm` not found

**Code Reference**: Lines 101-112 of `moments/scripts/startdev.sh`

**Related Code**:
- `moments/web/package.json` line 6 defines `generate-version` script
- `moments/web/scripts/generate-version.js` implements version file generation
- `moments/web/package.json` line 7 defines `dev` script that also runs `generate-version`

### Phase 7: Service Startup

#### Step 7.1: Create Log Directory and Files

```bash
	info "Starting all services..."
	echo ""

	mkdir -p "${MOMENTS_ROOT}/tmp"
	local timestamp=$(date +%Y%m%d-%H%M%S)
	local backend_log="${MOMENTS_ROOT}/tmp/backend-${timestamp}.log"
	local frontend_log="${MOMENTS_ROOT}/tmp/frontend-${timestamp}.log"
	: >"${backend_log}"
	: >"${frontend_log}"
```

**Pseudocode**:
```
PRINT "Starting all services..."
CREATE tmp directory if it doesn't exist
timestamp = current date/time in format YYYYMMDD-HHMMSS
backend_log = tmp/backend-{timestamp}.log
frontend_log = tmp/frontend-{timestamp}.log
CREATE empty log files (truncate if exists)
```

**What happens**:
- Creates `tmp/` directory for log files
- Generates timestamp for unique log filenames
- Creates empty log files for backend and frontend
- Uses `: >file` syntax (no-op command redirecting to file) to create/truncate files

**Code Reference**: Lines 114-123 of `moments/scripts/startdev.sh`

**Related Code**: Log files are written to `moments/tmp/` directory

#### Step 7.2: Start Backend Server

```bash
	info "Starting backend server (make run)..."
	pushd "${MOMENTS_ROOT}/backend" >/dev/null
	PORT="${backend_port}" make run >"${backend_log}" 2>&1 &
	BACKEND_PID=$!
	popd >/dev/null
```

**Pseudocode**:
```
PRINT "Starting backend server..."
CHANGE directory to backend/
SET PORT environment variable
RUN make run (redirect stdout and stderr to log file, run in background)
CAPTURE process ID
RESTORE previous directory
```

**What happens**:
- Changes to `backend/` directory
- Sets `PORT` environment variable for Makefile
- Runs `make run` which:
  - First runs `make bootstrap` (ensures DB and keys exist)
  - Then runs `go run ./cmd/moments-server serve` with ldflags
- Redirects all output (stdout and stderr) to log file
- Runs in background (`&`)
- Captures process ID in `BACKEND_PID` variable

**Code Reference**: Lines 125-130 of `moments/scripts/startdev.sh`

**Related Code**:
- `moments/backend/Makefile` lines 34-39 define `run` target
- `moments/backend/Makefile` line 28-32 define `bootstrap` target
- `moments/backend/scripts/bootstrap-startup.sh` is executed by `make bootstrap`
- `moments/backend/cmd/moments-server/serve.go` is the entry point for the server

**Detailed Flow of `make run`**:
1. `make bootstrap` runs `backend/scripts/bootstrap-startup.sh`
   - Ensures Postgres database exists
   - Runs migrations via `go run ./cmd/moments-server migrate up`
   - Generates RSA JWT keys if missing
2. `go run ./cmd/moments-server serve` starts HTTP server
   - Initializes appconfig from YAML files
   - Connects to database via Ent
   - Initializes Redis connection
   - Registers HTTP routes (identity, RPC, webchat, etc.)
   - Starts HTTP server on configured port

#### Step 7.3: Start Frontend Dev Server

```bash
	info "Starting Vite dev server..."
	pushd "${MOMENTS_ROOT}/web" >/dev/null
	pnpm run dev -- --port "${frontend_port}" \
		> >(tee -a "${frontend_log}") \
		2> >(tee -a "${frontend_log}" >&2) &
	WEB_PID=$!
	popd >/dev/null
```

**Pseudocode**:
```
PRINT "Starting Vite dev server..."
CHANGE directory to web/
RUN pnpm run dev --port {frontend_port}
    REDIRECT stdout to tee (append to log, also print to terminal)
    REDIRECT stderr to tee (append to log, also print to stderr)
RUN in background
CAPTURE process ID
RESTORE previous directory
```

**What happens**:
- Changes to `web/` directory
- Runs `pnpm run dev` with `--port` flag
- Uses `tee -a` to both append to log file AND display in terminal
- Process substitution `>(tee ...)` allows simultaneous logging and display
- Runs in background and captures PID

**Code Reference**: Lines 132-139 of `moments/scripts/startdev.sh`

**Related Code**:
- `moments/web/package.json` line 7 defines `dev` script: `"dev": "pnpm generate-version && vite"`
- `moments/web/vite.config.mts` configures Vite dev server
- Vite reads `VITE_*` environment variables set earlier in script

### Phase 8: Health Checking and Status Reporting

#### Step 8.1: Wait for Services to Start

```bash
	info "Waiting for services to start..."
	local max_wait=30
	local elapsed=0
	while [ $elapsed -lt $max_wait ]; do
		local all_up=true

		if ! check_port "${backend_port}"; then all_up=false; fi
		if ! check_port "${frontend_port}"; then all_up=false; fi

		if [ "$all_up" = true ]; then
			info "All services are up!"
			break
		fi

		sleep 1
		elapsed=$((elapsed + 1))
		if [ $((elapsed % 5)) -eq 0 ]; then
			info "  Still waiting... (${elapsed}s)"
		fi
	done
```

**Pseudocode**:
```
PRINT "Waiting for services to start..."
max_wait = 30 seconds
elapsed = 0
WHILE elapsed < max_wait:
    all_up = true
    IF backend port not listening:
        all_up = false
    IF frontend port not listening:
        all_up = false
    IF all_up is true:
        PRINT "All services are up!"
        BREAK loop
    SLEEP 1 second
    elapsed = elapsed + 1
    IF elapsed is multiple of 5:
        PRINT "Still waiting... (elapsed seconds)"
```

**What happens**:
- Waits up to 30 seconds for both services to start
- Checks ports every second using `check_port()` function
- Sets `all_up` flag to false if any port not listening
- Breaks early if both ports are listening
- Prints progress message every 5 seconds
- Continues loop until timeout or both services up

**Code Reference**: Lines 141-162 of `moments/scripts/startdev.sh`

**Limitation**: Only checks if ports are listening, not if services are actually healthy. A service could be listening but not ready to serve requests.

#### Step 8.2: Display Service Status

```bash
	echo ""
	echo "=============================================="
	echo "Service Status"
	echo "=============================================="
	echo ""

	local backend_status="❌ FAILED"
	local frontend_status="❌ FAILED"
	if check_port "${backend_port}"; then
		backend_status="✅ RUNNING"
	fi

	if check_port "${frontend_port}"; then
		frontend_status="✅ RUNNING"
	fi

	echo "Services:"
	echo "  Backend (${backend_port}):     ${backend_status}"
	echo "  Frontend (${frontend_port}):   ${frontend_status}"
```

**Pseudocode**:
```
PRINT separator line
PRINT "Service Status"
PRINT separator line
PRINT empty line

backend_status = "❌ FAILED"
frontend_status = "❌ FAILED"
IF backend port is listening:
    backend_status = "✅ RUNNING"
IF frontend port is listening:
    frontend_status = "✅ RUNNING"

PRINT "Services:"
PRINT "  Backend (port): status"
PRINT "  Frontend (port): status"
```

**What happens**:
- Prints formatted status section
- Checks each port again to determine final status
- Displays port numbers and status with emoji indicators
- Provides visual feedback on service health

**Code Reference**: Lines 164-184 of `moments/scripts/startdev.sh`

#### Step 8.3: Display Log File Locations

```bash
	echo ""
	echo "=============================================="
	echo "Log Files"
	echo "=============================================="
	echo ""
	echo "  Backend:        ${backend_log}"
	echo "  Frontend:       ${frontend_log}"
	echo ""
	echo "Quick commands:"
	echo "  tail -f ${backend_log}"
	echo "  tail -f ${frontend_log}"
	echo "  tail -f ${backend_log} ${frontend_log}"
```

**What happens**:
- Prints log file locations for easy reference
- Provides copy-paste ready `tail -f` commands for monitoring logs
- Shows both individual and combined log tailing commands

**Code Reference**: Lines 185-195 of `moments/scripts/startdev.sh`

### Phase 9: Process Management and Cleanup

#### Step 9.1: Wait for Process Completion

```bash
	echo ""
	echo "Press Ctrl+C to stop all servers"
	echo ""

	if help wait 2>/dev/null | grep -q -- '-n'; then
		wait -n "${BACKEND_PID}" "${WEB_PID}"
	else
		wait "${BACKEND_PID}" "${WEB_PID}"
	done
	cleanup
```

**Pseudocode**:
```
PRINT "Press Ctrl+C to stop all servers"

IF wait command supports -n flag (wait for first process):
    WAIT for first of [BACKEND_PID, WEB_PID] to exit
ELSE:
    WAIT for both processes to exit
CALL cleanup function
```

**What happens**:
- Checks if `wait` command supports `-n` flag (wait for first process to exit)
- If supported, uses `wait -n` to exit when first process dies (better UX)
- Otherwise, waits for both processes (script exits when both done)
- Calls cleanup function on exit

**Code Reference**: Lines 197-206 of `moments/scripts/startdev.sh`

**Note**: The `wait -n` check uses `help wait` which may not work on all systems. This is a best-effort check for improved behavior.

## Configuration Flow Diagram

```
startdev.sh execution flow:

1. Initialize script
   ├─ Set error handling (set -Eeuo pipefail)
   ├─ Resolve paths (SCRIPT_DIR, MOMENTS_ROOT)
   └─ Define helper functions (info, warn, error, cleanup)

2. Ensure moments-config CLI
   ├─ Check if binary exists
   ├─ If missing: run `make build` in backend/
   └─ Verify binary is executable

3. Resolve configuration
   ├─ Call moments-config get for each key:
   │  ├─ platform.mento-service-public-base-url
   │  ├─ platform.mento-service-identity-base-url
   │  ├─ integrations.stytch.stytch-public-token
   │  └─ integrations.stytch.stytch-hosted-url
   ├─ Apply fallback defaults
   └─ Export VITE_* environment variables

4. Port conflict resolution
   ├─ Check backend_port (default: 8083)
   ├─ Check frontend_port (default: 5173)
   ├─ Kill existing processes on ports
   └─ Wait 1 second for ports to release

5. Prepare frontend dependencies
   ├─ Change to web/ directory
   ├─ Run pnpm install --prefer-offline
   └─ Run pnpm generate-version

6. Start backend server
   ├─ Change to backend/ directory
   ├─ Set PORT environment variable
   ├─ Run `make run` in background
   │  ├─ make bootstrap (ensures DB + keys)
   │  └─ go run ./cmd/moments-server serve
   └─ Capture process ID

7. Start frontend server
   ├─ Change to web/ directory
   ├─ Run `pnpm run dev --port` in background
   └─ Capture process ID

8. Health checking
   ├─ Wait up to 30 seconds
   ├─ Check ports every second
   └─ Break when both ports listening

9. Display status and wait
   ├─ Print service status
   ├─ Print log file locations
   └─ Wait for processes (or Ctrl+C)
```

## Dependencies and Interactions

### External Dependencies

1. **moments-config CLI** (`backend/dist/moments-config`)
   - Built from `backend/cmd/moments-config/`
   - Reads YAML config files via `appconfig` package
   - Provides `get` and `bootstrap` commands

2. **make** (GNU Make)
   - Used to build backend binaries
   - Executes `backend/Makefile` targets

3. **pnpm** (Package Manager)
   - Manages frontend dependencies
   - Runs npm scripts defined in `web/package.json`

4. **lsof** (List Open Files)
   - Checks port availability
   - Kills processes on ports

5. **PostgreSQL**
   - Required by backend server
   - Database initialized by `bootstrap-startup.sh`

6. **Go toolchain**
   - Required to build and run backend
   - Used by `make run` to execute server

### Internal Dependencies

1. **backend/Makefile**
   - Defines `run` target (depends on `bootstrap`)
   - Defines `bootstrap` target (runs `bootstrap-startup.sh`)
   - Defines `build` target (builds binaries)

2. **backend/scripts/bootstrap-startup.sh**
   - Ensures database exists
   - Runs migrations
   - Generates JWT keys

3. **web/package.json**
   - Defines `dev` script
   - Defines `generate-version` script

4. **web/vite.config.mts**
   - Configures Vite dev server
   - Sets up proxy rules using `VITE_*` env vars

5. **config/app/base.yaml** and **config/app/local.yaml**
   - Source of configuration values
   - Read by `moments-config` CLI

## Error Handling

### Error Scenarios

1. **moments-config CLI missing or build fails**
   - Script exits with error code 1
   - Error message printed to stderr

2. **pnpm not found**
   - Script exits with error code 1
   - Error message instructs user to install pnpm

3. **Port already in use (after kill attempt)**
   - Script continues (may fail later when binding port)
   - No explicit error handling

4. **Backend server fails to start**
   - Process exits, `wait` returns
   - Script exits via cleanup handler
   - Log file contains error details

5. **Frontend server fails to start**
   - Process exits, `wait` returns
   - Script exits via cleanup handler
   - Log file contains error details

6. **Services don't start within 30 seconds**
   - Script continues (doesn't exit)
   - Status shows "FAILED" for non-listening ports
   - User can check log files for details

### Error Recovery

- Script uses `set -e` to exit on errors
- Background processes write errors to log files
- User can inspect log files for detailed error messages
- Port conflicts are automatically resolved (processes killed)

## Process Management

### Background Process Execution

Both backend and frontend run as background processes:
- Started with `&` operator
- PIDs captured in `BACKEND_PID` and `WEB_PID`
- Output redirected to log files

### Signal Handling

- `trap cleanup INT TERM` registers cleanup handler
- Ctrl+C (SIGINT) triggers cleanup
- SIGTERM triggers cleanup
- Cleanup function prints message and exits

**Note**: Cleanup doesn't kill background processes. Processes exit naturally when:
- User presses Ctrl+C (signals propagate to child processes)
- Process encounters error
- Process completes (unlikely for servers)

### Process Monitoring

- Script waits for processes using `wait` command
- Uses `wait -n` if available (exits when first process dies)
- Otherwise waits for both processes

## Logging Strategy

### Log File Management

- Log files created in `moments/tmp/` directory
- Filenames include timestamp: `backend-{timestamp}.log`
- Files created empty, then appended to

### Backend Logging

- All stdout and stderr redirected to log file
- No terminal output (except via `make` echo statements)
- Log contains Go server output, database logs, etc.

### Frontend Logging

- Uses `tee -a` to both log and display
- Terminal shows Vite dev server output
- Log file contains same output
- Useful for debugging while seeing real-time output

### Status Output

- Script prints status messages to terminal
- Service status displayed with emoji indicators
- Log file locations printed for easy access

## Configuration Resolution Details

### moments-config CLI Usage

The script calls `moments-config get <key>` for each configuration value:

```bash
cfg_get() {
	local key="$1"
	"${CONFIG_CLI}" get "${key}" --repo-root "${REPO_ROOT}" 2>/dev/null || true
}
```

**Supported keys**:
- `platform.mento-service-public-base-url` → `VITE_BACKEND_URL`
- `platform.mento-service-identity-base-url` → `VITE_IDENTITY_BACKEND_URL`
- `integrations.stytch.stytch-public-token` → `VITE_STYTCH_PUBLIC_TOKEN`
- `integrations.stytch.stytch-hosted-url` → `VITE_STYTCH_HOSTED_URL`

**Key resolution flow**:
1. `moments-config get` reads YAML config files
2. Resolves dot-notation path through config hierarchy
3. Applies environment variable overrides (MOMENTS_* prefix)
4. Returns value or empty string

**Related Code**:
- `moments/backend/cmd/moments-config/get_cmd.go` lines 11-82
- `moments/backend/cmd/moments-config/bootstrap_config.go` lines 102-178
- `moments/backend/pkg/appconfig/` for config loading

### Environment Variable Propagation

VITE_* variables are exported in script's environment:
- Available to `pnpm run dev` process
- Vite reads them via `loadEnv()` function
- Frontend code accesses via `import.meta.env.VITE_*`

**Related Code**:
- `moments/web/vite.config.mts` line 34: `const env = loadEnv(mode, process.cwd(), '')`
- Vite only exposes variables prefixed with `VITE_` to frontend

## Port Management Details

### Port Checking Implementation

```bash
check_port() {
	lsof -Pi :"$1" -sTCP:LISTEN -t >/dev/null 2>&1
}
```

**lsof flags**:
- `-Pi :PORT` - Check IPv4/IPv6 port
- `-sTCP:LISTEN` - Only TCP sockets in LISTEN state
- `-t` - Print only PIDs (used for exit code, output suppressed)

**Return values**:
- Exit code 0: Port is in use
- Exit code non-zero: Port is free

### Port Conflict Resolution

```bash
kill -9 $(lsof -Pi :"${port}" -sTCP:LISTEN -t) 2>/dev/null || true
```

**Process**:
1. `lsof` finds PID of process using port
2. `kill -9` sends SIGKILL (cannot be caught)
3. Errors suppressed (process may exit between check and kill)
4. `|| true` prevents script exit on error

**Safety considerations**:
- SIGKILL is forceful (no graceful shutdown)
- Acceptable for dev environment
- May cause data loss if process has unsaved state

## Backend Startup Details

### make run Execution

```makefile
run: bootstrap
	@echo "[make] Starting $(BIN) on :$(PORT)"
	@GIT_COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "dev"); \
	BUILD_TIME=$$(date -u +"%Y-%m-%dT%H:%M:%SZ"); \
	PORT='$(PORT)' \
	go run -ldflags "-X main.gitCommit=$$GIT_COMMIT -X main.buildTime=$$BUILD_TIME -X github.com/mento/moments/backend/pkg/version.GitCommit=$$GIT_COMMIT -X github.com/mento/moments/backend/pkg/version.BuildTime=$$BUILD_TIME" $(CMD_DIR) serve
```

**Steps**:
1. Runs `make bootstrap` (dependency)
2. Gets git commit hash (or "dev")
3. Gets build timestamp
4. Sets PORT environment variable
5. Runs `go run` with ldflags for version info

**Related Code**: `moments/backend/Makefile` lines 34-39

### make bootstrap Execution

```makefile
bootstrap:
	@echo "[make] Running idempotent bootstrap (DB + keys)..."
	@echo "[make] Ensuring config CLI is built..."
	@go build -o $(DIST_DIR)/$(BIN_CFG) $(CMD_DIR_CFG)
	@bash scripts/bootstrap-startup.sh
```

**Steps**:
1. Builds `moments-config` CLI
2. Runs `bootstrap-startup.sh` script

**Related Code**: `moments/backend/Makefile` lines 28-32

### bootstrap-startup.sh Execution

The bootstrap script (detailed in `moments/backend/scripts/bootstrap-startup.sh`):

1. **Resolves repo root** - Finds moments/ directory
2. **Loads configuration** - Uses moments-config CLI to get DB/keys config
3. **Waits for Postgres** - Checks database connectivity
4. **Ensures database exists** - Creates database if missing
5. **Runs migrations** - Executes `go run ./cmd/moments-server migrate up`
6. **Generates JWT keys** - Creates RSA key pair if missing

**Related Code**: `moments/backend/scripts/bootstrap-startup.sh` lines 226-238

### moments-server serve Execution

The server (detailed in `moments/backend/cmd/moments-server/serve.go`):

1. **Initializes appconfig** - Loads YAML config files
2. **Connects to database** - Via Ent ORM
3. **Initializes Redis** - For caching/sessions
4. **Registers routes**:
   - Identity routes (`/api/v1/auth/*`)
   - RPC routes (`/rpc/v1/*`)
   - Webchat routes
   - Health endpoint (`/rpc/v1/health`)
   - SPA handler (serves frontend)
5. **Starts HTTP server** - Listens on configured port

**Related Code**: `moments/backend/cmd/moments-server/serve.go` lines 68-406

## Frontend Startup Details

### pnpm run dev Execution

```json
"dev": "pnpm generate-version && vite"
```

**Steps**:
1. Runs `pnpm generate-version` (creates version metadata)
2. Runs `vite` (starts dev server)

**Related Code**: `moments/web/package.json` line 7

### Vite Dev Server Configuration

Vite reads configuration from `moments/web/vite.config.mts`:

1. **Loads environment variables** - Via `loadEnv()`
2. **Sets up proxy** - Routes API calls to backend
3. **Configures plugins** - React, SPA fallback
4. **Starts dev server** - On configured port (default 5173)

**Proxy routes** (from `vite.config.mts`):
- `/api/v1/*` → Identity backend
- `/rpc/v1/*` → Backend (with WebSocket support)
- `/api/v1/oauth` → Identity backend
- `/api/v1/connections` → Identity backend
- `/api/v1/google` → Identity backend
- `/config.js` → Identity backend (runtime config)

**Related Code**: `moments/web/vite.config.mts` lines 69-103

## Health Checking Limitations

The script only checks if ports are listening, not if services are healthy:

```bash
if check_port "${backend_port}"; then
	backend_status="✅ RUNNING"
fi
```

**What this checks**:
- Port is bound and listening
- Process is running

**What this doesn't check**:
- HTTP endpoints responding (e.g., `/rpc/v1/health`)
- Database connectivity
- Service initialization complete
- No startup errors

**Improvement opportunity**: Could make HTTP request to health endpoint to verify actual readiness.

## Security Considerations

1. **Stytch token masking** - Token value masked in output (`***`)
2. **Log file permissions** - Log files created with default permissions (may contain sensitive data)
3. **Port conflict resolution** - Uses `kill -9` which is forceful but acceptable for dev
4. **Configuration exposure** - Config values printed to terminal (may contain sensitive URLs)

## Performance Considerations

1. **pnpm install** - Uses `--prefer-offline` to speed up installs
2. **Port checking** - Checks every second (could be optimized with exponential backoff)
3. **Process startup** - 30 second timeout may be too short for slow systems
4. **Log file I/O** - Backend logs all output to file (may impact performance)

## Maintenance and Enhancement Opportunities

### Potential Improvements

1. **Health endpoint checking** - Verify services are actually ready, not just ports listening
2. **Better error messages** - More specific errors for common failure modes
3. **Configurable timeouts** - Allow override of 30 second startup timeout
4. **Process management** - Actually kill processes in cleanup handler
5. **Log rotation** - Prevent log files from growing indefinitely
6. **Parallel startup** - Start backend and frontend in parallel (already done, but could optimize)
7. **Status dashboard** - Real-time status updates during startup
8. **Dependency checking** - Verify all required tools before starting

### Known Issues

1. **Backend URL default** - Defaults to port 8082 but backend runs on 8083
2. **wait -n check** - Uses `help wait` which may not work on all systems
3. **Cleanup handler** - Doesn't actually kill background processes
4. **Health checking** - Only checks ports, not actual service health

## Conclusion

`startdev.sh` is a well-structured orchestration script that handles the complexity of starting a full-stack development environment. It manages configuration resolution, dependency preparation, port conflicts, process lifecycle, and provides useful status output. While there are opportunities for improvement (health checking, error handling, cleanup), the script serves its purpose effectively for daily development workflows.

The script demonstrates good practices:
- Idempotent operations (safe to run repeatedly)
- Clear error messages
- Structured logging
- Automatic conflict resolution
- User-friendly status output

Future enhancements should focus on:
- Actual health checking (not just port listening)
- Better process management in cleanup
- More robust error handling
- Configurable timeouts and options
