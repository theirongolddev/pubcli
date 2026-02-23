# Agent CLI Notes

The `pubcli` CLI is intentionally tolerant of minor syntax mistakes when intent is clear.

Accepted flexible forms include:
- `-zip 33101` -> interpreted as `--zip 33101`
- `zip=33101` -> interpreted as `--zip=33101`
- `--ziip 33101` -> interpreted as `--zip 33101`
- `categoriess` -> interpreted as `categories`

The CLI prints a `note:` line when it auto-corrects input. Use canonical syntax in future commands:
- `pubcli --zip 33101`
- `pubcli --store 1425 --bogo`
- `pubcli categories --zip 33101`
- `pubcli stores --zip 33101 --json`

When intent is unclear, errors include a direct explanation and relevant examples.
