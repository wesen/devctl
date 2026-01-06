---
Title: moments-config and Configuration Phase Analysis
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
Summary: "Detailed analysis of moments-config CLI, how startdev.sh uses it for configuration resolution, integration with appconfig system, and VITE environment variable derivation."
LastUpdated: 2026-01-06T13:10:20.123456789-05:00
WhatFor: "Understanding the configuration resolution flow from YAML files through appconfig to moments-config CLI to startdev.sh environment variables, enabling proper design of Go replacement."
WhenToUse: "When implementing configuration management in Go replacement or understanding how configuration flows through the system."
---

# moments-config and Configuration Phase Analysis

## Executive Summary

The configuration phase of `startdev.sh` relies on the `moments-config` CLI tool to bridge the gap between the backend's structured `appconfig` system and the shell script's need for environment variables. The `moments-config` CLI uses the same `appconfig` package that the backend server uses, ensuring consistency between development tooling and runtime configuration. This analysis documents how configuration flows from YAML files → appconfig registration → moments-config CLI → startdev.sh environment variables, with special attention to how VITE_* variables are derived for the frontend.

## moments-config CLI Overview

### Purpose and Architecture

The `moments-config` CLI (`backend/cmd/moments-config/`) is a configuration utility that provides access to configuration values computed from the same YAML files and appconfig schemas used by the backend server. It serves as a bridge between the structured Go configuration system and shell scripts or other external tools.

**Location**: `moments/backend/cmd/moments-config/`  
**Entry Point**: `main.go`  
**Commands**:
- `get KEY` - Get a single configuration value
- `bootstrap env` - Print bootstrap configuration as environment variables
- `bootstrap json` - Print bootstrap configuration as JSON

### Command Structure

```go
// moments/backend/cmd/moments-config/main.go
func main() {
    root := &cobra.Command{
        Use:   "moments-config",
        Short: "Moments configuration utility",
    }
    root.AddCommand(newBootstrapCommand(), newGetCommand())
    _ = root.Execute()
}
```

The CLI uses Cobra for command structure, maintaining consistency with other Moments CLI tools.

## Configuration Resolution Flow

### Phase 1: YAML File Loading

Configuration starts with YAML files in `moments/config/app/`:

**File Hierarchy** (low → high precedence):
1. `base.yaml` - Base configuration (always loaded)
2. `<env>.yaml` - Environment-specific overrides (e.g., `production.yaml`, `staging.yaml`)
3. `local.yaml` - Local development overrides (only in development)

**Code Reference**: `moments/backend/pkg/appconfig/config_paths.go` lines 18-42

```go
func DefaultConfigPaths(repoRoot, env string, includeLocal bool, extraOverrides ...string) []string {
    configDir := filepath.Join(repoRoot, "config", "app")
    paths := []string{
        filepath.Join(configDir, "base.yaml"),
    }
    // Add environment-specific file if provided
    // Add local.yaml if includeLocal is true
    return paths
}
```

**Example YAML Structure** (`config/app/base.yaml`):
```yaml
platform:
  mento-service-identity-base-url: ""
  mento-service-public-base-url: ""
  
integrations:
  stytch:
    stytch-public-token: ""
    stytch-hosted-url: ""
```

### Phase 2: Viper Merging and Environment Overrides

The `appconfig` package uses Viper to merge YAML files and apply environment variable overrides:

**Code Reference**: `moments/backend/pkg/appconfig/viper_merge.go` lines 14-63

```go
func NewMergedViperYAML(filesInPriority []string, envPrefix string) (*viper.Viper, error) {
    v := viper.New()
    
    // Set environment variable prefix (default: "MOMENTS")
    v.SetEnvPrefix(envPrefix)
    v.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
    v.AutomaticEnv()
    
    // Load files in order (later files override earlier ones)
    for _, f := range filesInPriority {
        if firstLoaded {
            v.MergeInConfig() // Merge subsequent files
        } else {
            v.ReadInConfig() // Read first file
        }
    }
    return v, nil
}
```

**Environment Variable Mapping**:
- YAML key: `platform.mento-service-identity-base-url`
- Environment variable: `MOMENTS_PLATFORM_MENTO_SERVICE_IDENTITY_BASE_URL`
- Dot notation (`.`) and dashes (`-`) are replaced with underscores (`_`)

