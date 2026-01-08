# devctl CLI workflow demo
# Shows the basic commands: plugins list → plan → up → status → logs → down

Output vhs/01-cli-workflow.gif

Set Shell "bash"
Set FontSize 14
Set Width 1400
Set Height 900
Set Padding 20
Set Theme "Dracula"

# Start in a demo repo
Type "cd /tmp/devctl-demo-repo" Enter
Sleep 1s

# Show the config file
Type "cat .devctl.yaml" Enter
Sleep 2s

Type "clear" Enter
Sleep 500ms

# List plugins to verify configuration
Type "# First, verify our plugin is configured and loads correctly" Enter
Sleep 1s
Type "devctl plugins list" Enter
Sleep 3s

# Show the plan
Type@500ms "" Enter
Type "# See what devctl would do (without starting anything)" Enter
Sleep 1s
Type "devctl plan" Enter
Sleep 3s

# Start everything
Type@500ms "" Enter
Type "# Start all services" Enter
Sleep 1s
Type "devctl up" Enter
Sleep 5s

# Check status
Type@500ms "" Enter
Type "# Check what's running" Enter
Sleep 1s
Type "devctl status" Enter
Sleep 3s

# Show logs for a service
Type@500ms "" Enter
Type "# View logs for the API service" Enter
Sleep 1s
Type "devctl logs --service api" Enter
Sleep 3s

# Stop everything
Type "q"
Sleep 500ms
Type@500ms "" Enter
Type "# Stop all services and clean up" Enter
Sleep 1s
Type "devctl down" Enter
Sleep 2s

# Final status to show everything stopped
Type@500ms "" Enter
Type "# Confirm everything is stopped" Enter
Sleep 1s
Type "devctl status 2>&1 || echo 'No state - everything stopped cleanly'" Enter
Sleep 3s

