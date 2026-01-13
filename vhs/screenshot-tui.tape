# devctl TUI screenshots
# Captures static PNG screenshots for README documentation

# No GIF output - we only want screenshots
# Output "screenshot-tui.gif"

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 800
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms

# Start fresh - ensure clean state
Hide
Type "cd /tmp/devctl-demo-repo && devctl down 2>/dev/null; clear" Enter
Sleep 500ms
Show

# Start the TUI
Type "devctl tui --alt-screen=false" Enter
Sleep 3s

# Start services to show real data
Type "u"
Sleep 5s

# Screenshot the dashboard view
Screenshot ../docs/screenshots/devctl-tui-dashboard.png
Sleep 500ms

# Switch to Pipeline view (Tab twice)
Type "	"
Sleep 500ms
Type "	"
Sleep 1s

# Screenshot the pipeline view
Screenshot ../docs/screenshots/devctl-tui-pipeline.png
Sleep 500ms

# Switch to Plugins view (Tab once more)
Type "	"
Sleep 1s

# Screenshot the plugins view
Screenshot ../docs/screenshots/devctl-tui-plugins.png
Sleep 500ms

# Clean up - stop services and quit
Type "d"
Sleep 500ms
Type "y"
Sleep 2s
Type "q"
Sleep 500ms

