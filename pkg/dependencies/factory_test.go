package dependencies

import (
	"testing"
)

// TestNewFactory verifies that factory creation works correctly
func TestNewFactory(t *testing.T) {
	factory := NewFactory()

	if factory == nil {
		t.Fatal("NewFactory returned nil")
	}
}

// TestCreateAnalyzerPoetry tests creating a Poetry analyzer via factory
func TestCreateAnalyzerPoetry(t *testing.T) {
	factory := NewFactory()
	analyzer, err := factory.CreateAnalyzer("poetry")

	if err != nil {
		t.Fatalf("Failed to create Poetry analyzer: %v", err)
	}

	if analyzer == nil {
		t.Fatal("CreateAnalyzer returned nil analyzer")
	}

	// Verify we got the correct type
	if _, ok := analyzer.(*PoetryAnalyzer); !ok {
		t.Errorf("Expected *PoetryAnalyzer, got %T", analyzer)
	}

	// Verify the name is correct
	if analyzer.Name() != "poetry" {
		t.Errorf("Expected analyzer name 'poetry', got '%s'", analyzer.Name())
	}
}

// TestCreateAnalyzerPipfile tests creating a Pipfile analyzer via factory
func TestCreateAnalyzerPipfile(t *testing.T) {
	factory := NewFactory()
	analyzer, err := factory.CreateAnalyzer("pipfile")

	if err != nil {
		t.Fatalf("Failed to create Pipfile analyzer: %v", err)
	}

	if analyzer == nil {
		t.Fatal("CreateAnalyzer returned nil analyzer")
	}

	// Verify we got the correct type
	if _, ok := analyzer.(*PipfileAnalyzer); !ok {
		t.Errorf("Expected *PipfileAnalyzer, got %T", analyzer)
	}

	// Verify the name is correct
	if analyzer.Name() != "pipfile" {
		t.Errorf("Expected analyzer name 'pipfile', got '%s'", analyzer.Name())
	}
}

// TestCreateAnalyzerUvLock tests creating a uv.lock analyzer via factory
func TestCreateAnalyzerUvLock(t *testing.T) {
	factory := NewFactory()
	analyzer, err := factory.CreateAnalyzer("uvlock")

	if err != nil {
		t.Fatalf("Failed to create UvLock analyzer: %v", err)
	}

	if analyzer == nil {
		t.Fatal("CreateAnalyzer returned nil analyzer")
	}

	// Verify we got the correct type
	if _, ok := analyzer.(*UvLockAnalyzer); !ok {
		t.Errorf("Expected *UvLockAnalyzer, got %T", analyzer)
	}

	// Verify the name is correct
	if analyzer.Name() != "uvlock" {
		t.Errorf("Expected analyzer name 'uvlock', got '%s'", analyzer.Name())
	}
}

// TestCreateAnalyzerCaseInsensitive verifies analyzer names are case-insensitive
func TestCreateAnalyzerCaseInsensitive(t *testing.T) {
	factory := NewFactory()

	testCases := []struct {
		analyzerType string
		expectedType interface{}
	}{
		{"Poetry", &PoetryAnalyzer{}},
		{"POETRY", &PoetryAnalyzer{}},
		{"poetry", &PoetryAnalyzer{}},
		{"PoEtRy", &PoetryAnalyzer{}},
		{"Pipfile", &PipfileAnalyzer{}},
		{"PIPFILE", &PipfileAnalyzer{}},
		{"pipfile", &PipfileAnalyzer{}},
		{"UvLock", &UvLockAnalyzer{}},
		{"UVLOCK", &UvLockAnalyzer{}},
		{"uvlock", &UvLockAnalyzer{}},
	}

	for _, tc := range testCases {
		t.Run(tc.analyzerType, func(t *testing.T) {
			analyzer, err := factory.CreateAnalyzer(tc.analyzerType)
			if err != nil {
				t.Fatalf("Failed to create analyzer for type %s: %v", tc.analyzerType, err)
			}

			if analyzer == nil {
				t.Fatalf("CreateAnalyzer returned nil for type %s", tc.analyzerType)
			}

			// Check the type matches expected
			switch tc.expectedType.(type) {
			case *PoetryAnalyzer:
				if _, ok := analyzer.(*PoetryAnalyzer); !ok {
					t.Errorf("Expected *PoetryAnalyzer for %s, got %T", tc.analyzerType, analyzer)
				}
			case *PipfileAnalyzer:
				if _, ok := analyzer.(*PipfileAnalyzer); !ok {
					t.Errorf("Expected *PipfileAnalyzer for %s, got %T", tc.analyzerType, analyzer)
				}
			case *UvLockAnalyzer:
				if _, ok := analyzer.(*UvLockAnalyzer); !ok {
					t.Errorf("Expected *UvLockAnalyzer for %s, got %T", tc.analyzerType, analyzer)
				}
			}
		})
	}
}

