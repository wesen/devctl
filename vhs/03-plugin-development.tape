# devctl plugin development workflow
# Shows creating a simple plugin from scratch

Output "03-plugin-development.gif"

Set Shell "bash"
Set FontSize 15
Set Width 1400
Set Height 900
Set Padding 20
Set Theme "Dracula"
Set TypingSpeed 30ms

# Start in a fresh directory
Hide
Type "rm -rf /tmp/my-project && mkdir -p /tmp/my-project && cd /tmp/my-project && clear" Enter
Sleep 500ms
Show

Type "# Let's create a devctl plugin from scratch" Enter
Sleep 1s

Type "# First, create the plugin script" Enter
Sleep 500ms
Type "cat > plugin.py << 'PLUGIN'" Enter
Sleep 300ms

Type "#!/usr/bin/env python3" Enter
Type "import json, sys" Enter
Type "" Enter
Type "def emit(obj):" Enter
Type "    sys.stdout.write(json.dumps(obj) + '\\n')" Enter
Type "    sys.stdout.flush()" Enter
Type "" Enter
Type "# Handshake - tell devctl what we support" Enter
Type "emit({" Enter
Type "    'type': 'handshake'," Enter
Type "    'protocol_version': 'v2'," Enter
Type "    'plugin_name': 'myapp'," Enter
Type "    'capabilities': {'ops': ['launch.plan']}" Enter
Type "})" Enter
Type "" Enter
Type "# Handle requests" Enter
Type "for line in sys.stdin:" Enter
Type "    if not line.strip(): continue" Enter
Type "    req = json.loads(line)" Enter
Type "    rid = req.get('request_id', '')" Enter
Type "    " Enter
Type "    if req.get('op') == 'launch.plan':" Enter
Type "        emit({'type': 'response', 'request_id': rid, 'ok': True," Enter
Type "              'output': {'services': [" Enter
Type "                  {'name': 'app', 'command': ['python3', '-m', 'http.server', '8000']}" Enter
Type "              ]}})" Enter
Type "PLUGIN" Enter
Sleep 1s

Type "chmod +x plugin.py" Enter
Sleep 500ms

# Create config
Type "" Enter
Type "# Create the devctl config" Enter
Sleep 500ms
Type "cat > .devctl.yaml << 'EOF'" Enter
Type "plugins:" Enter
Type "  - id: myapp" Enter
Type "    path: python3" Enter
Type "    args: ['./plugin.py']" Enter
Type "    priority: 10" Enter
Type "EOF" Enter
Sleep 1s

# Test
Type "" Enter
Type "# Test that the plugin loads" Enter
Sleep 500ms
Type "devctl plugins list" Enter
Sleep 3s

# See what it would do
Type "" Enter
Type "# See the service it would start" Enter
Sleep 500ms
Type "devctl plan" Enter
Sleep 3s

# Run it
Type "" Enter
Type "# Start it!" Enter
Sleep 500ms
Type "devctl up" Enter
Sleep 4s

Type "devctl status" Enter
Sleep 2s

# Test it
Type "curl -s http://localhost:8000 | head -3" Enter
Sleep 2s

# Clean up
Type "devctl down" Enter
Sleep 2s

Type "" Enter
Type "# That's it! A working dev environment in minutes." Enter
Sleep 3s
