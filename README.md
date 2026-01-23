## faro

`faro` is a unified dependency management utility for Go, Node.js, and Python. Run it in a project root to see which dependencies can be upgraded, choose the ones you want interactively, and let it update your lockfiles automatically.

![faro preview](images/faro-preview.png)

## Highlights

- **Multi-language support**: Works with Go, Node.js (npm, yarn, pnpm), and Python (pip, poetry, uv).
- **Interactive UI**: Bubble Tea-powered terminal interface for selective upgrades (`-i`).
- **Safety checks**: Cooldown window to skip freshly published versions (`--cooldown 14`).
- **Script-friendly**: JSON output or custom line formatting for CI/CD pipelines.
- **Vulnerability scanning**: Check for security advisories via OSV integration (`-v`).

## Supported Package Managers

| Ecosystem | Detected via | Notes |
| :--- | :--- | :--- |
| **Go** | `go.mod` | Uses `go list` and `go get` |
| **npm** | `package-lock.json` | Uses `npm outdated` and `npm install` |
| **Yarn** | `yarn.lock` | Uses `yarn outdated` and `yarn add` |
| **pnpm** | `pnpm-lock.yaml` | Uses `pnpm outdated` and `pnpm update` |
| **Pip** | `requirements.txt` | Uses generic PyPI scanning |
| **Poetry** | `poetry.lock` | Uses `poetry show` and `poetry add` |
| **uv** | `uv.lock` | Uses `uv` commands |

## Install

```bash
go install github.com/pragmaticivan/faro/cmd/faro@latest
```

From source:

```bash
git clone https://github.com/pragmaticivan/faro.git
cd faro
go build -o faro ./cmd/faro
```

## Quick start

| Task | Command | Notes |
| --- | --- | --- |
| Dry run (recommended) | `faro` | Lists updates for the detected manager |
| Upgrade everything | `faro -u` | Applies all updates to config/lockfiles |
| Interactive picker | `faro -i` | Use space to select, enter to update |
| Check vulnerabilities | `faro -v` | Shows vulnerability counts |
| Specific manager | `faro --manager npm` | Override auto-detection |
| Filter packages | `faro --filter react` | Regex filter for package names |
| Include transitive | `faro --all` | Adds indirect/transitive dependencies |

### Output formats

```bash
# Pipe-friendly
faro --format lines

# Group by category (e.g. dev vs prod) and show publish dates
faro --format group,time
```

## How it works

1. `faro` **auto-detects** your package manager by looking for lockfiles (e.g., `go.mod`, `package-lock.json`, `poetry.lock`).
2. It **scans** for updates using the native tool's CLI (e.g., `npm outdated --json`) or direct registry queries.
3. When upgrading, it runs the native installation command (e.g., `go get`, `npm install`, `poetry add`) to ensure lockfiles remain consistent.

### Vulnerability scanning

When using `-v` / `--vulnerabilities`, `faro` queries the [OSV (Open Source Vulnerabilities) API](https://osv.dev) to check for known security issues.

Example output:
```
gopkg.in/yaml.v3   v3.0.0  →  v3.0.1 [H (1)] → ✓ (fixes 1)
```
This indicates the current version has 1 HIGH severity vulnerability that will be fixed by upgrading.

## Development

```bash
go test ./...
```

## License

Licensed under the Apache License 2.0. See [LICENSE](LICENSE).

## Inspiration

* `npm-check-updates`