// TestCreateAnalyzerUnsupportedType tests error handling for unsupported analyzer types
func TestCreateAnalyzerUnsupportedType(t *testing.T) {
	factory := NewFactory()

	unsupportedTypes := []string{
		"npm",
		"maven",
		"gradle",
		"unknown",
		"",
		"   ",
	}

	for _, analyzerType := range unsupportedTypes {
		t.Run(analyzerType, func(t *testing.T) {
			analyzer, err := factory.CreateAnalyzer(analyzerType)

			if err == nil {
				t.Errorf("Expected error for unsupported analyzer type %s, got nil", analyzerType)
			}

			if analyzer != nil {
				t.Errorf("Expected nil analyzer for unsupported type %s, got %T", analyzerType, analyzer)
			}
		})
	}
}

// TestNewAnalyzer tests the convenience function for creating analyzers
func TestNewAnalyzer(t *testing.T) {
	// Test Poetry
	poetryAnalyzer, err := NewAnalyzer("poetry")
	if err != nil {
		t.Fatalf("NewAnalyzer failed for poetry: %v", err)
	}
	if poetryAnalyzer == nil {
		t.Fatal("NewAnalyzer returned nil for poetry")
	}
	if _, ok := poetryAnalyzer.(*PoetryAnalyzer); !ok {
		t.Errorf("Expected *PoetryAnalyzer, got %T", poetryAnalyzer)
	}

	// Test Pipfile
	pipfileAnalyzer, err := NewAnalyzer("pipfile")
	if err != nil {
		t.Fatalf("NewAnalyzer failed for pipfile: %v", err)
	}
	if pipfileAnalyzer == nil {
		t.Fatal("NewAnalyzer returned nil for pipfile")
	}
	if _, ok := pipfileAnalyzer.(*PipfileAnalyzer); !ok {
		t.Errorf("Expected *PipfileAnalyzer, got %T", pipfileAnalyzer)
	}

	// Test UvLock
	uvlockAnalyzer, err := NewAnalyzer("uvlock")
	if err != nil {
		t.Fatalf("NewAnalyzer failed for uvlock: %v", err)
	}
	if uvlockAnalyzer == nil {
		t.Fatal("NewAnalyzer returned nil for uvlock")
	}
	if _, ok := uvlockAnalyzer.(*UvLockAnalyzer); !ok {
		t.Errorf("Expected *UvLockAnalyzer, got %T", uvlockAnalyzer)
	}

	// Test unsupported analyzer
	_, err = NewAnalyzer("unsupported")
	if err == nil {
		t.Error("Expected error for unsupported analyzer, got nil")
	}
}

// TestSupportedAnalyzers verifies the list of supported analyzers
func TestSupportedAnalyzers(t *testing.T) {
	analyzers := SupportedAnalyzers()

	if len(analyzers) == 0 {
		t.Fatal("SupportedAnalyzers returned empty list")
	}

	// Check that expected analyzers are in the list
	expectedAnalyzers := map[string]bool{
		"poetry":  false,
		"pipfile": false,
		"uvlock":  false,
	}

	for _, analyzer := range analyzers {
		if _, exists := expectedAnalyzers[analyzer]; exists {
			expectedAnalyzers[analyzer] = true
		}
	}

	// Verify all expected analyzers were found
	for analyzer, found := range expectedAnalyzers {
		if !found {
			t.Errorf("Expected analyzer %s not found in SupportedAnalyzers()", analyzer)
		}
	}
}

// TestAnalyzerTypeConstants verifies analyzer type constants
func TestAnalyzerTypeConstants(t *testing.T) {
	if AnalyzerPoetry != "poetry" {
		t.Errorf("Expected AnalyzerPoetry to be 'poetry', got '%s'", AnalyzerPoetry)
	}
	if AnalyzerPipfile != "pipfile" {
		t.Errorf("Expected AnalyzerPipfile to be 'pipfile', got '%s'", AnalyzerPipfile)
	}
	if AnalyzerUvLock != "uvlock" {
		t.Errorf("Expected AnalyzerUvLock to be 'uvlock', got '%s'", AnalyzerUvLock)
	}
}

// TestPoetryAnalyzerName tests the Name method
func TestPoetryAnalyzerName(t *testing.T) {
	analyzer := NewPoetryAnalyzer()

	if analyzer.Name() != "poetry" {
		t.Errorf("Expected name 'poetry', got '%s'", analyzer.Name())
	}
}

// TestPipfileAnalyzerName tests the Name method
func TestPipfileAnalyzerName(t *testing.T) {
	analyzer := NewPipfileAnalyzer()

	if analyzer.Name() != "pipfile" {
		t.Errorf("Expected name 'pipfile', got '%s'", analyzer.Name())
	}
}

// TestUvLockAnalyzerName tests the Name method
func TestUvLockAnalyzerName(t *testing.T) {
	analyzer := NewUvLockAnalyzer()

	if analyzer.Name() != "uvlock" {
		t.Errorf("Expected name 'uvlock', got '%s'", analyzer.Name())
	}
}
