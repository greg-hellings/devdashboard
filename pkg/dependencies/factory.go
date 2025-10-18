package dependencies

import (
	"fmt"
	"strings"
)

// Factory creates dependency analyzers based on the analyzer type
type Factory struct{}

// NewFactory creates a new factory instance for dependency analyzers
func NewFactory() *Factory {
	return &Factory{}
}

// CreateAnalyzer creates a new dependency analyzer based on the analyzer type
// The analyzerType parameter is case-insensitive and supports the following values:
//   - "poetry" - Creates a Poetry (Python) analyzer
//   - "pipfile" - Creates a Pipfile (Python) analyzer
//   - "uvlock" - Creates a uv.lock (Python) analyzer
//
// Returns an error if the analyzer type is not recognized
func (f *Factory) CreateAnalyzer(analyzerType string) (Analyzer, error) {
	// Normalize analyzer type to lowercase for comparison
	normalized := strings.ToLower(strings.TrimSpace(analyzerType))

	switch AnalyzerType(normalized) {
	case AnalyzerPoetry:
		return NewPoetryAnalyzer(), nil
	case AnalyzerPipfile:
		return NewPipfileAnalyzer(), nil
	case AnalyzerUvLock:
		return NewUvLockAnalyzer(), nil
	default:
		return nil, fmt.Errorf("unsupported analyzer type: %s (supported: poetry, pipfile, uvlock)", analyzerType)
	}
}

// NewAnalyzer is a convenience function that creates a dependency analyzer
// without needing to instantiate a Factory first
// This is useful for simple use cases where you only need one analyzer
func NewAnalyzer(analyzerType string) (Analyzer, error) {
	factory := NewFactory()
	return factory.CreateAnalyzer(analyzerType)
}

// SupportedAnalyzers returns a list of all supported analyzer types
func SupportedAnalyzers() []string {
	return []string{
		string(AnalyzerPoetry),
		string(AnalyzerPipfile),
		string(AnalyzerUvLock),
	}
}
