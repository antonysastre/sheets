# she

A simple command-line cheat sheet manager.

## Installation

### Pre-built binary (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/antonysastre/sheets/main/scripts/install.sh | bash
```

Or download from [Releases](https://github.com/antonysastre/sheets/releases).

### From source

```bash
go install github.com/antonysastre/sheets/cmd/she@latest
```

## Usage

```
she list          List all cheat sheets
she <tool>       View cheat sheet
she edit <tool>  Edit cheat sheet (creates if missing)
she new <tool>   Create new cheat sheet
she help         Show help
```

## Sheet Format

Each line should use the format:
```
command > description
```

Lines starting with `//` are ignored.

## Configuration

Set your preferred editor:
```bash
export EDITOR=vim  # or VISUAL
```

Sheets are stored in `~/.sheets/`.