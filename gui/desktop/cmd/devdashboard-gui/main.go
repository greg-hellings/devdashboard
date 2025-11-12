// Package main implements the DevDashboard Desktop GUI application.
package main

// DevDashboard Desktop GUI (Phase 2 Integration)
// ----------------------------------------------
// Features implemented in this version:
//   - Persistent YAML-backed GUI state (load on startup, save on exit & mutations)
//   - Asynchronous dependency report using DependencyService (progress streamed)
//   - Auto-refresh capability honoring state.GUI.AutoRefresh settings
//   - Repository Add dialog (basic form to append repositories)
//   - Tracked Packages management (modal editor)
//   - JSON report export (similar shape to CLI JSON output)
//   - Ring-buffer log capture with level filtering
//   - Sidebar navigation (Providers, Repositories, Dependencies, Packages, Logs)
//   - Row detail modal for full dependency list per repository
//
// State Persistence:
//   Uses statepkg.LoadGUIState("") and statepkg.SaveGUIState(st, "").
//   Default path resolution is handled by DefaultStatePath().
//   State mutations trigger a debounced save.
//
// Tracked Packages:
//   If state.TrackedPackages is empty, the Dependencies table falls back to
//   displaying all packages discovered in the current report. Managing tracked
//   packages allows users to focus comparisons.
//
// Auto-Refresh:
//   If enabled in YAML state (gui.autoRefresh.enabled), a background goroutine
//   triggers a report refresh at gui.autoRefresh.intervalSeconds. Safeguards
//   prevent overlapping runs.
//
// DependencyService Progress:
//   Progress events are displayed in a minimal list below the status line.
//   Future phases can enhance with per-repository progress bars.
//
// NOTE: Credentials remain ephemeral prototypes and are not stored securely.
//       DO NOT distribute builds with real tokens stored in YAML; integrate
//       OS keyring before production use.
//
// Module Imports:
//   - Core domain logic: github.com/greg-hellings/devdashboard/core/...
//   - GUI state & helpers: github.com/greg-hellings/devdashboard/gui/desktop
//
// Build:
//   go build -o devdashboard-gui ./cmd/devdashboard-gui
//
// Run:
//   ./devdashboard-gui
//
// Future Enhancements (Phase 3+):
//   - Keyring integration for tokens
//   - Repository edit/remove dialogs
//   - JSON export refinement (error section toggles)
//   - Detailed progress with granular phases
//   - History/diff of previous reports
//   - Package search/filter toolbar
//   - Error pane separate from Logs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	fapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/greg-hellings/devdashboard/core/pkg/config"
	"github.com/greg-hellings/devdashboard/core/pkg/report"
	"github.com/greg-hellings/devdashboard/core/pkg/services"
	statepkg "github.com/greg-hellings/devdashboard/core/pkg/state"
)

// version override via -ldflags "-X main.version=..."
var version = "devdesktop-phase2"

// ----- Runtime Layer (Non-persisted fields) -----
//
// Theme handling:
// We persist the desired variant (light|dark) in rt.state.GUI.Theme and
// store the preference using the app's Preferences under key "themeVariant".
// Fyne will apply the variant automatically; we no longer use deprecated
// theme.LightTheme()/theme.DarkTheme() helpers nor a custom theme wrapper.

// Runtime encapsulates the live (non-persisted) GUI execution state,
// including the current dependency report, progress events, credential
// resolution store, and auto-refresh control channels. It is created
// once per application run and coordinates background report generation
// with UI updates.
type Runtime struct {
	mu sync.RWMutex

	state *statepkg.GUIState

	// Ephemeral report + progress
	currentReport  *report.Report
	reportRunning  bool
	progressEvents []services.ReportProgress

	// Progress indexing for quick lookup
	progressIndex map[string]services.ReportProgress

	// Dependency service
	depSvc services.DependencyService

	// Credential store (env/YAML/keyring resolution)
	credentialStore statepkg.CredentialStore

	// Auto-refresh control
	autoRefreshStopChan chan struct{}
}

// NewRuntime constructs a Runtime wrapper around a loaded GUIState,
// initializing progress bookkeeping, a dependency service instance,
// and a fallback credential store. Call this after loading persistent
// state to begin coordinating reports and UI interactions.
func NewRuntime(st *statepkg.GUIState) *Runtime {
	return &Runtime{
		state:               st,
		currentReport:       nil,
		reportRunning:       false,
		progressEvents:      []services.ReportProgress{},
		progressIndex:       map[string]services.ReportProgress{},
		depSvc:              services.NewDependencyService(nil),
		credentialStore:     statepkg.NewFallbackCredentialStore(nil, statepkg.NewInMemoryCredentialStore()),
		autoRefreshStopChan: nil,
	}
}

// LogEntry is a structured log record captured in the in-memory ring buffer for GUI display.
// It preserves timestamp, level, message, and any structured attributes emitted with the original slog record.
type LogEntry struct {
	Time    time.Time
	Level   slog.Level
	Message string
	Attrs   []slog.Attr
}

// RingLogHandler implements slog.Handler and captures recent log entries in a
// bounded ring buffer for GUI inspection while delegating to an underlying
// handler (next). It is safe for concurrent use.
type RingLogHandler struct {
	next     slog.Handler
	capacity int

	mu    sync.RWMutex
	logs  []LogEntry
	level slog.Level
}

// NewRingLogHandler constructs a RingLogHandler that records up to 'capacity' log entries at or above the provided 'level' while forwarding all records to the wrapped 'next' handler. A non-positive capacity falls back to 5000.
func NewRingLogHandler(next slog.Handler, capacity int, level slog.Level) *RingLogHandler {
	if capacity <= 0 {
		capacity = 5000
	}
	return &RingLogHandler{
		next:     next,
		capacity: capacity,
		logs:     make([]LogEntry, 0, capacity),
		level:    level,
	}
}

// Enabled reports whether a log of the given level should be processed (captured + forwarded). Only levels >= handler level are retained.
func (h *RingLogHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return lvl >= h.level && h.next.Enabled(ctx, lvl)
}

