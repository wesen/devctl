# devctl plugin development workflow
# Shows creating a simple plugin, testing it, and iterating

Output vhs/03-plugin-development.gif

Set Shell "bash"
Set FontSize 13
Set Width 1600
Set Height 1000
Set Padding 20
Set Theme "Dracula"

# Start in a demo repo
Type "cd /tmp/devctl-demo-repo" Enter
Sleep 1s

# Show we're creating a plugin
Type "# Let's create a simple devctl plugin" Enter
Sleep 1s
Type "cat > devctl-plugin.py << 'EOF'" Enter
Sleep 500ms

Type@100ms "#!/usr/bin/env python3" Enter
Type@100ms "import json, sys" Enter
Type@100ms "" Enter
Type@100ms "def emit(obj):" Enter
Type@100ms "    sys.stdout.write(json.dumps(obj) + '\\n')" Enter
Type@100ms "    sys.stdout.flush()" Enter
Type@100ms "" Enter
Type@100ms "emit({" Enter
Type@100ms "    'type': 'handshake'," Enter
Type@100ms "    'protocol_version': 'v2'," Enter
Type@100ms "    'plugin_name': 'demo'," Enter
Type@100ms "    'capabilities': {'ops': ['launch.plan']}" Enter
Type@100ms "})" Enter
Type@100ms "" Enter
Type@100ms "for line in sys.stdin:" Enter
Type@100ms "    if not line.strip(): continue" Enter
Type@100ms "    req = json.loads(line)" Enter
Type@100ms "    rid = req.get('request_id', '')" Enter
Type@100ms "    if req.get('op') == 'launch.plan':" Enter
Type@100ms "        emit({'type': 'response', 'request_id': rid, 'ok': True," Enter
Type@100ms "              'output': {'services': [{'name': 'demo', 'command': ['sleep', '3600']}]}})" Enter
Type@100ms "EOF" Enter
Sleep 1s

Type "chmod +x devctl-plugin.py" Enter
Sleep 500ms

# Create config
Type@500ms "" Enter
Type "# Create the devctl config" Enter
Sleep 1s
Type "cat > .devctl.yaml << 'EOF'" Enter
Type@100ms "plugins:" Enter
Type@100ms "  - id: demo" Enter
Type@100ms "    path: python3" Enter
Type@100ms "    args: ['./devctl-plugin.py']" Enter
Type@100ms "    priority: 10" Enter
Type@100ms "EOF" Enter
Sleep 1s

# Test the plugin
Type@500ms "" Enter
Type "# Test that the plugin loads" Enter
Sleep 1s
Type "devctl plugins list" Enter
Sleep 3s

# See what it would do
Type@500ms "" Enter
Type "# See the service plan" Enter
Sleep 1s
Type "devctl plan" Enter
Sleep 3s

# Actually run it
Type@500ms "" Enter
Type "# Start the service" Enter
Sleep 1s
Type "devctl up" Enter
Sleep 3s

# Check status
Type "devctl status" Enter
Sleep 2s

# Clean up
Type "devctl down" Enter
Sleep 2s

