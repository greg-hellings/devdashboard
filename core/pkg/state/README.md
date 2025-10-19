# State & Configuration Consolidation (Placeholder)

This package (`core/pkg/state`) will become the canonical location for logic that manages:
1. Loading and saving GUI application state (currently implemented in the desktop GUI module).
2. Parsing, merging, and exporting CLI-compatible configuration (currently in `pkg/config`).
3. Future profile management mechanisms (multi-profile, environment-aware/state layering).
4. Credential indirection interfaces (keyring abstraction, token redaction, secure storage fallbacks).
5. Common serialization utilities (atomic writes, migration/versioning, redaction).

## Goals

- Eliminate duplication between GUI and CLI config handling.
- Provide a cohesive set of APIs for both headless (CLI) and interactive (GUI/web/mobile) modes.
- Ensure all state mutations and persistence semantics are testable outside a GUI context.
- Support future derivatives (web, mobile) without duplicating YAML or path logic.

## Planned Structure

```
state/
  README.md               (this file)
  gui_state.go            (GUI state structs, load/save, merge, redact)
  migrate.go              (future version migration logic)
  credentials.go          (credential storage abstraction)
  export.go               (CLI YAML export helpers)
  import.go               (merge/import routines)
  validation.go           (schema & integrity checks)
```

## Migration Plan (Phases)

Phase 3:
- Move structs and functions from `gui/desktop/state.go` into `gui_state.go`.
- Introduce an interface surface:
  ```go
  type GUIStateStore interface {
      Load(path string) (*State, error)
      Save(*State, path string) error
      MergeCLIConfig(*State, cliPath string) error
      Redacted(*State) *State
  }
  ```
- Provide default implementation (`filesystemStore`).

Phase 4:
- Add credential abstraction:
  ```go
  type CredentialStore interface {
      SaveToken(provider, token string) error
      GetToken(provider string) (string, error)
  }
  ```
  Backed initially by in-memory / YAML, later OS keyring.

Phase 5:
- Migration utilities triggered when `stateVersion` increases.
- Report history feature integrated with disk layout strategy.

## Testing Strategy

- Table-driven tests for:
  - Load/Save round trips.
  - Merge conflict resolution.
  - Redaction correctness.
  - Atomic save behavior (failure simulation).
- Use temporary directories; avoid reliance on GUI artifacts.

## Design Notes

- Keep GUI-specific layout details (e.g., window geometry) in the state model but ensure they are harmless for headless consumers.
- Avoid importing Fyne or any UI framework here; this package stays pure Go.
- Exported types should remain stable—additive changes prefer optional fields.
- YAML remains the primary serialization format for human readability.

## Open Questions

- Should credential storage errors block overall state save?
- How will encrypted storage keys be derived (user passphrase vs system keyring)?
- Do we unify CLI config and GUI state into a single superset file or keep them separate with explicit export/import?

## Next Step

After this placeholder:
1. Introduce `gui_state.go` and relocate existing code.
2. Refactor desktop GUI to depend on `state` package instead of its local `state.go`.
3. Remove the duplicated implementation from the GUI module.

---
Generated placeholder to stage consolidation work. Implementation to follow in subsequent commits.