// Handle records the log entry in the ring buffer if its level meets the threshold, while always delegating to the wrapped handler.
func (h *RingLogHandler) Handle(ctx context.Context, rec slog.Record) error {
	_ = h.next.Handle(ctx, rec)

	if rec.Level < h.level {
		return nil
	}

	entry := LogEntry{
		Time:    rec.Time,
		Level:   rec.Level,
		Message: rec.Message,
	}
	rec.Attrs(func(a slog.Attr) bool {
		entry.Attrs = append(entry.Attrs, a)
		return true
	})

	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.logs) == h.capacity {
		// Drop oldest to maintain bounded size.
		copy(h.logs[0:], h.logs[1:])
		h.logs = h.logs[:h.capacity-1]
	}
	h.logs = append(h.logs, entry)
	return nil
}

// WithAttrs returns a new RingLogHandler wrapping the underlying handler augmented with the provided attributes; captured entries remain separate.
func (h *RingLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return NewRingLogHandler(h.next.WithAttrs(attrs), h.capacity, h.level)
}

// WithGroup returns a new RingLogHandler scoping subsequent attributes under the provided group name; ring buffer semantics are unchanged.
func (h *RingLogHandler) WithGroup(name string) slog.Handler {
	return NewRingLogHandler(h.next.WithGroup(name), h.capacity, h.level)
}

// Entries returns a snapshot copy of all retained log entries in FIFO order.
func (h *RingLogHandler) Entries() []LogEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	cp := make([]LogEntry, len(h.logs))
	copy(cp, h.logs)
	return cp
}

// ----- Main -----

func main() {
	app := fapp.NewWithID("devdashboard.desktop")
	state, err := statepkg.LoadGUIState("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load GUI state: %v\n", err)
		state = statepkg.NewDefaultGUIState()
	}
	runtime := NewRuntime(state)

	// Initialize theme preference based on persisted state (light|dark).
	// Store it so Fyne applies the preferred variant.
	switch strings.ToLower(state.GUI.Theme) {
	case "dark":
		app.Preferences().SetString("themeVariant", "dark")
	default:
		app.Preferences().SetString("themeVariant", "light")
		runtime.state.GUI.Theme = "light"
	}

	// Logging level mapping
	logLevel := slog.LevelInfo
	switch strings.ToLower(state.GUI.Logging.Level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}

	baseHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})
	logHandler := NewRingLogHandler(baseHandler, state.GUI.Logging.RingBufferSize, logLevel)
	slog.SetDefault(slog.New(logHandler))
	slog.Info("GUI starting", "version", version, "statePath", statepkg.DefaultGUIStatePath())

	w := app.NewWindow("DevDashboard")
	w.Resize(fyne.NewSize(float32(state.GUI.LastWindow.Width), float32(state.GUI.LastWindow.Height)))
	if state.GUI.LastWindow.Maximized {
		w.SetFullScreen(true)
	}

	// --- Serialized UI Event Dispatcher ---
	uiQueue := make(chan func(), 256)
	var uiOnce sync.Once
	go func() {
		for fn := range uiQueue {
			// Recover to prevent a single panic from stopping dispatcher
			func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("UI dispatcher panic recovered", "error", r)
					}
				}()
				fn()
			}()
		}
	}()
	enqueueUI := func(fn func()) {
		select {
		case uiQueue <- fn:
		default:
			// Fallback: drop oldest to ensure forward progress
			slog.Warn("UI queue full; dropping oldest event")
			<-uiQueue
			uiQueue <- fn
		}
	}

	root := buildUI(app, w, runtime, logHandler, enqueueUI)
	w.SetContent(root)

	// Start auto-refresh if enabled (pass dispatcher)
	startAutoRefresh(runtime, enqueueUI)

	w.SetCloseIntercept(func() {
		slog.Info("Window closing - saving state")
		saveState(runtime)
		if runtime.autoRefreshStopChan != nil {
			close(runtime.autoRefreshStopChan)
		}
		uiOnce.Do(func() { close(uiQueue) })
		app.Quit()
	})

	w.ShowAndRun()
}

// ----- Auto-Refresh -----

func startAutoRefresh(rt *Runtime, enqueueUI func(func())) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if !rt.state.GUI.AutoRefresh.Enabled || rt.state.GUI.AutoRefresh.IntervalSeconds <= 0 {
		return
	}
	if rt.autoRefreshStopChan != nil {
		return
	}
	ch := make(chan struct{})
	rt.autoRefreshStopChan = ch
	interval := time.Duration(rt.state.GUI.AutoRefresh.IntervalSeconds) * time.Second
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				rt.mu.RLock()
				running := rt.reportRunning
				rt.mu.RUnlock()
				if !running {
					slog.Info("Auto-refresh triggering report")
					fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "Auto-refresh", Content: "Refreshing dependencies"})
					enqueueUI(func() {
						runReportAsync(rt, enqueueUI, nil, nil) // status label and table updated in view if present
					})
				} else {
					slog.Debug("Skipping auto-refresh; report already running")
				}
			case <-ch:
				slog.Info("Auto-refresh stopped")
				return
			}
		}
	}()
}

// ----- UI Composition -----

type viewID string

const (
	viewProviders    viewID = "Providers"
	viewRepositories viewID = "Repositories"
	viewDependencies viewID = "Dependencies"
	viewPackages     viewID = "Packages"
	viewLogs         viewID = "Logs"
	viewHistory      viewID = "History"
)

func buildUI(app fyne.App, w fyne.Window, rt *Runtime, logHandler *RingLogHandler, enqueueUI func(func())) fyne.CanvasObject {
	dyn := container.NewStack()

	// Pre-build views
	providersView := buildProvidersView(rt, app, w)
	reposView := buildRepositoriesView(rt, app, w)
	depsView := buildDependenciesView(rt, w, enqueueUI)
	packagesView := buildPackagesView(rt, app, w)
	logsView := buildLogsView(rt, app, w, logHandler)

	historyView := buildHistoryView(rt)

	views := map[viewID]fyne.CanvasObject{
		viewProviders:    providersView,
		viewRepositories: reposView,
		viewDependencies: depsView,
		viewPackages:     packagesView,
		viewLogs:         logsView,
		viewHistory:      historyView,
	}

	// Track current view for highlighting
	currentView := viewDependencies

	sidebar := buildSidebar(app, dyn, views, rt, &currentView)

	// Initial view
	dyn.Objects = []fyne.CanvasObject{depsView}

	split := container.NewHSplit(sidebar, dyn)
	split.SetOffset(0.20)
	return split
}

