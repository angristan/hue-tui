# VHS Demo for hue-tui
# Run with: vhs demo/demo.tape

Output demo/demo.gif

Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set Padding 20
Set TypingSpeed 50ms
Set Shell "bash"

# Start the app
Type "./hue --demo"
Enter
Sleep 2500ms

# Navigate to a light
Down
Sleep 300ms

# Toggle light off/on
Space
Sleep 600ms
Space
Sleep 600ms

# Brightness up/down
Type "l"
Sleep 150ms
Type "l"
Sleep 150ms
Type "l"
Sleep 400ms
Type "h"
Sleep 150ms
Type "h"
Sleep 600ms

# Color temperature warm/cold
Type "w"
Sleep 300ms
Type "w"
Sleep 400ms
Type "c"
Sleep 300ms
Type "c"
Sleep 600ms

# Navigate to color light
Down
Sleep 300ms
Down
Sleep 400ms

# Change hue color
Type "e"
Sleep 200ms
Type "e"
Sleep 200ms
Type "e"
Sleep 200ms
Type "e"
Sleep 600ms

# Go to room header
Up
Sleep 200ms
Up
Sleep 200ms
Up
Sleep 400ms

# Toggle room
Space
Sleep 600ms
Space
Sleep 600ms

# Room brightness
Type "l"
Sleep 150ms
Type "l"
Sleep 150ms
Type "l"
Sleep 600ms

# Open scenes and select one
Type "s"
Sleep 800ms
Down
Sleep 250ms
Enter
Sleep 1200ms

# Search/filter
Type "/"
Sleep 300ms
Type "floor"
Sleep 600ms
Enter
Sleep 500ms

# Adjust brightness on filtered result
Type "5"
Sleep 800ms

# Clear and quit
Escape
Sleep 400ms
Type "q"
