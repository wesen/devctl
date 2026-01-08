# devctl TUI demo
# Shows the interactive terminal UI: navigation, starting services, viewing logs

Output vhs/02-tui-demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 1600
Set Height 1000
Set Padding 20
Set Theme "Dracula"

# Start in a demo repo
Type "cd /tmp/devctl-demo-repo" Enter
Sleep 1s

Type "# Start the TUI" Enter
Sleep 1s
Type "devctl tui" Enter
Sleep 2s

# Show the help overlay
Type "?"
Sleep 3s
Type "?"
Sleep 1s

# Start services
Type "u"
Sleep 1s
Type "y"
Sleep 5s

# Navigate through services
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Type "k"
Sleep 1s

# Open service logs
Type "l"
Sleep 3s

# Toggle stdout/stderr
Tab
Sleep 2s

# Toggle follow mode
Type "f"
Sleep 2s

# Go back to dashboard
Escape
Sleep 1s

# Switch to Events view
Tab
Sleep 3s

# Switch to Pipeline view
Tab
Sleep 3s

# Switch to Plugins view
Tab
Sleep 3s

# Back to Dashboard
Tab
Sleep 2s

# Stop services
Type "d"
Sleep 1s
Type "y"
Sleep 2s

# Quit
Type "q"
Sleep 1s

