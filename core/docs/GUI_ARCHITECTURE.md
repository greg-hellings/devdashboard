# DevDashboard GUI & Multi-Module Architecture

Status: Draft (Design Specification)
Audience: Core maintainers & contributors planning desktop/mobile/web GUI layers.

---

## 1. Goals

1. Keep the core (CLI + library logic) lean and independent from GUI dependencies.
2. Support multiple UI frontends (Desktop, Mobile, Web) without cross-pollinating dependency trees.
3. Provide a clean API boundary between "core domain logic" and any presentation layer.
4. Enable asynchronous operations (repo fetch, dependency analysis) with observable progress suitable for UI rendering.
5. Centralize state, logging capture, and persistence for GUI clients.
6. Preserve backward compatibility for existing CLI consumers (import path stability).
7. Allow future distribution targets (macOS `.app`, Windows installer, Linux packages, mobile app stores, web build via WASM) without restructuring again.

---

## 2. High-Level Architecture Overview

```
+---------------------+         +-----------------------+
|  Desktop GUI (Fyne) |---+     |  Mobile GUI (Fyne/Gio)|  (Future)
+---------------------+   |     +-----------------------+
                          |
+---------------------+   |     +-----------------------+
|   Web GUI (WASM)    |---+---->|  HTTP/Local API Layer | (Optional intermediate)
+---------------------+         +-----------------------+
             |                               |
             +-----------+-------------------+
                         |
                 +---------------+
                 |    Core API   |
                 | (Domain Layer)|
                 +-------+-------+
                         |
             +-----------+-------------+
             |   Providers / Analyzers |
             +-----------+-------------+
                         |
                 External services
```

The GUIs depend ONLY on stable Core APIs (no direct internal package imports that would break with refactors).
The Core continues to expose:
- Config model & parsing
- Repository/provider abstractions
- Analyzer & report generation
- Logging interfaces
- (New) Application service interfaces

---

## 3. Multi-Module Workspace (Approach D)

### 3.1 Proposed Directory Layout (Incremental Refactor)

Current (simplified):
```
devdashboard/
  go.mod                (module github.com/greg-hellings/devdashboard)
  cmd/devdashboard/
  pkg/...
```

Target (phase 1 – non-breaking):
```
devdashboard/
  go.work
  core/                 (was root; move existing go.mod + code here)
    go.mod              (module github.com/greg-hellings/devdashboard)
    cmd/devdashboard/
    pkg/...
  gui/
    desktop/
      go.mod            (module github.com/greg-hellings/devdashboard/gui/desktop)
      cmd/devdashboard-gui/
      internal/ui/...
    mobile/             (future)
      go.mod
    web/                (future WASM or SPA scaffold)
      go.mod
  shared/               (optional: cross-GUI utilities, if truly generic)
```

### 3.2 Why keep the original module path inside `core/`?
To avoid breaking existing consumers (`go install github.com/greg-hellings/devdashboard/cmd/devdashboard@latest`).
We treat `core/` as a *relocated root*. The repository root adds a `go.work` to bind modules.

### 3.3 `go.work` Example

```
go 1.24

use (
  ./core
  ./gui/desktop
  # ./gui/mobile
  # ./gui/web
  # ./shared
)
```

Developers doing `go work sync` get a unified workspace; consumers outside the repo remain unaffected.

### 3.4 Migration Steps

1. Create `core/` directory; move all existing root-level Go sources + `cmd/devdashboard`.
2. Move `go.mod` → `core/go.mod`.
3. Adjust relative references in scripts/Makefile (prefix with `core/`).
4. Add `go.work` at repo root referencing modules.
5. Create `gui/desktop/go.mod` (new module).
6. Remove build tags guarding the desktop GUI main (tags are no longer needed because GUI lives in its own module).
7. Update README to point to new module path for GUI builds.

---

## 4. Desktop GUI Architecture (Fyne)

### 4.1 Functional Areas (Sidebar Navigation)

Left sidebar (vertical navigation):
1. Providers
2. Repositories
3. Dependencies (Report View)
4. Logs
5. Settings (future)
6. About (future)

