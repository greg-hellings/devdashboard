//go:build gui
// +build gui

package main

// DevDashboard GUI (experimental)
// --------------------------------
// This is a placeholder entrypoint for an optional Fyne-based GUI for DevDashboard.
//
// Optional Build Strategy:
//   1. The file uses a build tag `gui` so it is excluded from normal builds.
//   2. To build the GUI binary:
//        go build -tags gui -o devdashboard-gui ./cmd/devdashboard-gui
//   3. To run it directly:
//        go run -tags gui ./cmd/devdashboard-gui
//
// Why a build tag?
//   - Keeps GUI dependencies (Fyne) out of default builds: `go build ./...` won't pull them in.
//   - Avoids forcing all users to install graphical dependencies if they only want the CLI.
//
// To integrate Fyne:
//   Add the following to go.mod (will happen automatically if you build with this file present):
//        require fyne.io/fyne/v2 latest
//
// Future Plans (suggested):
//   - Config file selection (file picker)
//   - Execute dependency report in a goroutine and render table results
//   - Progress indicator and error panel
//   - Theme switching and packaging (macOS .app, Windows .exe w/ manifest)
//
// NOTE: Until fully implemented, this GUI simply renders a stub window.

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	fapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// build-time override (e.g. -ldflags "-X main.version=1.2.3")
var version = "dev-gui"

// guiState holds transient GUI application state.
// As features grow (config selection, running background tasks), expand this.
type guiState struct {
	lastRun time.Time
	cancel  context.CancelFunc
}

// main initializes the Fyne application and sets up a placeholder window.
func main() {
	// Initialize logging similar to CLI (simplified for now).
	slog.SetDefault(slog.New(slog.NewTextHandler(fyneLogWriter{}, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	app := fapp.NewWithID("devdashboard.gui")
	app.SetIcon(nil) // Could embed an icon later.

	state := &guiState{}
	w := app.NewWindow("DevDashboard GUI (Prototype)")
	w.Resize(fyne.NewSize(640, 400))

	header := widget.NewLabel(fmt.Sprintf("DevDashboard GUI (version: %s)", version))
	header.Alignment = fyne.TextAlignCenter
	header.Wrapping = fyne.TextWrapWord

	statusLabel := widget.NewLabel("Status: Idle")
	statusLabel.Wrapping = fyne.TextWrapWord

	runButton := widget.NewButton("Run Dependency Report (stub)", func() {
		// Placeholder: integrate report logic here.
		state.lastRun = time.Now()
		statusLabel.SetText(fmt.Sprintf("Status: Ran stub at %s", state.lastRun.Format(time.RFC822)))
	})

	openConfigButton := widget.NewButton("Open Config (stub)", func() {
		dialog.ShowInformation("Config Loader", "File picker not yet implemented.", w)
	})

	quitButton := widget.NewButton("Quit", func() {
		app.Quit()
	})

	content := container.NewVBox(
		header,
		widget.NewSeparator(),
		widget.NewLabel("This is an experimental GUI interface for DevDashboard.\n\nRoadmap:\n  • Select a config file\n  • Run dependency report asynchronously\n  • Display table results\n  • Show errors and summary\n  • Export to JSON / console\n"),
		statusLabel,
		container.NewHBox(runButton, openConfigButton, quitButton),
		widget.NewSeparator(),
		widget.NewLabel("Build with: go build -tags gui -o devdashboard-gui ./cmd/devdashboard-gui"),
	)

	w.SetContent(content)
	w.SetCloseIntercept(func() {
		// Add any cleanup (cancel goroutines, etc.) here.
		app.Quit()
	})

	w.ShowAndRun()
}

// fyneLogWriter is a minimal adapter to satisfy slog handler output needs.
// For richer logging (log panel inside GUI), buffer logs and render in a widget.
type fyneLogWriter struct{}

func (fyneLogWriter) Write(p []byte) (int, error) {
	// For now, just send to stdout. Could append to a scrolling text widget later.
	fmt.Print(string(p))
	return len(p), nil
}
