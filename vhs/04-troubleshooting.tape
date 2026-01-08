# devctl troubleshooting demo
# Shows common errors and how to debug them

Output vhs/04-troubleshooting.gif

Set Shell "bash"
Set FontSize 14
Set Width 1400
Set Height 900
Set Padding 20
Set Theme "Dracula"

Type "cd /tmp/devctl-debug-demo" Enter
Sleep 1s

# Scenario 1: No config file
Type "# What happens if there's no .devctl.yaml?" Enter
Sleep 1s
Type "devctl plan" Enter
Sleep 3s

Type@500ms "" Enter
Type "# The error is actionable: no plugins configured" Enter
Sleep 2s

# Scenario 2: Plugin with stdout contamination
Type@500ms "" Enter
Type "# Create a plugin that prints to stdout (wrong!)" Enter
Sleep 1s
Type "cat > bad-plugin.py << 'EOF'" Enter
Type@100ms "#!/usr/bin/env python3" Enter
Type@100ms "import json, sys" Enter
Type@100ms "print('Starting up...')  # WRONG: this breaks the protocol" Enter
Type@100ms "EOF" Enter
Sleep 1s

Type "cat > .devctl.yaml << 'EOF'" Enter
Type@100ms "plugins:" Enter
Type@100ms "  - id: bad" Enter
Type@100ms "    path: python3" Enter
Type@100ms "    args: ['./bad-plugin.py']" Enter
Type@100ms "EOF" Enter
Sleep 1s

Type "chmod +x bad-plugin.py" Enter
Sleep 500ms

Type@500ms "" Enter
Type "# Try to load it" Enter
Sleep 1s
Type "devctl plugins list 2>&1 | head -10" Enter
Sleep 3s

Type@500ms "" Enter
Type "# The error tells us exactly what's wrong: stdout contamination" Enter
Sleep 3s

# Scenario 3: Fix the plugin
Type@500ms "" Enter
Type "# Fix it by using stderr for logs" Enter
Sleep 1s
Type "cat > good-plugin.py << 'EOF'" Enter
Type@100ms "#!/usr/bin/env python3" Enter
Type@100ms "import json, sys" Enter
Type@100ms "print('Starting up...', file=sys.stderr)  # RIGHT: stderr for logs" Enter
Type@100ms "sys.stdout.write(json.dumps({" Enter
Type@100ms "    'type': 'handshake', 'protocol_version': 'v2'," Enter
Type@100ms "    'plugin_name': 'good', 'capabilities': {'ops': []}" Enter
Type@100ms "}) + '\\n')" Enter
Type@100ms "sys.stdout.flush()" Enter
Type@100ms "for line in sys.stdin: pass" Enter
Type@100ms "EOF" Enter
Sleep 1s

Type "chmod +x good-plugin.py" Enter
Sleep 500ms

Type "sed -i 's/bad-plugin/good-plugin/' .devctl.yaml" Enter
Sleep 500ms

Type@500ms "" Enter
Type "# Now it works!" Enter
Sleep 1s
Type "devctl plugins list" Enter
Sleep 3s