### 4.2 Screens

#### Providers Screen
Purpose: Manage credentials / tokens (GitHub, GitLab).
Components:
- Provider selector (list: GitHub, GitLab)
- Form fields:
  - URL / base API endpoint (for GitLab self-hosted)
  - Personal Access Token (masked)
  - Validate button (pings basic endpoint)
- Save button → persists in secure store or config file
- Provide ephemeral in-memory fallback if storage disabled

Persistence Strategy:
- Use OS keyring (future) or encrypted local file `~/.config/devdashboard/credentials.json`
- Abstract through `CredentialStore` interface.

#### Repositories Screen
Purpose: Manage tracked repositories.
UI:
- List (table or list view): Provider | Owner | Repo | Analyzer | Paths Count | Packages Count | Status
- Buttons: Add / Edit / Remove / Refresh Single
- Add/Edit Dialog:
  - Provider (dropdown)
  - Owner
  - Name
  - Ref (branch/tag; default main)
  - Analyzer (dropdown: poetry, future maven, etc.)
  - Paths (multi-line or list-edit widget)
  - Packages (tag input / multi-entry)
- Bulk import (future): paste newline list or load config file.

Internal Model:
- `RepositoryEntry` struct (maps closely to `config.RepoConfig` + provider name)
- Stored as part of a persistent user config file (GUI-managed superset).
  GUI config separate from CLI YAML?
  Strategy:
  - Maintain GUI store JSON: includes provider creds references + repos + UI preferences.
  - Export button: generate CLI-compatible YAML.

#### Dependencies (Report View)
Purpose: Display summarized dependency versions across repositories.

Top Controls:
- Refresh (async)
- Export JSON
- Filter (search packages or repos)
- Toggle show errors panel

Main Table:
- Rows: Repositories
- Columns: Selected packages (user-defined)
- Cell Value: Resolved version (color-coded out-of-sync / missing / error)

Row Selection:
- On select → open right-side pane or modal with:
  - All detected packages (not just the tracked subset)
  - Analyzer metadata
  - Raw error if analysis failed
  - Last fetch timestamp
  - Manual re-fetch button (single repository)

Background Operation:
- Launch goroutine(s) to generate report.
- Use worker pool or concurrency limit (configurable).
- Stream progress events to UI event bus.

#### Logs Screen
Purpose: Central log viewer for runtime session.

Features:
- Scrollable buffer (ring buffer size configurable, e.g., 5,000 lines)
- Level filter (Info / Warn / Error / Debug if enabled)
- Search
- “Copy All” / “Save to File”
- “Clear” button

Implementation:
- Custom `slog.Handler` that writes to channels + ring buffer.
- UI subscribes to log events; marshals into display.

---

## 5. Shared Application Layer (Core <-> GUI Boundary)

Introduce an `app` or `services` package inside `core/`:

Interfaces (examples):
```go
type DependencyService interface {
    RunReport(ctx context.Context, repos []RepoWithProvider, opts ReportOptions) (<-chan ReportProgress, *ReportResultHandle)
}

type RepositoryService interface {
    ListRepositories(ctx context.Context) ([]RepoWithProvider, error)
    SaveRepository(ctx context.Context, repo RepoWithProvider) error
    DeleteRepository(ctx context.Context, repo RepoWithProvider) error
}

type ProviderService interface {
    ListProviders(ctx context.Context) ([]ProviderInfo, error)
    SaveProvider(ctx context.Context, provider ProviderConfigInput) error
    ValidateProvider(ctx context.Context, id string) error
}

type LogStream interface {
    Subscribe() <-chan LogEntry
}
```

Event Streaming:
- `ReportProgress` includes:
  - Repository ID
  - Phase (queued, analyzing, complete, error)
  - Timestamp
  - Optional partial dependency findings
- GUI consumes channel; updates progress bar or per-row status.

Thread Safety:
- Services encapsulate locking + caching (avoid UI-level races).
- Use a background manager struct coordinating inflight tasks with cancellation support.

---

## 6. Asynchronous Task Model