func buildSidebar(app fyne.App, dyn *fyne.Container, views map[viewID]fyne.CanvasObject, rt *Runtime, currentView *viewID) fyne.CanvasObject {
	title := widget.NewLabel(fmt.Sprintf("DevDashboard %s", version))
	title.Alignment = fyne.TextAlignCenter
	title.TextStyle = fyne.TextStyle{Bold: true}

	// Map to store button references for styling updates
	buttons := make(map[viewID]*widget.Button)

	switchViewBtn := func(id viewID) *widget.Button {
		btn := widget.NewButton(string(id), func() {
			slog.Info("Switch view", "view", id)
			*currentView = id
			dyn.Objects = []fyne.CanvasObject{views[id]}
			dyn.Refresh()

			// Update button styling to highlight active view
			for viewName, button := range buttons {
				if viewName == id {
					button.Importance = widget.HighImportance
				} else {
					button.Importance = widget.MediumImportance
				}
				button.Refresh()
			}
		})

		// Set initial importance based on current view
		if id == *currentView {
			btn.Importance = widget.HighImportance
		} else {
			btn.Importance = widget.MediumImportance
		}

		buttons[id] = btn
		return btn
	}

	themeToggle := widget.NewButton("Toggle Theme", func() {
		// Toggle persisted variant and update app preference.
		if strings.ToLower(rt.state.GUI.Theme) == "dark" {
			rt.state.GUI.Theme = "light"
			app.Preferences().SetString("themeVariant", "light")
		} else {
			rt.state.GUI.Theme = "dark"
			app.Preferences().SetString("themeVariant", "dark")
		}
		saveState(rt)
	})

	return container.NewVBox(
		title,
		widget.NewSeparator(),
		switchViewBtn(viewProviders),
		switchViewBtn(viewRepositories),
		switchViewBtn(viewDependencies),
		switchViewBtn(viewPackages),
		switchViewBtn(viewLogs),
		widget.NewSeparator(),
		themeToggle,
		layout.NewSpacer(),
		widget.NewLabel("© DevDashboard"),
	)
}

// ----- Providers View -----

func buildProvidersView(rt *Runtime, _ fyne.App, _ fyne.Window) fyne.CanvasObject {
	// Token entries (prototype only)
	githubToken := widget.NewPasswordEntry()
	githubToken.SetPlaceHolder("GitHub token (optional)")
	gitlabToken := widget.NewPasswordEntry()
	gitlabToken.SetPlaceHolder("GitLab token (optional)")

	status := widget.NewLabel("Status: Idle")

	saveBtn := widget.NewButton("Save Tokens (Ephemeral)", func() {
		rt.mu.Lock()
		if rt.state.Credentials == nil {
			rt.state.Credentials = &statepkg.CredentialSnapshot{}
		}
		rt.state.Credentials.GitHubToken = githubToken.Text
		rt.state.Credentials.GitLabToken = gitlabToken.Text
		rt.mu.Unlock()
		saveState(rt)
		status.SetText("Status: Saved (in YAML; do not use in prod)")
	})

	validateBtn := widget.NewButton("Validate (Stub)", func() {
		status.SetText("Status: Validation not implemented")
	})

	return container.NewVBox(
		widget.NewLabelWithStyle("Provider Management", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewForm(
			&widget.FormItem{Text: "GitHub Token", Widget: githubToken},
			&widget.FormItem{Text: "GitLab Token", Widget: gitlabToken},
		),
		container.NewHBox(saveBtn, validateBtn),
		status,
		layout.NewSpacer(),
	)
}

// ----- Repositories View -----

func buildRepositoriesView(rt *Runtime, _ fyne.App, w fyne.Window) fyne.CanvasObject {
	repoList := widget.NewList(
		func() int {
			rt.mu.RLock()
			defer rt.mu.RUnlock()
			return len(rt.state.RepositoriesCache)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			rt.mu.RLock()
			defer rt.mu.RUnlock()
			if i >= len(rt.state.RepositoriesCache) {
				o.(*widget.Label).SetText("")
				return
			}
			r := rt.state.RepositoriesCache[i]
			o.(*widget.Label).SetText(fmt.Sprintf("%s: %s/%s@%s (%s)",
				r.Provider, r.Owner, r.Repository, r.Ref, r.Analyzer))
		},
	)
	// Contextual edit/remove when a repository row is selected.
	repoList.OnSelected = func(i widget.ListItemID) {
		rt.mu.RLock()
		if i < 0 || i >= len(rt.state.RepositoriesCache) {
			rt.mu.RUnlock()
			return
		}
		selected := rt.state.RepositoriesCache[i]
		rt.mu.RUnlock()

		editBtn := widget.NewButton("Edit", func() {
			// Build edit form pre-populated with existing values
			providerEntry := widget.NewSelect([]string{"github", "gitlab"}, nil)
			providerEntry.SetSelected(selected.Provider)

			ownerEntry := widget.NewEntry()
			ownerEntry.SetText(selected.Owner)

			repoEntry := widget.NewEntry()
			repoEntry.SetText(selected.Repository)

			refEntry := widget.NewEntry()
			refEntry.SetText(selected.Ref)

			analyzerEntry := widget.NewSelect([]string{"poetry"}, nil)
			analyzerEntry.SetSelected(selected.Analyzer)

			pathsEntry := widget.NewMultiLineEntry()
			pathsEntry.SetText(strings.Join(selected.Paths, "\n"))

			packagesEntry := widget.NewMultiLineEntry()
			packagesEntry.SetText(strings.Join(selected.Packages, "\n"))

			form := &widget.Form{
				Items: []*widget.FormItem{
					{Text: "Provider", Widget: providerEntry},
					{Text: "Owner", Widget: ownerEntry},
					{Text: "Repository", Widget: repoEntry},
					{Text: "Ref", Widget: refEntry},
					{Text: "Analyzer", Widget: analyzerEntry},
					{Text: "Paths", Widget: pathsEntry},
					{Text: "Packages", Widget: packagesEntry},
				},
				OnSubmit: func() {
					newProvider := providerEntry.Selected
					newOwner := strings.TrimSpace(ownerEntry.Text)
					newRepo := strings.TrimSpace(repoEntry.Text)
					newRef := strings.TrimSpace(refEntry.Text)
					newAnalyzer := analyzerEntry.Selected
					if newProvider == "" || newOwner == "" || newRepo == "" || newAnalyzer == "" {
						dialog.ShowError(fmt.Errorf("required fields missing"), w)
						return
					}
					newPaths := filterNonEmptyLines(pathsEntry.Text)
					newPackages := filterNonEmptyLines(packagesEntry.Text)

					// Apply changes
					rt.mu.Lock()
					// Remove old entry from its provider slice
					for pi, wrapper := range rt.state.Providers {
						updated := wrapper.Repositories[:0]
						for _, r := range wrapper.Repositories {
							if pi == selected.Provider &&
								r.Owner == selected.Owner &&
								r.Repository == selected.Repository &&
								r.Ref == selected.Ref {
								continue // drop old
							}
							updated = append(updated, r)
						}
						wrapper.Repositories = updated
						rt.state.Providers[pi] = wrapper
					}
					// Add updated entry to new provider
					wrapper := rt.state.Providers[newProvider]
					wrapper.Repositories = append(wrapper.Repositories, config.RepoConfig{
						Token:      selected.Token, // preserve token if any
						Owner:      newOwner,
						Repository: newRepo,
						Ref:        newRef,
						Paths:      newPaths,
						Packages:   newPackages,
						Analyzer:   newAnalyzer,
					})
					rt.state.Providers[newProvider] = wrapper
					rt.state.RebuildRepositoriesCache()
					rt.mu.Unlock()

					saveState(rt)
					repoList.Refresh()
					dialog.ShowInformation("Updated", "Repository updated successfully.", w)
				},
				SubmitText: "Save",
			}

			dialog.ShowCustom(fmt.Sprintf("Edit %s/%s", selected.Owner, selected.Repository), "Close",
				container.NewVScroll(form), w)
		})

		removeBtn := widget.NewButton("Remove", func() {
			dialog.ShowConfirm("Remove Repository",
				fmt.Sprintf("Remove %s/%s@%s?", selected.Owner, selected.Repository, selected.Ref),
				func(ok bool) {
					if !ok {
						return
					}
					rt.mu.Lock()
					for pname, wrapper := range rt.state.Providers {
						if pname != selected.Provider {
							continue
						}
						filtered := wrapper.Repositories[:0]
						for _, r := range wrapper.Repositories {
							if r.Owner == selected.Owner &&
								r.Repository == selected.Repository &&
								r.Ref == selected.Ref {
								continue
							}
							filtered = append(filtered, r)
						}
						wrapper.Repositories = filtered
						rt.state.Providers[pname] = wrapper
					}
					rt.state.RebuildRepositoriesCache()
					rt.mu.Unlock()
					saveState(rt)
					repoList.Refresh()
					dialog.ShowInformation("Removed", "Repository removed.", w)
				}, w)
		})

		closeBtn := widget.NewButton("Close", func() {})

		dialog.ShowCustom("Repository Actions", "Dismiss",
			container.NewVBox(
				widget.NewLabel(fmt.Sprintf("Selected: %s/%s@%s (%s)",
					selected.Owner, selected.Repository, selected.Ref, selected.Analyzer)),
				widget.NewSeparator(),
				container.NewHBox(editBtn, removeBtn, closeBtn),
			), w)
	}

	status := widget.NewLabel("No repos loaded.")

	loadConfigBtn := widget.NewButton("Load CLI YAML...", func() {
		fd := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if rc == nil {
				return
			}
			defer func() { _ = rc.Close() }()
			path := rc.URI().Path()
			if path == "" {
				return
			}
			if err := rt.state.MergeCLIConfig(path); err != nil {
				dialog.ShowError(err, w)
				return
			}
			saveState(rt)
			repoList.Refresh()
			status.SetText(fmt.Sprintf("Loaded %d repositories", len(rt.state.RepositoriesCache)))
			slog.Info("Config merged", "path", path, "repos", len(rt.state.RepositoriesCache))
		}, w)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".yaml", ".yml"}))
		fd.Show()
	})

	addRepoBtn := widget.NewButton("Add Repository...", func() {
		showAddRepositoryDialog(rt, w, repoList, status)
	})

	return container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("Repository Management", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewSeparator(),
			container.NewHBox(addRepoBtn, loadConfigBtn),
			status,
		),
		nil, nil, nil,
		repoList,
	)
}

