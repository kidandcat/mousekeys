# MouseKeys

Control your mouse with your keyboard. A lightweight macOS utility for keyboard-based mouse control.

## Features

- **WASD Movement** - Move the mouse cursor with familiar gaming controls
- **Diagonal Movement** - Use Q, E, Z, X for diagonal directions
- **Progressive Acceleration** - Starts slow for precision, speeds up as you hold
- **Click Support** - Space for left click (hold for drag), Ctrl for right click, Shift for middle click
- **Scroll Support** - R to scroll up, F to scroll down
- **System Tray** - Shows current status with easy quit option
- **Caps Lock Toggle** - Quickly enable/disable with Caps Lock key

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/kidandcat/mousekeys.git
cd mousekeys

# Build
go build -o mousekeys .

# Run
./mousekeys
```

### Requirements

- macOS (tested on macOS 15+)
- Go 1.21+ (for building from source)
- Accessibility permissions (required for keyboard/mouse control)

## Usage

1. Run `mousekeys`
2. Grant Accessibility permissions when prompted
3. Press **Caps Lock** to toggle mouse control mode
4. Use the controls below to move and click

### Controls

| Key | Action |
|-----|--------|
| Caps Lock | Toggle mouse control on/off |
| W | Move up |
| A | Move left |
| S | Move down |
| D | Move right |
| Q | Move up-left (diagonal) |
| E | Move up-right (diagonal) |
| Z | Move down-left (diagonal) |
| X | Move down-right (diagonal) |
| Space | Left click (hold for drag) |
| Left Ctrl | Right click |
| Left Shift | Middle click |
| R | Scroll up |
| F | Scroll down |

### System Tray

The app shows an icon in your menu bar:
- ⌨️ - Mouse control is **inactive**
- 🖱️ - Mouse control is **active**

Click the icon to see status or quit the app.

## Why MouseKeys?

- **Accessibility** - Control your Mac without a mouse or trackpad
- **Precision** - Fine-grained cursor control with progressive acceleration
- **Ergonomics** - Keep your hands on the keyboard
- **Gaming-style controls** - Familiar WASD layout

## License

MIT License - see [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