**Precedence Order** (lowest → highest):
1. `base.yaml`
2. `<env>.yaml` (if environment specified)
3. `local.yaml` (if development)
4. Environment variables (`MOMENTS_*`)
5. CLI flags (if provided)

### Phase 3: Schema Registration and Type Hydration

The `appconfig` package uses a registration system where each configuration domain registers its schema:

**Code Reference**: `moments/backend/pkg/appconfig/registry.go` lines 25-49

```go
func RegisterSchema[T Validator](schema Schema) {
    registry = append(registry, registration{
        typ:    reflect.TypeOf(zero),
        slug:   schema.Slug,
        schema: &schema,
    })
}
```

**Example Registration** (`moments/backend/pkg/platform/settings.go`):
```go
func init() {
    appconfig.RegisterSchema[Settings](appconfig.Schema{
        Slug:        "platform",
        Description: "Platform-global environment and integration settings",
        ConfigPath:  []string{"platform"},
        Fields: []appconfig.Field{
            {
                Name: "mento-service-identity-base-url",
                Type: appconfig.ParamString,
                Help: "Base URL for identity service...",
            },
            {
                Name: "mento-service-public-base-url",
                Type: appconfig.ParamString,
                Help: "Public base URL for downstream tool calls...",
            },
        },
    })
}
```

**Schema Structure**:
- `Slug`: Unique identifier for the settings type
- `ConfigPath`: YAML path where values are located (e.g., `["platform"]`)
- `Fields`: List of configuration fields with types, help text, defaults

**All Registrations** (`moments/backend/pkg/appconfig/registrations/imports.go`):
The `registrations` package aggregates all schema registrations via side-effect imports:
- `bootstrapcfg` - Bootstrap tooling settings
- `platform` - Platform environment settings
- `keycfg` - Authentication keys
- `sqlcfg` - Database connection settings
- `stytchcfg` - Stytch integration
- And many more...

### Phase 4: Appconfig Initialization

When `moments-config` runs, it initializes the appconfig system:

**Code Reference**: `moments/backend/cmd/moments-config/bootstrap_config.go` lines 102-130

```go
func computeBootstrapConfig(...) (*BootstrapConfig, error) {
    // 1. Resolve repo root
    repoRoot, err := helpers.ResolveRepoRoot(repoRootFlag)
    
    // 2. Determine config paths
    configEnv := appconfig.ConfigEnv()
    includeLocal := configEnv == "" || configEnv == "development"
    configPaths := appconfig.DefaultConfigPaths(repoRoot, configEnv, includeLocal)
    
    // 3. Initialize appconfig from files
    _, err := appconfig.InitializeFromConfigFiles(envPrefix, configPaths)
    
    // 4. Access typed settings
    bootstrapSettings := appconfig.Must[bootstrapcfg.Settings]()
    keysSettings := appconfig.Must[keycfg.Settings]()
    platformSettings := appconfig.Must[platform.Settings]()
    
    // 5. Build BootstrapConfig from typed settings
    cfg := &BootstrapConfig{...}
    return cfg, nil
}
```

**Initialization Flow** (`moments/backend/pkg/appconfig/initialize.go`):
1. `InitializeFromConfigFiles()` → `NewMergedViperYAML()` (loads YAML + env)
2. `InitializeFromViper()` → `BuildLayers()` (builds Glazed parameter layers from schemas)
3. `GatherFromViperWithSchema()` (gathers values from Viper using schema paths)
4. `Parse()` (hydrates typed structs, validates, stores in global registry)

**Code Reference**: `moments/backend/pkg/appconfig/initialize.go` lines 17-36

### Phase 5: Bootstrap Config Computation

The `computeBootstrapConfig()` function extracts values from typed settings and builds a `BootstrapConfig` struct:

**Code Reference**: `moments/backend/cmd/moments-config/bootstrap_config.go` lines 132-177

