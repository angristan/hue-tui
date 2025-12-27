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

# Navigate to Living Room (rooms sorted: Bedroom, Kitchen, Living Room, Office)
# Bedroom(0), BedsideL(1), BedsideR(2), CeilingL(3), Kitchen(4), MainL(5), UnderCab(6), LivingRoom(7)
# From Bedroom header: Down 7 times to reach Living Room header
Down
Sleep 150ms
Down
Sleep 150ms
Down
Sleep 150ms
Down
Sleep 150ms
Down
Sleep 150ms
Down
Sleep 150ms
Down
Sleep 400ms

# Now on Living Room header - navigate to Ceiling Light
# LivingRoom(7) -> AccentStrip(8) -> CeilingLight(9)
Down
Sleep 200ms
Down
Sleep 400ms

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

# Color temperature warm/cold (Ceiling Light supports color temp)
Type "w"
Sleep 400ms
Type "w"
Sleep 400ms
Type "c"
Sleep 400ms
Type "c"
Sleep 600ms

# Navigate to TV Bias Light (Floor Lamp, then TV Bias)
Down
Sleep 300ms
Down
Sleep 500ms

# Change hue color - cycle through colors (TV Bias Light is blue, supports color)
Type "]"
Sleep 400ms
Type "]"
Sleep 400ms
Type "]"
Sleep 400ms
Type "]"
Sleep 400ms
Type "]"
Sleep 400ms
Type "]"
Sleep 400ms
Type "]"
Sleep 400ms
Type "]"
Sleep 600ms

# Go back to Living Room header
Up
Sleep 200ms
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
