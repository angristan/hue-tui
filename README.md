# hue-tui

[![CI](https://github.com/angristan/hue-tui/actions/workflows/ci.yml/badge.svg)](https://github.com/angristan/hue-tui/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/angristan/hue-tui)](https://go.dev/)
[![License](https://img.shields.io/github/license/angristan/hue-tui)](LICENSE)

A terminal-based Philips Hue control application written in Go.

![demo](demo/demo.gif)

## Features

- **Bridge Discovery**: Automatic discovery via mDNS and Philips Hue cloud
- **Bridge Pairing**: Easy link button pairing flow
- **Light Control**: Toggle, brightness, color temperature
- **Room Grouping**: Lights organized by room with group controls
- **Scene Activation**: Browse and activate scenes
- **Real-time Updates**: Server-sent events for live state updates
- **Search**: Filter lights by name
- **Keyboard-driven**: Full vim-style navigation

## Installation

### From releases

Download the latest release from the [releases page](https://github.com/angristan/hue-tui/releases).

### Using Go

```bash
go install github.com/angristan/hue-tui/cmd/hue@latest
```

### From source

```bash
git clone https://github.com/angristan/hue-tui
cd hue-tui
make build
./hue
```

## Usage

```bash
hue
```

On first run, the app will:

1. Search for Hue bridges on your network
2. Prompt you to press the link button on your bridge
3. Save the connection credentials for future use

## Keybindings

### Navigation

| Key       | Action              |
| --------- | ------------------- |
| `j` / `↓` | Move down           |
| `k` / `↑` | Move up             |
| `h` / `←` | Decrease brightness |
| `l` / `→` | Increase brightness |

### Light Control

| Key     | Action                   |
| ------- | ------------------------ |
| `Space` | Toggle light on/off      |
| `0`     | Set brightness to 100%   |
| `1-9`   | Set brightness to 10-90% |
| `w`     | Warmer color temperature |
| `c`     | Cooler color temperature |

### Room Control

| Key | Action                      |
| --- | --------------------------- |
| `a` | Turn all lights in room on  |
| `x` | Turn all lights in room off |

### Other

| Key   | Action            |
| ----- | ----------------- |
| `s`   | Open scenes modal |
| `/`   | Search lights     |
| `Tab` | Toggle side panel |
| `r`   | Refresh           |
| `q`   | Quit              |

## Configuration

Configuration is stored in `~/.config/hue-cli/config.json`:

```json
{
  "bridges": [
    {
      "host": "192.168.1.100",
      "username": "<app-key>",
      "bridge_id": "001788FFFE123456"
    }
  ],
  "last_bridge_id": "001788FFFE123456"
}
```

## Requirements

- Philips Hue Bridge (v2 API)
- Network access to the bridge

## Tech Stack

| Component      | Library                                                   |
| -------------- | --------------------------------------------------------- |
| TUI Framework  | [Bubble Tea](https://github.com/charmbracelet/bubbletea)  |
| Styling        | [Lip Gloss](https://github.com/charmbracelet/lipgloss)    |
| mDNS Discovery | [hashicorp/mdns](https://github.com/hashicorp/mdns)       |
| WebSocket      | [gorilla/websocket](https://github.com/gorilla/websocket) |

## Project Structure

```
hue-tui/
├── cmd/hue/              Entry point
└── internal/
    ├── api/              Hue Bridge V2 API client
    │   ├── client.go     HTTP client
    │   ├── discovery.go  mDNS + cloud discovery
    │   ├── events.go     Server-sent events
    │   └── pairing.go    Link button pairing
    ├── config/           Configuration management
    ├── models/           Data models (Light, Room, Scene, Color)
    └── tui/              Terminal UI
        ├── screens/      Setup, Main, Scenes screens
        ├── components/   Reusable UI components
        ├── styles/       Color theme
        └── messages/     Cross-screen messages
```

## Development

```bash
# Build
make build

# Run
make run

# Test
make test

# Lint (requires golangci-lint)
make lint

# Format
make fmt
```

## License

MIT
