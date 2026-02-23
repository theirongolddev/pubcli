# Agent CLI Notes

The `pubcli` CLI is intentionally tolerant of minor syntax mistakes when intent is clear.

## Commands

| Command | Purpose | Requires |
|---------|---------|----------|
| `pubcli` | Fetch deals | `--store` or `--zip` |
| `pubcli stores` | List nearby stores | `--zip` |
| `pubcli categories` | List categories with counts | `--store` or `--zip` |
| `pubcli compare` | Rank nearby stores by deal quality | `--zip` |
| `pubcli tui` | Interactive deal browser | `--store` or `--zip`, interactive terminal |

## Input Tolerance

Accepted flexible forms include:
- `-zip 33101` -> interpreted as `--zip 33101`
- `zip=33101` -> interpreted as `--zip=33101`
- `--ziip 33101` -> interpreted as `--zip 33101`
- `categoriess` -> interpreted as `categories`

Flag aliases: `zipcode`/`postal-code` -> `--zip`, `dept` -> `--department`, `search` -> `--query`, `sortby`/`orderby` -> `--sort`, `max` -> `--limit`.

The CLI prints a `note:` line when it auto-corrects input. Use canonical syntax in future commands:
- `pubcli --zip 33101`
- `pubcli --store 1425 --bogo`
- `pubcli categories --zip 33101`
- `pubcli stores --zip 33101 --json`
- `pubcli compare --zip 33101 --category produce`
- `pubcli compare --zip 33101 --bogo --count 3 --json`

## Filtering and Sorting

Deal filter flags (`--bogo`, `--category`, `--department`, `--query`, `--sort`, `--limit`) are available on `pubcli`, `compare`, and `tui`.

Sort accepts: `relevance` (default), `savings`, `ending`. Aliases `end`, `expiry`, `expiration` map to `ending`.

Category synonyms: `veggies` -> `produce`, `chicken` -> `meat`, `bread` -> `bakery`, `cheese` -> `dairy`, `cold cuts` -> `deli`, etc.

## Auto JSON

When stdout is not a TTY, JSON output is enabled automatically. This means piping to `jq` or another process produces JSON without requiring `--json`.

## Errors

When intent is unclear, errors include a direct explanation and relevant examples. In JSON mode, errors are structured:

```json
{"error":{"code":"INVALID_ARGS","message":"...","suggestions":["..."],"exitCode":2}}
```

Exit codes: `0` success, `1` not found, `2` invalid args, `3` upstream error, `4` internal error.
