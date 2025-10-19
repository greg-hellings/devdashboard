package services

// Dependency service scaffold providing an abstraction for running
// dependency reports asynchronously with progress streaming.
// This initial version wraps the existing report.Generator and
// simulates per-repository progress events. In a future phase,
// deeper integration can emit granular phases (download, analyze,
// aggregate, etc.).
//
// Module path note: internal imports use the renamed core module
// path: github.com/greg-hellings/devdashboard/core/...
//
// Usage (example):
//    svc := NewDependencyService(nil) // uses default generator
//    ch, handle, err := svc.RunReport(ctx, repos, ReportOptions{}
//    for p := range ch {
//        fmt.Println(p.RepoID, p.Phase, p.Error)
//    }
//    rpt, err := handle.Result()
//
// Future Enhancements:
//  - Real-time per-repository analysis progress from report.Generator internals
//  - Cancellation propagation for individual repository tasks
//  - Metrics / durations per repository
//  - Retry support for transient failures
//  - Progress phases for discovery vs. analysis vs. aggregation
//
// TODO: Integrate granular stage-level progress (e.g., fetch metadata,
//       enumerate dependency files, analyze dependencies, aggregate results)
//       by instrumenting report.Generator to emit callbacks so GUI layers
//       can reflect finer-grained statuses.
//
// NOTE: This scaffold intentionally keeps implementation simple
// while defining stable interfaces for GUI & future web/mobile layers.

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/greg-hellings/devdashboard/core/pkg/config"
	"github.com/greg-hellings/devdashboard/core/pkg/report"
)

// ProgressPhase represents lifecycle phases for repository analysis.
// Additional phases (e.g., "discover", "fetch", "analyze") can be added later.
type ProgressPhase string

const (
	// PhaseQueued indicates the repository has been queued for analysis but not yet started.
	PhaseQueued ProgressPhase = "queued"
	// PhaseRunning indicates the repository analysis is currently in progress.
	PhaseRunning ProgressPhase = "running"
	// PhaseComplete indicates the repository analysis finished successfully.
	PhaseComplete ProgressPhase = "complete"
	// PhaseError indicates the repository analysis ended with an error.
	PhaseError ProgressPhase = "error"
	// PhaseAggregate represents synthetic aggregate events (overall report assembly).
	PhaseAggregate ProgressPhase = "aggregate" // overall report assembly
)

// ReportProgress conveys status updates for a single repository (or aggregate).
type ReportProgress struct {
	RepoID    string        // Provider:Owner/Repo@Ref (empty for aggregate events)
	Phase     ProgressPhase // Current phase
	Error     error         // Non-nil if PhaseError
	Timestamp time.Time     // Event emission time
}

// ReportOptions defines tunable behavior for a report run.
type ReportOptions struct {
	// Concurrency hint for future expansion (currently ignored, generator manages itself).
	Concurrency int

	// EmitAggregateEvents controls whether aggregate start/finish progress events are sent.
	EmitAggregateEvents bool

	// Reserved for future caching / retry strategy, etc.
}

// ResultHandle provides access to the final report.
type ResultHandle struct {
	mu     sync.RWMutex
	report *report.Report
	err    error
	done   chan struct{}
}

// Result blocks until the report completes (or context canceled).
func (h *ResultHandle) Result() (*report.Report, error) {
	<-h.done
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.report, h.err
}

// Done returns a channel closed when the report finishes.
func (h *ResultHandle) Done() <-chan struct{} {
	return h.done
}

// DependencyService is the public interface for asynchronous report generation.
type DependencyService interface {
	// RunReport initiates an asynchronous dependency report for the provided repositories.
	// Returns:
	//   progressCh  - channel streaming repository progress events (auto-closed)
	//   resultHandle - handle to obtain final report
	//   error        - immediate setup error (not analysis error)
	RunReport(ctx context.Context, repos []config.RepoWithProvider, opts ReportOptions) (<-chan ReportProgress, *ResultHandle, error)
}

// dependencyService is the default implementation.
type dependencyService struct {
	generator *report.Generator
}