```go
// Access typed configuration through appconfig
bootstrapSettings := appconfig.Must[bootstrapcfg.Settings]()
keysSettings := appconfig.Must[keycfg.Settings]()
platformSettings := appconfig.Must[platform.Settings]()

// Initialize bootstrap config
cfg := &BootstrapConfig{
    RepoRoot:           repoRoot,
    OpenSSLBin:         bootstrapSettings.OpenSSLBin,
    PrivateKeyFilename: defaultPrivateKeyFilename,
    PublicKeyFilename:  defaultPublicKeyFilename,
}

// Database URL from sqlcfg
dsn, err := sqlcfg.GetConnectionString()
cfg.DatabaseURL = dsn
cfg.DBName = extractDbNameFromURL(cfg.DatabaseURL)
cfg.DBAdminURL = deriveAdminURLFromDatabaseURL(cfg.DatabaseURL, cfg.DBName)

// Keys directory from keycfg
privPath := keysSettings.InternalJWTPrivateKeyPath
pubPath := keysSettings.InternalJWTPublicKeyPath
cfg.KeysDir = filepath.Dir(privPath)
```

**BootstrapConfig Fields**:
- `RepoRoot` - Repository root directory
- `DatabaseURL` - Full PostgreSQL connection string
- `DBName` - Database name (extracted from URL)
- `DBAdminURL` - Admin connection URL (for CREATE DATABASE)
- `KeysDir` - Directory for JWT keys
- `PrivateKeyFilename` - JWT private key filename
- `PublicKeyFilename` - JWT public key filename
- `OpenSSLBin` - Path to OpenSSL binary

### Phase 6: Output Formatting

The CLI outputs configuration in two formats:

**Environment Variables** (`bootstrap env`):
```go
func printAsEnv(cfg *BootstrapConfig) {
    lines := []string{
        "REPO_ROOT='%s'",
        "DATABASE_URL='%s'",
        "DB_NAME='%s'",
        // ... etc
    }
    // Shell-escapes values and prints as KEY='value' lines
}
```

**JSON** (`bootstrap json`):
```go
enc := json.NewEncoder(os.Stdout)
enc.SetIndent("", "  ")
enc.Encode(cfg)
```

## startdev.sh Configuration Usage

### How startdev.sh Uses moments-config

**Code Reference**: `moments/scripts/startdev.sh` lines 41-44, 56-61

```bash
cfg_get() {
    local key="$1"
    "${CONFIG_CLI}" get "${key}" --repo-root "${REPO_ROOT}" 2>/dev/null || true
}

# Resolve configuration values
backend_url="$(cfg_get 'platform.mento-service-public-base-url')"
identity_url="$(cfg_get 'platform.mento-service-identity-base-url')"
stytch_public="$(cfg_get 'integrations.stytch.stytch-public-token')"
stytch_hosted="$(cfg_get 'integrations.stytch.stytch-hosted-url')"
```

**Important Note**: The `cfg_get()` function uses dot-notation keys (e.g., `platform.mento-service-public-base-url`), but `moments-config get` only supports a limited set of hardcoded keys (`repo_root`, `database_url`, etc.). The script is calling `get` with keys that don't exist in the `get` command's switch statement!

**Actual Behavior**: These calls likely return empty strings (due to `2>/dev/null || true`), and the script falls back to defaults.

**Code Reference**: `moments/backend/cmd/moments-config/get_cmd.go` lines 31-73 shows only these keys are supported:
- `repo_root`
- `database_url`
- `db_name`
- `db_admin_url`
- `keys_dir`
- `private_key_filename`
- `public_key_filename`
- `openssl_bin`

**This is a bug/inconsistency**: The script tries to get platform/stytch config via `get`, but those keys aren't supported. The script should either:
1. Use `bootstrap env` and parse the output
2. Extend `moments-config get` to support arbitrary keys
3. Use a different approach

### Configuration Fallbacks

**Code Reference**: `moments/scripts/startdev.sh` lines 63-65

```bash
# Fallbacks for local dev
if [[ -z "${backend_url}" ]]; then backend_url="http://localhost:8082"; fi
if [[ -z "${identity_url}" ]]; then identity_url="http://localhost:8083"; fi
```

Since `cfg_get()` likely returns empty strings for unsupported keys, the script falls back to hardcoded defaults.

## VITE Environment Variable Configuration

### Purpose

Vite (the frontend build tool) only exposes environment variables prefixed with `VITE_` to the frontend code. This is a security feature to prevent accidentally exposing sensitive backend environment variables.

### How startdev.sh Sets VITE Variables

**Code Reference**: `moments/scripts/startdev.sh` lines 67-73

```bash
export VITE_BACKEND_URL="${backend_url}"
export VITE_IDENTITY_BACKEND_URL="${identity_url}"
export VITE_IDENTITY_SERVICE_URL="${identity_url}"
if [[ -n "${stytch_public}" ]]; then
    export VITE_STYTCH_PUBLIC_TOKEN="${stytch_public}"
fi
```

