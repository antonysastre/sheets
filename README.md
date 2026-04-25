# she

A simple command-line cheat sheet manager.

## Installation

```bash
go install ./...
```

Or copy the `she` binary to your PATH.

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