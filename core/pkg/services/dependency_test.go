package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/greg-hellings/devdashboard/core/pkg/config"
	"github.com/greg-hellings/devdashboard/core/pkg/report"
)

func TestNewDependencyService(t *testing.T) {
	t.Run("with nil generator creates default", func(t *testing.T) {
		svc := NewDependencyService(nil)
		if svc == nil {
			t.Fatal("expected non-nil service")
		}
	})

	t.Run("with custom generator", func(t *testing.T) {
		gen := report.NewGenerator()
		svc := NewDependencyService(gen)
		if svc == nil {
			t.Fatal("expected non-nil service")
		}
	})
}

func TestDependencyService_RunReport_NoRepos(t *testing.T) {
	svc := NewDependencyService(nil)
	ctx := context.Background()

	_, _, err := svc.RunReport(ctx, []config.RepoWithProvider{}, ReportOptions{})
	if err == nil {
		t.Fatal("expected error for empty repos slice")
	}
	if err.Error() != "no repositories provided" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestDependencyService_RunReport_ContextCancellation(t *testing.T) {
	repos := []config.RepoWithProvider{
		{
			Provider: "github",
			Config: config.RepoConfig{
				Owner:      "testowner",
				Repository: "testrepo",
				Ref:        "main",
				Analyzer:   "go",
			},
		},
	}

	svc := NewDependencyService(nil)
	ctx, cancel := context.WithCancel(context.Background())

	progressCh, handle, err := svc.RunReport(ctx, repos, ReportOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Cancel context immediately
	cancel()

	// Drain progress channel
	for range progressCh {
	}

	// Result should indicate cancellation
	_, err = handle.Result()
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestResultHandle_Done(t *testing.T) {
	repos := []config.RepoWithProvider{
		{
			Provider: "github",
			Config: config.RepoConfig{
				Owner:      "testowner",
				Repository: "nonexistent-repo-12345",
				Ref:        "main",
				Analyzer:   "go",
			},
		},
	}

	svc := NewDependencyService(nil)
	ctx := context.Background()

	_, handle, err := svc.RunReport(ctx, repos, ReportOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Done channel should eventually close
	select {
	case <-handle.Done():
		// Success
	case <-time.After(10 * time.Second):
		t.Fatal("Done() channel did not close in time")
	}
}

func TestProgressPhaseConstants(t *testing.T) {
	// Verify phase constants are defined correctly
	phases := []ProgressPhase{
		PhaseQueued,
		PhaseRunning,
		PhaseComplete,
		PhaseError,
		PhaseAggregate,
	}

	for _, phase := range phases {
		if string(phase) == "" {
			t.Errorf("phase constant should not be empty")
		}
	}
}

func TestReportProgress_Structure(t *testing.T) {
	now := time.Now()
	progress := ReportProgress{
		RepoID:    "github:owner/repo@main",
		Phase:     PhaseRunning,
		Error:     nil,
		Timestamp: now,
	}

	if progress.RepoID != "github:owner/repo@main" {
		t.Errorf("unexpected RepoID: %s", progress.RepoID)
	}
	if progress.Phase != PhaseRunning {
		t.Errorf("unexpected Phase: %s", progress.Phase)
	}
	if !progress.Timestamp.Equal(now) {
		t.Error("timestamp mismatch")
	}
}

func TestReportOptions_Defaults(t *testing.T) {
	opts := ReportOptions{}
	if opts.Concurrency != 0 {
		t.Errorf("expected default concurrency 0, got %d", opts.Concurrency)
	}
	if opts.EmitAggregateEvents {
		t.Error("expected EmitAggregateEvents to default to false")
	}
}

func TestReportProgress_ErrorPhase(t *testing.T) {
	testErr := errors.New("test error")
	progress := ReportProgress{
		RepoID:    "github:owner/repo@main",
		Phase:     PhaseError,
		Error:     testErr,
		Timestamp: time.Now(),
	}

	if progress.Phase != PhaseError {
		t.Error("expected PhaseError")
	}
	if progress.Error != testErr {
		t.Error("expected error to match")
	}
}

func TestDependencyService_Interface(t *testing.T) {
	// Verify that dependencyService implements DependencyService interface
	var _ DependencyService = (*dependencyService)(nil)
}

func TestResultHandle_MultipleCalls(t *testing.T) {
	repos := []config.RepoWithProvider{
		{
			Provider: "github",
			Config: config.RepoConfig{
				Owner:      "testowner",
				Repository: "nonexistent-repo-12345",
				Ref:        "main",
				Analyzer:   "go",
			},
		},
	}

	svc := NewDependencyService(nil)
	ctx := context.Background()

	_, handle, err := svc.RunReport(ctx, repos, ReportOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Call Result multiple times
	result1, err1 := handle.Result()
	result2, err2 := handle.Result()

	// Both calls should return the same result
	if result1 != result2 {
		t.Error("expected same result from multiple calls")
	}
	if err1 != err2 {
		t.Error("expected same error from multiple calls")
	}
}