Guidelines:
- Each report run creates a cancellable context.
- Concurrency limit (e.g., `N = runtime.NumCPU()` or user-configured).
- Use a small job dispatcher: buffered channel + worker goroutines.
- Emit progress events immediately when each repository finishes or errors.
- Final aggregation builds a stable `report.Report` for table model.

Error Handling:
- Distinguish between “partial success” vs “global failure.”
- Maintain per-repository error list for the error pane.

---

## 7. State Management Pattern (GUI Layer)

Option A (Recommended): Central `AppState` struct with:
- `Providers []ProviderModel`
- `Repositories []RepositoryModel`
- `CurrentReport *ReportViewModel`
- `Logs *LogBuffer`
- `Mutex` (RW) or channel-based write serialization

Expose mutation helpers:
```go
func (s *AppState) UpdateRepositories(fn func([]RepositoryModel) []RepositoryModel)
```

Notify UI:
- Pub-sub or simple observer list (`[]chan StateDelta`).
- Keep payloads small (diff or event type + identifiers).

---

## 8. Logging Capture

Custom `slog.Handler` design:
- Wrap base handler (e.g., text handler for stderr).
- On `Handle`:
  - Serialize record into `LogEntry{Time, Level, Message, Attrs}`.
  - Append to ring buffer (e.g., slice with head index modulo capacity).
  - Non-blocking send to subscribers (drop or queue if full).

Filtering in UI:
- Apply level + substring filter client-side (no need for server-side filtering initially).

---

## 9. Persistence

Files / Locations:
- UNIX: `$XDG_CONFIG_HOME/devdashboard/`
- macOS: `$HOME/Library/Application Support/devdashboard/`
- Windows: `%AppData%/devdashboard/`

Files:
- `gui_state.json` (UI preferences, last window size, etc.)
- `repositories.json` (GUI-managed repos)
- `providers.json` (non-sensitive provider metadata)
- Credentials:
  - Use OS keyring where available; fallback to `credentials.enc` (AES-GCM with master key derived from OS keyring or user-supplied passphrase).
  - Abstract behind `CredentialStore` to allow headless testing.

Atomic Writes:
- Write to temp file + `os.Rename`.
- JSON schema versioning: add `"version": 1` root field.

Export / Import:
- Export CLI YAML (generate provider-centric structure).
- Import CLI YAML (merge strategy: prompt on conflicts or create duplicates).

---

## 10. Dependency Report Table Rendering (Desktop)

Rendering Strategy:
- Use Fyne `widget.Table` for virtualization support.
- Column 0: Repository identifier (Provider:Owner/Repo@Ref)
- Columns 1..N: Tracked packages (user-managed list).
- Cell states:
  - Normal: version string
  - Missing: “—” (dim)
  - Error: icon + tooltip
  - Divergence (future): color highlight when version deviates from baseline.

Row Detail View:
- Trigger: double-click or “Details” button.
- Modal or split pane showing:
  - All detected dependencies (scrolling list)
  - Search filter
  - Raw analyzer metadata (future)
  - Re-fetch button (single repo execution)

Performance Considerations:
- Cache table model separate from raw `report.Report`.
- Debounce UI refresh on bulk updates.

---

## 11. Web GUI (Future)

Approaches:
1. WASM + Fyne (limited controls parity).
2. WASM + custom minimal UI (HTML/DOM via syscall/js).
3. SPA (React/Vue/Svelte) + local HTTP server embedded in Go (serves API endpoints calling core).
4. gRPC-Web / JSON over HTTP bridging.

Recommendation:
- For richer UX: SPA + Core compiled as long-running service exposing HTTP API.
- Keep boundary identical to GUI service interfaces.

---

## 12. Mobile GUI (Future)

Options:
- Fyne (multi-platform; reuse much desktop logic).
- Gio (more custom, higher effort).
- Flutter (separate non-Go stack; bigger divergence).

Recommendation:
- Start with Fyne reuse; share `services` + `AppState` package.
- Introduce `platform` interface for file paths, theming differences.

---

## 13. Testing Strategy

