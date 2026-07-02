# 🪞 dirmirror

A lightweight and fast directory synchronization CLI tool.

`dirmirror` monitors a source directory and synchronizes any changes (creations, updates, deletions) to a destination directory in real-time. It's written in Go, making it extremely fast and cross-platform.

## Features
- **Real-time Synchronization**: Polls directories and syncs changes automatically.
- **Cross-platform**: Runs natively on Windows, macOS, and Linux.
- **Customizable Ignore List**: Easily skip specific files or directories (e.g. `.DS_Store`, `node_modules`).
- **Easy Installation**: Available via `npx`, `choco`, `brew`, or native Go tools.

## Installation

### Using npx (Node.js)
The easiest way for JavaScript developers to use it without installing Go:
```bash
npx dirmirror --src ./source --dst ./destination
```

### Using Homebrew (macOS / Linux)
```bash
brew tap SHNakajima/dirmirror
brew install dirmirror
```

### Using Chocolatey (Windows)
```bash
choco install dirmirror
```

### Using Go (Developers)
```bash
go install github.com/SHNakajima/dirmirror@latest
```

## Usage

```bash
dirmirror --src <source_dir> --dst <dest_dir> [options]
```

### Options

| Flag | Description | Default |
|------|-------------|---------|
| `--src` | The source directory to monitor and sync from | (Required) |
| `--dst` | The destination directory to sync changes to | (Required) |
| `--interval`| How often to poll for changes | `2s` |
| `--ignore` | Comma-separated list of file or folder names to ignore | `.DS_Store,desktop.ini,Thumbs.db` |

### Example

Sync `my-project` to a backup folder, checking every 5 seconds, ignoring `.git` and `node_modules`:

```bash
dirmirror --src ./my-project --dst /backup/my-project --interval 5s --ignore .git,node_modules
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