func showAddRepositoryDialog(rt *Runtime, w fyne.Window, list *widget.List, status *widget.Label) {
	providerEntry := widget.NewSelect([]string{"github", "gitlab"}, func(string) {})
	providerEntry.SetSelected("github")

	ownerEntry := widget.NewEntry()
	ownerEntry.SetPlaceHolder("Owner / namespace")

	repoEntry := widget.NewEntry()
	repoEntry.SetPlaceHolder("Repository name")

	refEntry := widget.NewEntry()
	refEntry.SetText("main")

	analyzerEntry := widget.NewSelect([]string{"poetry"}, func(string) {})
	analyzerEntry.SetSelected("poetry")

	pathsEntry := widget.NewMultiLineEntry()
	pathsEntry.SetPlaceHolder("Paths (one per line, optional)")

	packagesEntry := widget.NewMultiLineEntry()
	packagesEntry.SetPlaceHolder("Packages (one per line)")

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Provider", Widget: providerEntry},
			{Text: "Owner", Widget: ownerEntry},
			{Text: "Repository", Widget: repoEntry},
			{Text: "Ref", Widget: refEntry},
			{Text: "Analyzer", Widget: analyzerEntry},
			{Text: "Paths", Widget: pathsEntry},
			{Text: "Packages", Widget: packagesEntry},
		},
		OnSubmit: func() {
			provider := providerEntry.Selected
			owner := strings.TrimSpace(ownerEntry.Text)
			repo := strings.TrimSpace(repoEntry.Text)
			ref := strings.TrimSpace(refEntry.Text)
			analyzer := analyzerEntry.Selected

			if provider == "" || owner == "" || repo == "" || analyzer == "" {
				dialog.ShowError(fmt.Errorf("required fields missing"), w)
				return
			}

			paths := filterNonEmptyLines(pathsEntry.Text)
			packages := filterNonEmptyLines(packagesEntry.Text)

			rt.mu.Lock()
			wrapper := rt.state.Providers[provider]
			if wrapper.Default.Analyzer == "" {
				wrapper.Default.Analyzer = "poetry"
			}
			wrapper.Repositories = append(wrapper.Repositories, config.RepoConfig{
				Owner:      owner,
				Repository: repo,
				Ref:        ref,
				Paths:      paths,
				Packages:   packages,
				Analyzer:   analyzer,
			})
			rt.state.Providers[provider] = wrapper
			rt.state.RebuildRepositoriesCache()
			rt.mu.Unlock()

			saveState(rt)
			list.Refresh()
			status.SetText(fmt.Sprintf("Repositories: %d", len(rt.state.RepositoriesCache)))
			dialog.ShowInformation("Added", fmt.Sprintf("Repository %s/%s added.", owner, repo), w)
		},
		SubmitText: "Add",
	}

	dialog.ShowCustom("Add Repository", "Close", container.NewVScroll(form), w)
}

func filterNonEmptyLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

// ----- Packages (Tracked) View -----

func buildPackagesView(rt *Runtime, _ fyne.App, w fyne.Window) fyne.CanvasObject {
	list := widget.NewList(
		func() int {
			rt.mu.RLock()
			defer rt.mu.RUnlock()
			return len(rt.state.TrackedPackages)
		},
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			rt.mu.RLock()
			defer rt.mu.RUnlock()
			if i < len(rt.state.TrackedPackages) {
				o.(*widget.Label).SetText(rt.state.TrackedPackages[i])
			} else {
				o.(*widget.Label).SetText("")
			}
		},
	)

	status := widget.NewLabel("No tracked packages defined (uses all).")

	editBtn := widget.NewButton("Edit Tracked Packages...", func() {
		editTrackedPackagesDialog(rt, w, list, status)
	})

	resetBtn := widget.NewButton("Clear", func() {
		rt.mu.Lock()
		rt.state.TrackedPackages = []string{}
		rt.mu.Unlock()
		saveState(rt)
		list.Refresh()
		status.SetText("Cleared; table will show all discovered packages.")
	})

	return container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("Tracked Packages", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewSeparator(),
			container.NewHBox(editBtn, resetBtn),
			status,
		),
		nil, nil, nil,
		list,
	)
}

func editTrackedPackagesDialog(rt *Runtime, w fyne.Window, list *widget.List, status *widget.Label) {
	rt.mu.RLock()
	current := append([]string{}, rt.state.TrackedPackages...)
	rt.mu.RUnlock()

	entry := widget.NewMultiLineEntry()
	entry.SetText(strings.Join(current, "\n"))
	entry.SetPlaceHolder("One package per line")

	saveBtn := widget.NewButton("Save", func() {
		newPkgs := filterNonEmptyLines(entry.Text)
		rt.mu.Lock()
		rt.state.TrackedPackages = newPkgs
		rt.mu.Unlock()
		saveState(rt)
		list.Refresh()
		if len(newPkgs) == 0 {
			status.SetText("No tracked packages defined (uses all).")
		} else {
			status.SetText(fmt.Sprintf("%d tracked packages.", len(newPkgs)))
		}
		dialog.ShowInformation("Saved", "Tracked packages updated.", w)
	})

	dialog.ShowCustom("Edit Tracked Packages", "Close",
		container.NewBorder(nil, container.NewHBox(saveBtn), nil, nil,
			widget.NewLabel("Enter packages to display in the main report table (one per line)."),
			entry,
		), w)
}

// ----- Dependencies (Report) View -----

// calculateRepoColumnWidth calculates the optimal width for the repository column
// based on the longest repository name in the report
func calculateRepoColumnWidth(rpt *report.Report) float32 {
	if rpt == nil || len(rpt.Repositories) == 0 {
		return 300 // default width
	}

	// Find the longest repository name
	maxLen := len("Repository") // Start with header text
	longestText := "Repository"

	for _, repo := range rpt.Repositories {
		repoText := fmt.Sprintf("%s/%s@%s", repo.Owner, repo.Repository, repo.Ref)
		if len(repoText) > maxLen {
			maxLen = len(repoText)
			longestText = repoText
		}
	}

	// Measure the text width using Fyne's text measurement with bold style
	// (header is bold, so use that for measurement)
	textSize := fyne.MeasureText(longestText, fyne.CurrentApp().Settings().Theme().Size("text"), fyne.TextStyle{Bold: true})

	// Add padding (20px on each side)
	width := textSize.Width + 40

	// Enforce minimum and maximum bounds
	if width < 150 {
		width = 150
	}
	if width > 600 {
		width = 600
	}

	return width
}

// calculatePackageColumnWidth calculates the optimal width for a package column
// based on the longest version string or package name (header)
func calculatePackageColumnWidth(rpt *report.Report, packageName string) float32 {
	if rpt == nil {
		return 120 // default width
	}

	// Start with the package name (header) as it's displayed in bold
	longestText := packageName

	// Check all version strings for this package
	for _, repo := range rpt.Repositories {
		version := repo.Dependencies[packageName]
		if version != "" && len(version) > len(longestText) {
			longestText = version
		}
	}

	// Also consider "ERR" as a possible value
	if len("ERR") > len(longestText) {
		longestText = "ERR"
	}

	// Measure the text width using Fyne's text measurement with bold style
	// (header is bold, and we want to ensure it fits)
	textSize := fyne.MeasureText(longestText, fyne.CurrentApp().Settings().Theme().Size("text"), fyne.TextStyle{Bold: true})

	// Add padding (15px on each side)
	width := textSize.Width + 30

	// Enforce minimum and maximum bounds
	if width < 80 {
		width = 80
	}
	if width > 300 {
		width = 300
	}

	return width
}