Layers:
1. Core services: pure Go unit tests (unchanged).
2. Application service layer: table-driven tests mocking providers/analyzers.
3. GUI logic (non-visual): test state reducers and event bus.
4. Visual smoke tests (optional): golden screenshot comparison (future).
5. Concurrency tests: race detector (`go test -race`).

CI:
- Core always builds/tests.
- GUI modules tested conditionally (matrix include gui-desktop).
- Web/mobile optional jobs.

---

## 14. Incremental Implementation Plan

Phase 1:
- Restructure repo into workspace (create `go.work`, move core).
- Create `gui/desktop` module; move current prototype there; remove build tag.

Phase 2:
- Introduce `services` package in core with first `DependencyService`.
- Implement basic async reporting from desktop GUI (refresh button, progress labeling).
- Add log handler capturing.

Phase 3:
- Providers screen: persistence of provider credentials (non-secure placeholder).
- Repositories CRUD + export/import CLI YAML.

Phase 4:
- Full dependency table (tracked packages).
- Row detail modal with full package list.

Phase 5:
- Error pane + JSON export.
- Persistent UI preferences & window geometry.

Phase 6:
- CredentialStore abstraction + OS keyring integration.
- Performance tuning (batch UI updates, concurrency controls).

Phase 7:
- Prep for web/mobile modules (skeleton go.mod + placeholder README).

---

## 15. Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Module path breakage | External users blocked | Keep core module path identical (`github.com/greg-hellings/devdashboard`) |
| GUI state race conditions | Intermittent bugs | Centralized `AppState` + controlled mutation API |
| Blocking UI during long ops | Poor UX | Always run heavy tasks in goroutines + progress events |
| Credential leakage | Security | Use keyring abstraction early; never log tokens |
| Table scaling with large repo/package counts | Performance | Lazy rendering + minimal cell text formatting |
| Diverging logic between CLI & GUI | Maintenance burden | Shared services layer; CLI uses same dependency/report pipeline |

---

## 16. Open Questions

1. Should the GUI allow editing analyzer-specific settings beyond packages/paths? (Pluggable config forms)
2. Do we introduce a local gRPC layer now for future web/mobile reuse? (Deferred until web planning)
3. Multi-user or profiles support? (Probably out-of-scope early)
4. Do we want persistent caching of previous reports for diffing? (Future “History” feature)

---

## 17. Immediate Action Items

- [ ] Create `core/` directory and move existing code.
- [ ] Add `go.work`.
- [ ] Create `gui/desktop/go.mod` + migrate prototype.
- [ ] Remove `gui` build tag (module separation renders it unnecessary).
- [ ] Add initial `services/dependency` abstraction.
- [ ] Implement logging handler with in-memory ring buffer.
- [ ] Scaffold Providers & Repositories screens (data structs + empty views).
- [ ] Add README updates for multi-module workspace.

---

## 18. Appendix

### 18.1 Example `gui/desktop/go.mod`

```
module github.com/greg-hellings/devdashboard/gui/desktop

go 1.24

require (
    fyne.io/fyne/v2 v2.4.0
    github.com/greg-hellings/devdashboard v0.0.0
)
replace github.com/greg-hellings/devdashboard => ../../core
```

### 18.2 Example `go.work` After Phase 1

```
go 1.24

use (
    ./core
    ./gui/desktop
)
```

### 18.3 Ring Buffer Log Handler Sketch

```
type RingLogHandler struct {
    next    slog.Handler
    buf     []LogEntry
    cap     int
    idx     int
    subs    []chan LogEntry
    mu      sync.RWMutex
}
```

### 18.4 Report Progress Event Sketch

```
type ProgressPhase string
const (
  PhaseQueued ProgressPhase = "queued"
  PhaseRunning ProgressPhase = "running"
  PhaseComplete ProgressPhase = "complete"
  PhaseError ProgressPhase = "error"
)

type ReportProgress struct {
  RepoID   string
  Phase    ProgressPhase
  Error    error
  Started  time.Time
  Finished *time.Time
}
```

---

Prepared for future expansion; feedback welcome before executing Phase 1.
