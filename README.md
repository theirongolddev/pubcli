# pubcli

`pubcli` is a Go CLI for fetching current Publix weekly ad deals by store number or ZIP code.

## Features

- Fetch weekly ad deals for a specific store
- Resolve nearest Publix store from a ZIP code
- Filter deals by category, department, keyword, and BOGO status
- List weekly categories with deal counts
- Output data as formatted terminal text or JSON
- Generate shell completions (`bash`, `zsh`, `fish`, `powershell`)
- Tolerate minor CLI syntax mistakes when intent is clear
- Robot-mode behavior for AI agents (compact help, auto JSON when piped, structured errors, meaningful exit codes)

## Requirements

- Go `1.24.4` or newer to build from source
- Network access to:
  - `https://services.publix.com/api/v4/savings`
  - `https://services.publix.com/api/v1/storelocation`

## Installation

### Build locally

```bash
git clone https://github.com/tayloree/publix-deals.git
cd publix-deals
go build -o pubcli ./cmd/pubcli
```

### Install with `go install`

```bash
go install github.com/tayloree/publix-deals/cmd/pubcli@latest
```

## Quick Start

Find nearby stores:

```bash
pubcli stores --zip 33101
```

Fetch deals using a store number:

```bash
pubcli --store 1425
```

Fetch deals by ZIP (uses the nearest store):

```bash
pubcli --zip 33101
```

Fetch JSON output:

```bash
pubcli --zip 33101 --json
```

## Commands

### `pubcli`

Fetch weekly ad deals.

```bash
pubcli [flags]
```

### `pubcli stores`

List up to 5 nearby stores for a ZIP code.

```bash
pubcli stores --zip 33101
pubcli stores -z 32801 --json
```

### `pubcli categories`

List available categories for the current week.

```bash
pubcli categories --store 1425
pubcli categories -z 33101 --json
```

## Flags

Global flags:

- `-s, --store string` Publix store number (example: `1425`)
- `-z, --zip string` ZIP code for store lookup
- `--json` Output JSON instead of styled terminal output

Deal filtering flags (`pubcli` root command):

- `--bogo` Show only BOGO deals
- `-c, --category string` Filter by category (example: `bogo`, `meat`, `produce`)
- `-d, --department string` Filter by department (substring match, case-insensitive)
- `-q, --query string` Search title/description (case-insensitive)
- `-n, --limit int` Limit results (`0` means no limit)

## Behavior Notes

- Either `--store` or `--zip` is required for deal and category lookups.
- If only `--zip` is provided, the nearest store is selected automatically.
- When using text output and ZIP-based store resolution, the selected store is shown.
- Filtering is applied in this order: `bogo`, `category`, `department`, `query`, `limit`.
- Category matching is exact and case-insensitive.
- Department and query filters use case-insensitive substring matching.
- Running `pubcli` with no args prints compact quick-start help.
- When stdout is not a TTY (for example piping to another process), JSON output is enabled automatically unless explicitly set.

## CLI Input Tolerance

The CLI auto-corrects common input mistakes and prints a `note:` describing the normalization:

- `-zip 33101` -> `--zip 33101`
- `zip=33101` -> `--zip=33101`
- `--ziip 33101` -> `--zip 33101`
- `stores zip 33101` -> `stores --zip 33101`

Command argument tokens are preserved for command workflows like:

- `pubcli completion zsh`
- `pubcli help stores`

## JSON Output

### Deals (`pubcli ... --json`)

Array of objects with fields:

- `title` (string)
- `savings` (string)
- `description` (string)
- `department` (string)
- `categories` (string[])
- `additionalDealInfo` (string)
- `brand` (string)
- `validFrom` (string)
- `validTo` (string)
- `isBogo` (boolean)
- `imageUrl` (string)

### Stores (`pubcli stores ... --json`)

Array of objects with fields:

- `number` (string)
- `name` (string)
- `address` (string)
- `distance` (string)

### Categories (`pubcli categories ... --json`)

Object map of category name to deal count:

```json
{
  "bogo": 175,
  "meat": 88,
  "produce": 81
}
```

## Structured Errors

When command execution fails, errors include:

- `code` (example: `INVALID_ARGS`, `NOT_FOUND`, `UPSTREAM_ERROR`)
- `message`
- `suggestions` (when available)
- `exitCode`

JSON-mode errors are emitted as:

```json
{"error":{"code":"INVALID_ARGS","message":"unknown flag: --ziip","suggestions":["Try `--zip`.","pubcli --zip 33101"],"exitCode":2}}
```

## Exit Codes

- `0` success
- `1` not found
- `2` invalid arguments
- `3` upstream/network failure
- `4` internal failure

## Shell Completion

Generate completions:

```bash
pubcli completion bash
pubcli completion zsh
pubcli completion fish
pubcli completion powershell
```

## Development

Run tests:

```bash
go test ./...
```

Run without building:

```bash
go run ./cmd/pubcli --zip 33101 --limit 10
```