func buildDependenciesView(rt *Runtime, w fyne.Window, enqueueUI func(func())) fyne.CanvasObject {
	var table *widget.Table // declare early so we can reference it
	var _ = table           // avoid unused variable error until table is assigned
	status := widget.NewLabel("No report generated.")
	progressList := widget.NewList(
		func() int {
			rt.mu.RLock()
			defer rt.mu.RUnlock()
			return len(rt.progressEvents)
		},
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			rt.mu.RLock()
			defer rt.mu.RUnlock()
			if i < len(rt.progressEvents) {
				ev := rt.progressEvents[i]
				txt := fmt.Sprintf("%s %s", ev.Phase, ev.RepoID)
				if ev.Error != nil {
					txt += " (error)"
				}
				o.(*widget.Label).SetText(txt)
			} else {
				o.(*widget.Label).SetText("")
			}
		},
	)

	refreshBtn := widget.NewButton("Refresh Report", func() {
		runReportAsync(rt, enqueueUI, status, table)
	})
	exportBtn := widget.NewButton("Export JSON", func() {
		exportJSONReport(rt, w)
	})

	table = widget.NewTable(
		func() (int, int) {
			rt.mu.RLock()
			defer rt.mu.RUnlock()
			if rt.currentReport == nil {
				return 1, 1
			}
			// header + repositories
			rows := len(rt.currentReport.Repositories) + 1
			tracked := rt.state.TrackedPackages
			var cols int
			if len(tracked) == 0 {
				cols = len(rt.currentReport.Packages) + 1
			} else {
				cols = len(tracked) + 1
			}
			return rows, cols
		},
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(cell widget.TableCellID, o fyne.CanvasObject) {
			rt.mu.RLock()
			defer rt.mu.RUnlock()
			lbl := o.(*widget.Label)
			if rt.currentReport == nil {
				if cell.Row == 0 && cell.Col == 0 {
					lbl.SetText("No data")
				} else {
					lbl.SetText("")
				}
				return
			}
			rpt := rt.currentReport
			tracked := rt.state.TrackedPackages
			var packages []string
			if len(tracked) == 0 {
				packages = rpt.Packages
			} else {
				packages = tracked
			}

			if cell.Row == 0 {
				if cell.Col == 0 {
					lbl.SetText("Repository")
				} else {
					lbl.SetText(packages[cell.Col-1])
				}
				lbl.TextStyle = fyne.TextStyle{Bold: true}
				return
			}

			repoIdx := cell.Row - 1
			if repoIdx >= len(rpt.Repositories) {
				lbl.SetText("")
				return
			}
			repoReport := rpt.Repositories[repoIdx]
			if cell.Col == 0 {
				lbl.SetText(fmt.Sprintf("%s/%s@%s", repoReport.Owner, repoReport.Repository, repoReport.Ref))
				return
			}
			pkgName := packages[cell.Col-1]
			version := repoReport.Dependencies[pkgName]
			if version == "" {
				if repoReport.Error != nil {
					lbl.SetText("ERR")
				} else {
					lbl.SetText("—")
				}
				return
			}
			lbl.SetText(version)
		},
	)

	table.OnSelected = func(id widget.TableCellID) {
		if id.Row == 0 {
			return
		}
		rt.mu.RLock()
		defer rt.mu.RUnlock()
		if rt.currentReport == nil {
			return
		}
		repoIdx := id.Row - 1
		if repoIdx >= len(rt.currentReport.Repositories) {
			return
		}
		showRepoDetailsModal(rt.currentReport.Repositories[repoIdx], w)
	}

	// Set initial column widths
	rt.mu.RLock()
	if rt.currentReport != nil {
		// Calculate and set repository column width based on content
		repoColWidth := calculateRepoColumnWidth(rt.currentReport)
		table.SetColumnWidth(0, repoColWidth)

		// Set package column widths dynamically based on content
		tracked := rt.state.TrackedPackages
		var packages []string
		if len(tracked) == 0 {
			packages = rt.currentReport.Packages
		} else {
			packages = tracked
		}
		for i, pkgName := range packages {
			colWidth := calculatePackageColumnWidth(rt.currentReport, pkgName)
			table.SetColumnWidth(i+1, colWidth)
		}
	} else {
		// No report yet, use default widths
		table.SetColumnWidth(0, 300)
		for i := 1; i < 20; i++ {
			table.SetColumnWidth(i, 120)
		}
	}
	rt.mu.RUnlock()

	return container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("Dependencies Report", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewSeparator(),
			container.NewHBox(refreshBtn, exportBtn),
			status,
			widget.NewLabel("Progress:"),
			progressList,
		),
		nil, nil, nil,
		container.NewStack(table),
	)
}

