# VHS Demo for hue-tui
# Run with: vhs demo.tape

Output demo.gif

Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set Padding 20

# Start the app
Type "./hue"
Enter
Sleep 2s

# Navigate down through lights
Sleep 500ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 500ms

# Toggle a light
Type " "
Sleep 1s

# Adjust brightness with number keys
Type "5"
Sleep 800ms
Type "8"
Sleep 800ms

# Navigate with vim keys
Type "k"
Sleep 300ms
Type "k"
Sleep 500ms

# Adjust brightness with h/l
Type "l"
Sleep 200ms
Type "l"
Sleep 200ms
Type "l"
Sleep 500ms

Type "h"
Sleep 200ms
Type "h"
Sleep 500ms

# Color temperature
Type "w"
Sleep 500ms
Type "w"
Sleep 500ms
Type "c"
Sleep 500ms

# Open scenes modal
Type "s"
Sleep 1s

# Navigate scenes
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms

# Select a scene
Enter
Sleep 1s

# Search for a light
Type "/"
Sleep 500ms
Type "desk"
Sleep 1s
Escape
Sleep 500ms

# Room controls - turn all on
Type "a"
Sleep 800ms

# Turn all off
Type "x"
Sleep 800ms

# Toggle side panel
Tab
Sleep 800ms
Tab
Sleep 500ms

# Quit
Type "q"
Sleep 500ms
