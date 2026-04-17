# Blight — AI Assistant Guide

Single source of truth for AI-assistant guidance. `CLAUDE.md` and any future `.cursorrules` / Copilot files are thin pointers to this file — do not duplicate content elsewhere.

---

## 1. Project snapshot

Blight is a Wails v2 desktop launcher. The Go backend exposes methods to a Vite + TypeScript frontend via Wails bindings. Windows is the primary target; darwin and linux are supported via build tags and must not regress.

---

## 2. Run / build / test

```sh
# Development
wails dev

# Production build
wails build

# Go tests
go test ./...

# Frontend tests
pnpm -C frontend test

# Cross-compile sanity checks (run all three before any backend PR)
GOOS=windows go build ./...
GOOS=darwin  go build ./...
GOOS=linux   go build ./...
```

---

## 3. Repo layout

| Path | Role |
|------|------|
| `app.go` | Core Wails app: config, search, commands, actions, all Wails-exported methods |
| `main.go` | Entry point; wires Wails options and starts the process |
| `app_platform_windows.go` | Windows-only platform helpers (`shellOpen`, `explorerSelect`, `runAsAdmin`, etc.) |
| `app_platform_nonwindows.go` | darwin/linux stubs for the same helpers |
| `internal/apps/` | App scanner, icon extraction, .lnk parsing (Windows), launcher helpers |
| `internal/commands/` | Calculator, clipboard history, system actions, shell proc attrs |
| `internal/debug/` | Debug logging, browser launcher for dev |
| `internal/files/` | File index, hidden-command helpers |
| `internal/hotkey/` | Global hotkey registration |
| `internal/installer/` | Self-update installer logic |
| `internal/search/` | Fuzzy matching (`fuzzy.go`), usage tracking (`usage.go`) |
| `internal/startup/` | OS startup/autorun registration |
| `internal/tray/` | System tray icon |
| `internal/updater/` | Update checker + installer dispatch |
| `frontend/src/main.ts` | Main launcher class: search input, result rendering, keyboard handling |
| `frontend/src/modules/` | Feature modules: settings, context-menu, calc-preview, filter-pills, icons, toast, etc. |
| `frontend/src/style.css` | All styles including spotlight-mode, themes, result rows |

---

## 4. Platform build-tag convention

Every OS-specific file must have a sibling covering the complement. Never add a Windows-only exported symbol without a stub for the other platforms.

Existing pairs (use these as the pattern):

| Windows | Non-Windows / Others |
|---------|----------------------|
| `app_platform_windows.go` | `app_platform_nonwindows.go` |
| `internal/apps/scanner_windows.go` | `internal/apps/scanner_nonwindows.go` |
| `internal/apps/icons.go` (`windows`) | `internal/apps/icons_nonwindows.go` |
| `internal/apps/lnk.go` (`windows`) | `internal/apps/lnk_nonwindows.go` |
| `internal/apps/launcher_windows.go` | `internal/apps/launcher_nonwindows.go` |
| `internal/commands/system.go` (`windows`) | `system_darwin.go`, `system_linux.go` |
| `internal/commands/sysprocattr_windows.go` | (no-op handled by `commands/` directly) |
| `internal/hotkey/hotkey.go` (`windows`) | `internal/hotkey/hotkey_nonwindows.go` |
| `internal/startup/startup.go` (`windows`) | `internal/startup/startup_nonwindows.go` |
| `internal/tray/tray.go` (`windows`) | `tray_darwin.go`, `tray_nonwindows.go` |
| `internal/updater/installer_windows.go` | `installer_darwin.go`, `installer_linux.go` |
| `internal/files/hiddencmd_windows.go` | `internal/files/hiddencmd_nonwindows.go` |
| `internal/debug/browser_windows.go` | `internal/debug/browser_nonwindows.go` |

---

## 5. Version source

The single source of truth is `wails.json` → `info.productVersion`.

- Do **not** bump `frontend/package.json` independently.
- Do **not** hardcode a version string in any `.go` file.
- To release: edit `wails.json` only.

---

## 6. Spotlight mode — do not remove

`launcher.spotlight-mode` is a load-bearing CSS class on the launcher element. It controls the no-query idle state (hides results container, footer, divider — see `style.css:126-135`).

Rules:
- `loadDefaultResults()` in `main.ts` **must** add `spotlight-mode` when the query is empty.
- Typing any character **must** remove `spotlight-mode` (`main.ts:519`, `main.ts:558`).
- Deleting back to empty **must** re-add `spotlight-mode`.
- The spotlight view may be **populated** with grouped home sections (Pinned, Recent Apps, Recent Commands, Clipboard) — this is additive. It must never be replaced by a plain search-results view.

---

## 7. Cross-platform rule

No feature may become Windows-exclusive. If a capability exists on Windows, it must have at least a no-op or best-effort implementation on darwin + linux.

Pattern: put Windows logic in `_windows.go`, put the stub/alternative in `_nonwindows.go`. See `app_platform_nonwindows.go` for the template.

---

## 8. Commit / PR conventions

- Short imperative subject line, e.g. `Add command mode UX` or `Bump product version from 0.4.3 to 0.4.4`.
- One PR per logical phase; do not bundle a ranking refactor with UI changes in the same PR.
- Smoke-test `wails dev` on Windows before merging any launcher change.
- Do not bypass hooks (`--no-verify`).

Reference: commits `636b9d5` and `6bcf872` reverted PR #50 for breaking launcher behavior by bundling too many changes.

---

## 9. Known pitfalls

- **Bundling refactors + UI in one PR** — the primary cause of the PR #50 revert. Ship them separately.
- **Removing spotlight mode** — it is intentional UX, not dead code.
- **Calculator preview divergence** — the frontend `calc-preview.ts` must delegate to backend `internal/commands/calculator.go`. Never compute independently in the frontend.
- **Silently disabling features on non-Windows** — always add a stub file; do not let `go build` fail on darwin/linux.
- **Version drift** — only `wails.json` is authoritative; do not chase it in other files.

---

## 10. Where AI guidance lives

This file (`AGENTS.md`) is the only place for AI-assistant guidance. All other files (`CLAUDE.md`, `.cursorrules`, `.github/copilot-instructions.md`) must be pointers to this file. PRs that duplicate guidance into multiple files should be rejected.
