# she

A simple command-line cheat sheet manager.

## Installation

### Pre-built binary (recommended)

```bash
curl -fsSL https://github.com/antonysastre/sheets/releases/latest/download/install.sh | bash
```

Or download from [Releases](https://github.com/antonysastre/sheets/releases).

### From source

```bash
go install github.com/antonysastre/sheets/cmd/she@latest
```

## Usage

```
she <tool>            View cheat sheet
she --list, -l        List all cheat sheets
she --edit, -e <tool> Edit cheat sheet (creates if missing)
she --new, -n <tool>  Create new cheat sheet
she --help, -h        Show help
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

## Releasing

Releases are cut by pushing a semver tag — GitHub Actions runs goreleaser
and publishes the binaries plus `install.sh` as release assets.

```bash
git tag -a v0.1.0 -m "v0.1.0"
git push origin v0.1.0
```

To dry-run goreleaser locally without publishing:

```bash
goreleaser release --snapshot --clean --skip=publish
```