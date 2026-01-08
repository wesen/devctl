#!/usr/bin/env bash
set -euo pipefail

# Generate all devctl demo GIFs using VHS

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
DEVCTL_DIR="$(dirname "$SCRIPT_DIR")"
DEMO_REPO="/tmp/devctl-demo-repo"
DEBUG_DEMO="/tmp/devctl-debug-demo"

cd "$SCRIPT_DIR"

# Check for required tools
if ! command -v vhs >/dev/null 2>&1; then
    echo "Error: vhs not found. Install with: brew install vhs"
    echo "Or see: https://github.com/charmbracelet/vhs#installation"
    exit 1
fi

echo "=== Setting up demo repositories ==="
echo

# Create the main demo repo
echo "→ Creating $DEMO_REPO..."
rm -rf "$DEMO_REPO"
mkdir -p "$DEMO_REPO"/{backend,frontend}

# Create a realistic backend service (simple Go HTTP server)
cat > "$DEMO_REPO/backend/main.go" << 'EOF'
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok"}`)
	})

	http.HandleFunc("/api/time", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"time":"%s"}`, time.Now().Format(time.RFC3339))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("GET %s", r.URL.Path)
		fmt.Fprintf(w, "Hello from the API!")
	})

	log.Printf("API server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
EOF

cat > "$DEMO_REPO/backend/go.mod" << 'EOF'
module demo/backend

go 1.21
EOF

# Create a simple frontend service (Python HTTP server serving static files)
cat > "$DEMO_REPO/frontend/index.html" << 'EOF'
<!DOCTYPE html>
<html>
<head><title>Demo App</title></head>
<body>
  <h1>Demo Frontend</h1>
  <p>API URL: <span id="api"></span></p>
  <script>
    document.getElementById('api').textContent = 
      window.VITE_API_URL || 'http://localhost:8080';
  </script>
</body>
</html>
EOF

cat > "$DEMO_REPO/frontend/server.py" << 'EOF'
#!/usr/bin/env python3
import http.server
import socketserver
import os
import sys

PORT = int(os.environ.get('PORT', 5173))

class Handler(http.server.SimpleHTTPRequestHandler):
    def log_message(self, format, *args):
        print(f"[frontend] {args[0]}", file=sys.stderr)

print(f"Frontend dev server starting on port {PORT}", file=sys.stderr)
with socketserver.TCPServer(("", PORT), Handler) as httpd:
    httpd.serve_forever()
EOF
chmod +x "$DEMO_REPO/frontend/server.py"

# Create the devctl plugin
cat > "$DEMO_REPO/devctl-plugin.py" << 'EOF'
#!/usr/bin/env python3
"""Demo devctl plugin for a backend + frontend setup."""
import json
import sys
import shutil

def emit(obj):
    sys.stdout.write(json.dumps(obj) + "\n")
    sys.stdout.flush()

def log(msg):
    sys.stderr.write(f"[plugin] {msg}\n")
    sys.stderr.flush()

# Handshake
emit({
    "type": "handshake",
    "protocol_version": "v2",
    "plugin_name": "demo",
    "capabilities": {"ops": ["config.mutate", "validate.run", "launch.plan"]},
})

log("Plugin started, waiting for requests...")

for line in sys.stdin:
    if not line.strip():
        continue
    req = json.loads(line)
    rid = req.get("request_id", "")
    op = req.get("op", "")
    ctx = req.get("ctx", {}) or {}

    log(f"Handling op: {op}")

    if op == "config.mutate":
        emit({
            "type": "response",
            "request_id": rid,
            "ok": True,
            "output": {
                "config_patch": {
                    "set": {
                        "env.API_PORT": "8080",
                        "env.FRONTEND_PORT": "5173",
                        "env.VITE_API_URL": "http://localhost:8080",
                        "services.api.port": 8080,
                        "services.web.port": 5173
                    },
                    "unset": []
                }
            }
        })

    elif op == "validate.run":
        errors = []
        warnings = []
        
        if not shutil.which("go"):
            errors.append({
                "code": "E_MISSING_TOOL",
                "message": "go not found. Install: brew install go"
            })
        
        if not shutil.which("python3"):
            errors.append({
                "code": "E_MISSING_TOOL", 
                "message": "python3 not found"
            })

        emit({
            "type": "response",
            "request_id": rid,
            "ok": True,
            "output": {
                "valid": len(errors) == 0,
                "errors": errors,
                "warnings": warnings
            }
        })

    elif op == "launch.plan":
        emit({
            "type": "response",
            "request_id": rid,
            "ok": True,
            "output": {
                "services": [
                    {
                        "name": "api",
                        "cwd": "backend",
                        "command": ["go", "run", "."],
                        "env": {"PORT": "8080"},
                        "health": {
                            "type": "http",
                            "url": "http://localhost:8080/health",
                            "timeout_ms": 30000
                        }
                    },
                    {
                        "name": "web",
                        "cwd": "frontend",
                        "command": ["python3", "server.py"],
                        "env": {
                            "PORT": "5173",
                            "VITE_API_URL": "http://localhost:8080"
                        }
                    }
                ]
            }
        })

    else:
        emit({
            "type": "response",
            "request_id": rid,
            "ok": False,
            "error": {"code": "E_UNSUPPORTED", "message": f"unsupported op: {op}"}
        })
EOF
chmod +x "$DEMO_REPO/devctl-plugin.py"

# Create .devctl.yaml
cat > "$DEMO_REPO/.devctl.yaml" << 'EOF'
plugins:
  - id: demo
    path: python3
    args: ["./devctl-plugin.py"]
    priority: 10
EOF

echo "  ✓ Created demo repo at $DEMO_REPO"

# Create the debug demo repo (for troubleshooting demo)
echo "→ Creating $DEBUG_DEMO..."
rm -rf "$DEBUG_DEMO"
mkdir -p "$DEBUG_DEMO"
echo "  ✓ Created debug repo at $DEBUG_DEMO"

# Build devctl
echo
echo "→ Building devctl..."
cd "$DEVCTL_DIR"
go build -o /tmp/devctl ./cmd/devctl
echo "  ✓ Built devctl"

# Add devctl to PATH for VHS
export PATH="/tmp:$PATH"

# Verify everything works
echo
echo "=== Verifying demo setup ==="
cd "$DEMO_REPO"
if /tmp/devctl plugins list >/dev/null 2>&1; then
    echo "  ✓ Plugin loads successfully"
else
    echo "  ✗ Plugin failed to load"
    /tmp/devctl plugins list
    exit 1
fi

# Generate GIFs
echo
echo "=== Generating demo GIFs ==="
cd "$SCRIPT_DIR"

for tape in *.tape; do
    echo
    echo "→ Processing $tape..."
    
    # Update tape to use our devctl binary
    TEMP_TAPE=$(mktemp)
    sed "s|devctl |/tmp/devctl |g" "$tape" > "$TEMP_TAPE"
    
    if vhs "$TEMP_TAPE"; then
        echo "  ✓ Generated ${tape%.tape}.gif"
    else
        echo "  ✗ Failed to generate ${tape%.tape}.gif"
    fi
    rm -f "$TEMP_TAPE"
done

# Clean up
echo
echo "=== Cleanup ==="
cd "$DEMO_REPO"
/tmp/devctl down 2>/dev/null || true
echo "  ✓ Stopped any running services"

echo
echo "=== Done! ==="
echo
echo "GIFs generated in: $SCRIPT_DIR/"
ls -lh "$SCRIPT_DIR"/*.gif 2>/dev/null || echo "(No GIFs found - check for errors above)"
echo
echo "Demo repos preserved at:"
echo "  $DEMO_REPO"
echo "  $DEBUG_DEMO"
echo
echo "To use in documentation:"
echo '  ![devctl demo](vhs/01-cli-workflow.gif)'