func runReportAsync(rt *Runtime, enqueueUI func(func()), statusLabel *widget.Label, table *widget.Table) {
	rt.mu.Lock()
	if rt.reportRunning {
		rt.mu.Unlock()
		if statusLabel != nil {
			statusLabel.SetText("Report already running...")
		}
		return
	}
	rt.reportRunning = true
	rt.progressEvents = []services.ReportProgress{}
	rt.progressIndex = map[string]services.ReportProgress{}
	repos := make([]config.RepoWithProvider, 0, len(rt.state.RepositoriesCache))
	for _, rc := range rt.state.RepositoriesCache {
		repos = append(repos, config.RepoWithProvider{
			Provider: rc.Provider,
			Config: config.RepoConfig{
				Token:      rc.Token,
				Owner:      rc.Owner,
				Repository: rc.Repository,
				Ref:        rc.Ref,
				Paths:      rc.Paths,
				Packages:   rc.Packages,
				Analyzer:   rc.Analyzer,
			},
		})
	}
	rt.mu.Unlock()

	if statusLabel != nil {
		enqueueUI(func() {
			statusLabel.SetText("Running report...")
		})
	}

	// Resolve provider tokens before generating the report.
	for idx := range repos {
		rp := &repos[idx]
		// If token already set in config, keep it; otherwise attempt resolution.
		if rp.Config.Token == "" {
			tok, terr := statepkg.ResolveProviderToken(rp.Provider, rt.state, rt.credentialStore)
			if terr != nil {
				// Record structured error (non-fatal for this repo, token may remain empty)
				rt.mu.Lock()
				rt.state.ErrorLog = append(rt.state.ErrorLog, statepkg.ErrorLogEntry{
					Time:     time.Now().UTC(),
					Source:   "token-resolve",
					Severity: "error",
					Message:  fmt.Sprintf("Failed to resolve token for %s:%s/%s", rp.Provider, rp.Config.Owner, rp.Config.Repository),
					Details:  terr.Error(),
				})
				rt.mu.Unlock()
			} else if tok != "" {
				rp.Config.Token = tok
				slog.Debug("Resolved token",
					"provider", rp.Provider,
					"owner", rp.Config.Owner,
					"repo", rp.Config.Repository,
					"tokenRedacted", statepkg.RedactToken(tok))
			} else {
				slog.Debug("No token resolved (anonymous access)",
					"provider", rp.Provider,
					"owner", rp.Config.Owner,
					"repo", rp.Config.Repository)
			}
		}
	}

	slog.Info("Starting dependency report", "repos", len(repos))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	progressCh, handle, err := rt.depSvc.RunReport(ctx, repos, services.ReportOptions{
		EmitAggregateEvents: true,
	})
	if err != nil {
		cancel()
		fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "Report Error", Content: fmt.Sprintf("Setup failed: %v", err)})
		if statusLabel != nil {
			enqueueUI(func() {
				statusLabel.SetText(fmt.Sprintf("Report setup failed: %v", err))
			})
		}
		enqueueUI(func() {
			enqueueUI(func() {
				if drv := fyne.CurrentApp().Driver(); drv != nil {
					for _, w := range fyne.CurrentApp().Driver().AllWindows() {
						w.Canvas().Refresh(w.Content())
					}
				}
			})
		})
		rt.mu.Lock()
		rt.reportRunning = false
		rt.mu.Unlock()
		slog.Error("RunReport failed", "error", err)
		return
	}

	// Progress collector
	go func() {
		for p := range progressCh {
			rt.mu.Lock()
			rt.progressEvents = append(rt.progressEvents, p)
			rt.progressIndex[p.RepoID] = p
			rt.mu.Unlock()
			if drv := fyne.CurrentApp().Driver(); drv != nil {
				for _, w := range fyne.CurrentApp().Driver().AllWindows() {
					w.Canvas().Refresh(w.Content())
				}
			}
		}
	}()

	// Completion
	go func() {
		defer cancel()
		rpt, rErr := handle.Result()
		rt.mu.Lock()
		rt.currentReport = rpt
		rt.reportRunning = false
		if rErr != nil {
			// Append aggregated error event entry if not already captured
			rt.progressEvents = append(rt.progressEvents, services.ReportProgress{
				RepoID:    "",
				Phase:     services.PhaseError,
				Error:     rErr,
				Timestamp: time.Now(),
			})
		} else {
			// Update last report meta
			rt.state.GUI.LastReport = &statepkg.LastReportMeta{
				GeneratedAt:  time.Now().UTC(),
				RepoCount:    len(rpt.Repositories),
				PackageCount: len(rpt.Packages),
			}
		}
		rt.mu.Unlock()
		saveState(rt)

		if rErr != nil {
			fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "Report Failed", Content: rErr.Error()})
			if statusLabel != nil {
				enqueueUI(func() {
					statusLabel.SetText(fmt.Sprintf("Report failed: %v", rErr))
				})
			}
			slog.Error("Report failed", "error", rErr)
		} else if rpt != nil {
			fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "Report Complete", Content: fmt.Sprintf("%d repos, %d packages", len(rpt.Repositories), len(rpt.Packages))})
			if statusLabel != nil {
				enqueueUI(func() {
					statusLabel.SetText(fmt.Sprintf("Report complete (%d repos, %d packages)", len(rpt.Repositories), len(rpt.Packages)))
				})
			}
			slog.Info("Report complete", "repos", len(rpt.Repositories), "packages", len(rpt.Packages))

			// Update table column widths based on new report data
			if table != nil && rpt != nil {
				enqueueUI(func() {
					// Calculate and set repository column width based on content
					repoColWidth := calculateRepoColumnWidth(rpt)
					table.SetColumnWidth(0, repoColWidth)

					// Update package column widths dynamically based on content
					rt.mu.RLock()
					tracked := rt.state.TrackedPackages
					var packages []string
					if len(tracked) == 0 {
						packages = rpt.Packages
					} else {
						packages = tracked
					}
					rt.mu.RUnlock()

					for i, pkgName := range packages {
						colWidth := calculatePackageColumnWidth(rpt, pkgName)
						table.SetColumnWidth(i+1, colWidth)
					}
					table.Refresh()
				})
			}
		}
		enqueueUI(func() {
			// Append successful history entry (only on success with a non-nil report)
			if rErr == nil && rpt != nil {
				rt.mu.Lock()
				rt.state.ReportHistory = append(rt.state.ReportHistory, statepkg.ReportHistoryEntry{
					GeneratedAt:  time.Now().UTC(),
					RepoCount:    len(rpt.Repositories),
					PackageCount: len(rpt.Packages),
					SummaryPath:  "",
				})
				rt.mu.Unlock()
				saveState(rt)
			}
			if drv := fyne.CurrentApp().Driver(); drv != nil {
				for _, w := range fyne.CurrentApp().Driver().AllWindows() {
					w.Canvas().Refresh(w.Content())
				}
			}
		})
	}()
}

// ----- Repo Detail Modal -----