**Variables Exported**:
- `VITE_BACKEND_URL` - Backend API URL (from `platform.mento-service-public-base-url`)
- `VITE_IDENTITY_BACKEND_URL` - Identity service URL (from `platform.mento-service-identity-base-url`)
- `VITE_IDENTITY_SERVICE_URL` - Alias for identity backend URL
- `VITE_STYTCH_PUBLIC_TOKEN` - Stytch public token (from `integrations.stytch.stytch-public-token`)

### How Vite Uses These Variables

**Code Reference**: `moments/web/vite.config.mts` lines 34-37

```typescript
const env = loadEnv(mode, process.cwd(), '');
const backendUrl = env.VITE_IDENTITY_BACKEND_URL || env.VITE_IDENTITY_SERVICE_URL || 'http://localhost:8083'
const identityBackend = env.VITE_IDENTITY_BACKEND_URL || env.VITE_IDENTITY_SERVICE_URL || 'http://localhost:8083'
```

Vite's `loadEnv()` function loads all `VITE_*` prefixed variables from the environment and makes them available via `import.meta.env.VITE_*` in frontend code.

**Proxy Configuration** (`moments/web/vite.config.mts` lines 69-103):
The Vite dev server uses these URLs to configure proxy routes:
- `/api/v1/*` → `identityBackend`
- `/rpc/v1/*` → `backendUrl`
- `/config.js` → `identityBackend`
- etc.

### Configuration Flow for VITE Variables

```
YAML Files (config/app/base.yaml)
  ↓
platform:
  mento-service-identity-base-url: "http://localhost:8083"
  mento-service-public-base-url: "http://localhost:8082"
  ↓
appconfig.InitializeFromConfigFiles()
  ↓
platform.Settings struct hydrated
  ↓
moments-config CLI (if extended to support platform keys)
  OR
startdev.sh cfg_get() → empty (bug)
  ↓
startdev.sh fallback defaults
  ↓
export VITE_IDENTITY_BACKEND_URL="http://localhost:8083"
export VITE_BACKEND_URL="http://localhost:8082"
  ↓
Vite loadEnv() reads VITE_* variables
  ↓
Frontend code: import.meta.env.VITE_BACKEND_URL
```

## Integration with Backend appconfig System

### Shared Infrastructure

Both `moments-config` CLI and the backend server use the same `appconfig` package:

**Backend Server** (`moments/backend/cmd/moments-server/serve.go`):
```go
// Initialize appconfig from YAML files
if _, err := appconfig.InitializeFromConfigFiles(envPrefix, configPaths); err != nil {
    return err
}

// Access typed settings
serverSettings := appconfig.Must[servercfg.Settings]()
```

**moments-config CLI** (`moments/backend/cmd/moments-config/bootstrap_config.go`):
```go
// Same initialization
_, err := appconfig.InitializeFromConfigFiles(envPrefix, configPaths)

// Same typed settings access
bootstrapSettings := appconfig.Must[bootstrapcfg.Settings]()
```

### Benefits of Shared Infrastructure

1. **Consistency**: Same configuration files and schemas used everywhere
2. **Type Safety**: Typed structs prevent configuration errors
3. **Validation**: Schema validation ensures correct configuration
4. **Single Source of Truth**: YAML files are the authoritative source

### Schema Registration System

Configuration domains register their schemas via `init()` functions:

**Example**: `moments/backend/pkg/platform/settings.go`
```go
func init() {
    appconfig.RegisterSchema[Settings](appconfig.Schema{
        Slug:        "platform",
        ConfigPath:  []string{"platform"},
        Fields: []appconfig.Field{
            {Name: "mento-service-identity-base-url", Type: appconfig.ParamString},
            {Name: "mento-service-public-base-url", Type: appconfig.ParamString},
        },
    })
}
```

**Registration Aggregation**: `moments/backend/pkg/appconfig/registrations/imports.go` imports all registration packages via side-effect imports.

## Configuration Key Mapping

### YAML → Environment Variable → Go Struct Field

**Example**: `platform.mento-service-identity-base-url`

1. **YAML** (`config/app/base.yaml`):
   ```yaml
   platform:
     mento-service-identity-base-url: "http://localhost:8083"
   ```

