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
she --sync, -s [repo] Sync sheets to a git repo (pass repo to set up)
she --help, -h        Show help
she --version, -V     Print version and exit
```

## Syncing

`she --sync` keeps your sheets in step across machines through a central git
repository.

Set it up once per machine, pointing at a **dedicated** git repository — one
created just for your sheets, empty on first use, with a `main` branch:

```bash
she --sync git@github.com:you/sheets.git
```

This turns `~/.sheets` into a git repository, merges in anything already on
the remote, and pushes your local sheets up. `she` writes a small `.shetag`
marker file into the repository to identify it as a sheets repo; setup
refuses a non-empty repository that lacks this marker, so you can't
accidentally point `--sync` at an unrelated project.

From then on, just run:

```bash
she --sync
```

Each run commits your local changes, rebases onto the latest remote, and
pushes — so every machine ends up with every machine's sheets. If the *same*
sheet was edited on two machines between syncs, `she` stops and tells you
which sheet to resolve by hand; your local changes are committed and safe in
the meantime.

If two machines run `she --sync <repo>` against the same empty remote at
exactly the same time, one push wins and the other is rejected. To recover
on the losing machine, delete `~/.sheets/.git` and run `she --sync <repo>`
again — it will adopt the winner's marker. Your sheet files in `~/.sheets/`
are untouched.

## Sheet Format

Each line should use the format:
```
command > description
```

Lines starting with `//` are ignored.

## Configuration

Set your preferred editor (VISUAL takes precedence over EDITOR, per Unix convention):
```bash
export VISUAL=vim  # or EDITOR
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