func showRepoDetailsModal(repo report.RepositoryReport, w fyne.Window) {
	content := container.NewVBox(
		widget.NewLabelWithStyle(fmt.Sprintf("Repository: %s/%s@%s",
			repo.Owner, repo.Repository, repo.Ref),
			fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
	)
	if repo.Error != nil {
		content.Add(widget.NewLabel(fmt.Sprintf("Error: %v", repo.Error)))
	}
	content.Add(widget.NewLabel("Dependencies:"))
	for pkg, ver := range repo.Dependencies {
		content.Add(widget.NewLabel(fmt.Sprintf("  %s: %s", pkg, ver)))
	}
	dialog.ShowCustom("Repository Details", "Close", container.NewVScroll(content), w)
}

// ----- Logs View -----

func buildLogsView(rt *Runtime, _ fyne.App, _ fyne.Window, logHandler *RingLogHandler) fyne.CanvasObject {
	// Filtering controls
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Filter text (substring)")

	// Forward declarations so callbacks can reference logList after assignment.
	var logList *widget.List
	errorOnlyToggle := widget.NewCheck("Show only errors", func(bool) {
		if logList != nil {
			logList.Refresh()
		}
	})

	structuredErrorsToggle := widget.NewCheck("Show structured errors", func(bool) {
		if logList != nil {
			logList.Refresh()
		}
	})

	levelSelect := widget.NewSelect([]string{"ALL", "DEBUG", "INFO", "WARN", "ERROR"}, func(string) {
		if logList != nil {
			logList.Refresh()
		}
	})
	levelSelect.SetSelected("ALL")

	// List with dynamic filtering (assigned after control declarations)
	logList = widget.NewList(
		func() int {
			entries := filteredLogs(logHandler, searchEntry.Text, levelSelect.Selected, errorOnlyToggle.Checked, structuredErrorsToggle.Checked, rt)
			return len(entries)
		},
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			entries := filteredLogs(logHandler, searchEntry.Text, levelSelect.Selected, errorOnlyToggle.Checked, structuredErrorsToggle.Checked, rt)
			if i < len(entries) {
				e := entries[i]
				o.(*widget.Label).SetText(fmt.Sprintf("%s [%s] %s",
					e.Time.Format(time.RFC3339), e.Level.String(), e.Message))
			} else {
				o.(*widget.Label).SetText("")
			}
		},
	)

	refreshBtn := widget.NewButton("Refresh", func() {
		if logList != nil {
			logList.Refresh()
		}
	})
	clearBtn := widget.NewButton("Clear", func() {
		logHandler.mu.Lock()
		logHandler.logs = []LogEntry{}
		logHandler.mu.Unlock()
		if logList != nil {
			logList.Refresh()
		}
	})

	controls := container.NewVBox(
		widget.NewLabelWithStyle("Logs", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		container.NewHBox(searchEntry, levelSelect),
		container.NewHBox(errorOnlyToggle, structuredErrorsToggle),
		container.NewHBox(refreshBtn, clearBtn),
	)

	return container.NewBorder(
		controls,
		nil, nil, nil,
		logList,
	)
}
func filteredLogs(logHandler *RingLogHandler, search, levelFilter string, errorsOnly bool, structuredErrors bool, rt *Runtime) []LogEntry {
	logHandler.mu.RLock()
	defer logHandler.mu.RUnlock()
	if len(logHandler.logs) == 0 {
		return nil
	}
	search = strings.TrimSpace(strings.ToLower(search))
	levelFilter = strings.ToUpper(strings.TrimSpace(levelFilter))

	var out []LogEntry
	if structuredErrors {
		// Map structured errors into LogEntry shape
		rt.mu.RLock()
		errs := append([]statepkg.ErrorLogEntry{}, rt.state.ErrorLog...)
		rt.mu.RUnlock()
		for _, se := range errs {
			msg := se.Message
			if se.Details != "" {
				msg = msg + " (" + se.Details + ")"
			}
			out = append(out, LogEntry{
				Time:    se.Time,
				Level:   slog.LevelError,
				Message: fmt.Sprintf("[%s] %s", se.Source, msg),
				Attrs:   []slog.Attr{slog.String("severity", se.Severity)},
			})
		}
	} else {
		for _, e := range logHandler.logs {
			// Level filter
			if levelFilter != "" && levelFilter != "ALL" && strings.ToUpper(e.Level.String()) != levelFilter {
				continue
			}
			// Errors-only toggle
			if errorsOnly && strings.ToUpper(e.Level.String()) != "ERROR" {
				continue
			}
			// Substring search (match message or level)
			if search != "" {
				if !strings.Contains(strings.ToLower(e.Message), search) &&
					!strings.Contains(strings.ToLower(e.Level.String()), search) {
					continue
				}
			}
			out = append(out, e)
		}
	}
	return out
}

// ----- JSON Export -----

type jsonOutput struct {
	Version      string                    `json:"cliVersion"`
	GeneratedAt  time.Time                 `json:"generatedAt"`
	Repositories []report.RepositoryReport `json:"repositories"`
	Packages     []string                  `json:"packages"`
	Summary      jsonSummary               `json:"summary"`
	Errors       map[string]string         `json:"errors,omitempty"`
}

type jsonSummary struct {
	RepositoryCount int `json:"repositoryCount"`
	PackageCount    int `json:"packageCount"`
	SuccessCount    int `json:"successCount"`
	ErrorCount      int `json:"errorCount"`
}

func exportJSONReport(rt *Runtime, w fyne.Window) {
	rt.mu.RLock()
	rpt := rt.currentReport
	rt.mu.RUnlock()

	if rpt == nil {
		dialog.ShowInformation("Export JSON", "No report to export.", w)
		return
	}

	fs := dialog.NewFileSave(func(uc fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		if uc == nil {
			return
		}
		defer func() { _ = uc.Close() }()

		successCount := 0
		for _, rr := range rpt.Repositories {
			if rr.Error == nil {
				successCount++
			}
		}
		errCount := len(rpt.Repositories) - successCount
		errMap := map[string]string{}
		for _, rr := range rpt.Repositories {
			if rr.Error != nil {
				key := fmt.Sprintf("%s:%s/%s@%s", rr.Provider, rr.Owner, rr.Repository, rr.Ref)
				errMap[key] = rr.Error.Error()
			}
		}

		payload := jsonOutput{
			Version:      version,
			GeneratedAt:  time.Now().UTC(),
			Repositories: rpt.Repositories,
			Packages:     rpt.Packages,
			Summary: jsonSummary{
				RepositoryCount: len(rpt.Repositories),
				PackageCount:    len(rpt.Packages),
				SuccessCount:    successCount,
				ErrorCount:      errCount,
			},
		}
		if len(errMap) > 0 {
			payload.Errors = errMap
		}

		data, mErr := json.MarshalIndent(payload, "", "  ")
		if mErr != nil {
			dialog.ShowError(mErr, w)
			return
		}
		if _, wErr := uc.Write(data); wErr != nil {
			dialog.ShowError(wErr, w)
			return
		}
		dialog.ShowInformation("Export JSON", "Report exported successfully.", w)
	}, w)
	fs.SetFileName("dependency-report.json")
	fs.Show()
}

// ----- State Saving (Debounced) -----

var saveMu sync.Mutex

var saveTimer *time.Timer

func saveState(rt *Runtime) {
	saveMu.Lock()
	defer saveMu.Unlock()

	if saveTimer != nil {
		saveTimer.Stop()
	}
	// Debounce writes (250ms)
	saveTimer = time.AfterFunc(250*time.Millisecond, func() {
		saveMu.Lock()
		defer saveMu.Unlock()
		rt.mu.RLock()
		st := rt.state
		rt.mu.RUnlock()

		if err := statepkg.SaveGUIState(st, ""); err != nil {
			slog.Error("Failed to save state", "error", err)
		} else {
			slog.Debug("State saved", "path", statepkg.DefaultGUIStatePath())
		}

	})

}

// ----- Utility for window geometry update -----

// (removed unused updateWindowGeometryOnResize)

// (Optional future) hooking updateWindowGeometryOnResize(rt) after UI creation.
// For now omitted to keep resource usage low.

// ----- END -----

// (removed unused debugRuntimeSnapshot)

// ----- History View (placeholder) -----
func buildHistoryView(rt *Runtime) fyne.CanvasObject {
	rt.mu.RLock()
	hist := rt.state.ReportHistory
	rt.mu.RUnlock()

	if len(hist) == 0 {
		return container.NewCenter(widget.NewLabel("No report history yet."))
	}

	list := widget.NewList(
		func() int { return len(hist) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i >= len(hist) {
				o.(*widget.Label).SetText("")
				return
			}
			entry := hist[i]
			o.(*widget.Label).SetText(fmt.Sprintf("%s - %d repos / %d packages",
				entry.GeneratedAt.Format(time.RFC3339),
				entry.RepoCount,
				entry.PackageCount,
			))
		},
	)

	return container.NewBorder(
		widget.NewLabelWithStyle("History", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil, nil, nil,
		list,
	)
}