// NewDependencyService constructs a DependencyService.
// If gen is nil, a new default generator is created.
func NewDependencyService(gen *report.Generator) DependencyService {
	if gen == nil {
		gen = report.NewGenerator()
	}
	return &dependencyService{generator: gen}
}

// RunReport launches the report generation asynchronously.
// Progress emission strategy:
//  1. Emit PhaseQueued for each repo.
//  2. Emit PhaseRunning for each repo just before starting the global generator (simulated).
//  3. Call generator.Generate once (current implementation is aggregate).
//  4. For each successful repo in final report emit PhaseComplete.
//     For each failed repo emit PhaseError.
//  5. Optionally emit aggregate start/finish events if opts.EmitAggregateEvents.
//
// NOTE: This is a coarse-grained simulation until deeper hooks are available.
func (s *dependencyService) RunReport(
	ctx context.Context,
	repos []config.RepoWithProvider,
	opts ReportOptions,
) (<-chan ReportProgress, *ResultHandle, error) {
	if len(repos) == 0 {
		return nil, nil, errors.New("no repositories provided")
	}

	progressCh := make(chan ReportProgress, len(repos)*4) // buffer heuristic

	handle := &ResultHandle{
		done: make(chan struct{}),
	}

	// Derive repo IDs
	repoIDs := make([]string, 0, len(repos))
	for _, r := range repos {
		id := fmt.Sprintf("%s:%s/%s@%s", r.Provider, r.Config.Owner, r.Config.Repository, r.Config.Ref)
		repoIDs = append(repoIDs, id)
	}

	go func() {
		defer close(progressCh)
		defer close(handle.done)

		// Emit queued events
		for _, id := range repoIDs {
			select {
			case <-ctx.Done():
				handle.mu.Lock()
				handle.err = ctx.Err()
				handle.mu.Unlock()
				return
			case progressCh <- ReportProgress{RepoID: id, Phase: PhaseQueued, Timestamp: time.Now()}:
			}
		}

		if opts.EmitAggregateEvents {
			select {
			case <-ctx.Done():
				handle.mu.Lock()
				handle.err = ctx.Err()
				handle.mu.Unlock()
				return
			case progressCh <- ReportProgress{RepoID: "", Phase: PhaseAggregate, Timestamp: time.Now()}:
			}
		}

		// Emit running (simulated)
		for _, id := range repoIDs {
			select {
			case <-ctx.Done():
				handle.mu.Lock()
				handle.err = ctx.Err()
				handle.mu.Unlock()
				return
			case progressCh <- ReportProgress{RepoID: id, Phase: PhaseRunning, Timestamp: time.Now()}:
			}
		}

		// Perform actual generation (single aggregate call)
		rpt, genErr := s.generator.Generate(ctx, repos)

		handle.mu.Lock()
		handle.report = rpt
		handle.err = genErr
		handle.mu.Unlock()

		// If generation failed entirely, emit error events for all repos.
		if genErr != nil {
			now := time.Now()
			for _, id := range repoIDs {
				progressCh <- ReportProgress{
					RepoID:    id,
					Phase:     PhaseError,
					Error:     genErr,
					Timestamp: now,
				}
			}
			if opts.EmitAggregateEvents {
				progressCh <- ReportProgress{
					RepoID:    "",
					Phase:     PhaseError,
					Error:     genErr,
					Timestamp: now,
				}
			}
			return
		}

		// Emit completion/error per repository from final report.
		if rpt != nil {
			now := time.Now()
			for _, rr := range rpt.Repositories {
				id := fmt.Sprintf("%s:%s/%s@%s", rr.Provider, rr.Owner, rr.Repository, rr.Ref)
				if rr.Error != nil {
					progressCh <- ReportProgress{
						RepoID:    id,
						Phase:     PhaseError,
						Error:     rr.Error,
						Timestamp: now,
					}
				} else {
					progressCh <- ReportProgress{
						RepoID:    id,
						Phase:     PhaseComplete,
						Timestamp: now,
					}
				}
			}
			if opts.EmitAggregateEvents {
				progressCh <- ReportProgress{
					RepoID:    "",
					Phase:     PhaseComplete,
					Timestamp: time.Now(),
				}
			}
		}
	}()

	return progressCh, handle, nil
}
