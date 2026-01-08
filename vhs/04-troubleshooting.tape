# devctl troubleshooting demo
# Shows common errors and how to debug them

Output "04-troubleshooting.gif"

Set Shell "bash"
Set FontSize 15
Set Width 1300
Set Height 850
Set Padding 20
Set Theme "Dracula"
Set TypingSpeed 40ms

Hide
Type "rm -rf /tmp/devctl-debug && mkdir -p /tmp/devctl-debug && cd /tmp/devctl-debug && clear" Enter
Sleep 500ms
Show

# Scenario 1: No config file
Type "# Common error #1: No config file" Enter
Sleep 1s
Type "devctl plan 2>&1" Enter
Sleep 3s

Type "" Enter
Type "# Fix: Create .devctl.yaml with your plugin config" Enter
Sleep 2s

# Scenario 2: Plugin with stdout contamination
Type "clear" Enter
Sleep 500ms
Type "# Common error #2: Plugin prints to stdout (breaks protocol)" Enter
Sleep 1s

Type "cat > bad-plugin.py << 'EOF'" Enter
Type "#!/usr/bin/env python3" Enter
Type "print('Starting up...')  # WRONG! Breaks the JSON protocol" Enter
Type "EOF" Enter
Sleep 500ms
Type "chmod +x bad-plugin.py" Enter
Sleep 300ms

Type "cat > .devctl.yaml << 'EOF'" Enter
Type "plugins:" Enter
Type "  - id: bad" Enter
Type "    path: python3" Enter
Type "    args: ['./bad-plugin.py']" Enter
Type "EOF" Enter
Sleep 500ms

Type "" Enter
Type "devctl plugins list 2>&1 | head -5" Enter
Sleep 3s

Type "" Enter
Type "# The error says 'stdout contamination' - non-JSON output" Enter
Sleep 2s

# Fix it
Type "clear" Enter
Sleep 500ms
Type "# Fix: Use stderr for logs, stdout only for JSON" Enter
Sleep 1s

Type "cat > good-plugin.py << 'EOF'" Enter
Type "#!/usr/bin/env python3" Enter
Type "import json, sys" Enter
Type "print('Starting up...', file=sys.stderr)  # RIGHT!" Enter
Type "sys.stdout.write(json.dumps({" Enter
Type "    'type': 'handshake'," Enter
Type "    'protocol_version': 'v2'," Enter
Type "    'plugin_name': 'good'," Enter
Type "    'capabilities': {'ops': []}" Enter
Type "}) + '\\n')" Enter
Type "sys.stdout.flush()" Enter
Type "for line in sys.stdin: pass" Enter
Type "EOF" Enter
Sleep 500ms

Type "chmod +x good-plugin.py" Enter
Type "sed -i 's/bad-plugin/good-plugin/' .devctl.yaml" Enter
Sleep 500ms

Type "" Enter
Type "devctl plugins list" Enter
Sleep 3s

Type "" Enter
Type "# Success! Remember: stdout = JSON protocol, stderr = logs" Enter
Sleep 3s