2. **Environment Variable** (if set):
   ```bash
   MOMENTS_PLATFORM_MENTO_SERVICE_IDENTITY_BASE_URL="http://localhost:8083"
   ```

3. **Go Struct Field** (`platform.Settings`):
   ```go
   type Settings struct {
       MentoServiceIdentityBaseURL string // kebab-case → PascalCase
   }
   ```

4. **Schema Field** (`appconfig.Field`):
   ```go
   Field{
       Name: "mento-service-identity-base-url", // kebab-case
       Type: appconfig.ParamString,
   }
   ```

### Field Name Transformation

- **YAML/Schema**: kebab-case (`mento-service-identity-base-url`)
- **Environment Variable**: UPPER_SNAKE_CASE (`MOMENTS_PLATFORM_MENTO_SERVICE_IDENTITY_BASE_URL`)
- **Go Struct Field**: PascalCase (`MentoServiceIdentityBaseURL`)

The `appconfig` package handles these transformations automatically.

## Current Limitations and Issues

### Issue 1: moments-config get Doesn't Support Platform Keys

**Problem**: `startdev.sh` calls `moments-config get platform.mento-service-public-base-url`, but the `get` command only supports hardcoded bootstrap keys.

**Impact**: Script falls back to defaults, losing configuration from YAML files.

**Solution Options**:
1. Extend `moments-config get` to support arbitrary dot-notation keys
2. Use `moments-config bootstrap env` and parse output
3. Add platform-specific commands to moments-config

### Issue 2: No Direct VITE Variable Export

**Problem**: `moments-config` doesn't have a command to export VITE_* variables directly.

**Impact**: Script must manually map configuration keys to VITE_* variables.

**Solution**: Add `moments-config vite-env` command that outputs VITE_* variables.

### Issue 3: Configuration Caching

**Problem**: Each `cfg_get()` call spawns a new `moments-config` process, which re-initializes appconfig.

**Impact**: Slow (multiple process spawns) and inefficient.

**Solution**: Cache configuration values or use a single `bootstrap env` call.

## Recommendations for Go Replacement

### 1. Direct appconfig Integration

Instead of shelling out to `moments-config`, the Go replacement should use `appconfig` directly:

```go
import "github.com/mento/moments/backend/pkg/appconfig"
import _ "github.com/mento/moments/backend/pkg/appconfig/registrations"

// Initialize appconfig
_, err := appconfig.InitializeFromConfigFiles("MOMENTS", configPaths)

// Access typed settings
platformSettings := appconfig.Must[platform.Settings]()
stytchSettings := appconfig.Must[stytchcfg.Settings]()

// Build VITE environment variables
viteEnv := map[string]string{
    "VITE_BACKEND_URL": platformSettings.MentoServicePublicBaseURL,
    "VITE_IDENTITY_BACKEND_URL": platformSettings.MentoServiceIdentityBaseURL,
    "VITE_STYTCH_PUBLIC_TOKEN": stytchSettings.StytchPublicToken,
}
```

### 2. Configuration Provider Interface

Create a `ConfigProvider` interface that abstracts configuration access:

```go
type ConfigProvider interface {
    GetPlatformSettings() (*platform.Settings, error)
    GetStytchSettings() (*stytchcfg.Settings, error)
    GetViteEnv() (map[string]string, error)
    GetBootstrapConfig() (*BootstrapConfig, error)
}
```

### 3. Caching Layer

Cache configuration values to avoid repeated initialization:

```go
type CachedConfigProvider struct {
    cache     *sync.Map
    initOnce  sync.Once
    configPaths []string
}
```

### 4. Support for Arbitrary Keys

If needed, support arbitrary dot-notation key access:

```go
func (p *ConfigProvider) Get(key string) (string, error) {
    // Parse dot-notation key
    // Access via appconfig.Must[Type]()
    // Return value
}
```

## Conclusion

The configuration phase of `startdev.sh` relies on `moments-config` CLI to bridge the gap between the structured `appconfig` system and shell script needs. However, there are limitations: the `get` command doesn't support platform/stytch keys, causing the script to fall back to defaults. The Go replacement should integrate directly with `appconfig`, providing type-safe access to configuration values and proper VITE_* variable derivation.

The shared `appconfig` infrastructure ensures consistency between development tooling and runtime configuration, making it an ideal foundation for the Go replacement's configuration management.
