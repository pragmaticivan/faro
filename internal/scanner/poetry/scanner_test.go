package poetry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pragmaticivan/faro/internal/scanner"
)

func TestGetUpdates(t *testing.T) {
	// Create temp directory with pyproject.toml
	tmpDir := t.TempDir()
	pyprojectToml := `[tool.poetry]
name = "test-project"
version = "0.1.0"

[tool.poetry.dependencies]
python = "^3.9"
requests = "^2.28.0"
flask = "^2.2.0"

[tool.poetry.dev-dependencies]
pytest = "^7.0.0"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte(pyprojectToml), 0644); err != nil {
		t.Fatalf("failed to write pyproject.toml: %v", err)
	}

	// Mock poetry show --outdated output
	mockOutput := `requests 2.28.0 2.31.0 HTTP library
flask    2.2.0  3.0.0  Web framework
pytest   7.0.0  7.4.0  Testing framework
`

	s := &Scanner{
		workDir: tmpDir,
		runPoetryCmd: func(_ ...string) ([]byte, error) {
			return []byte(mockOutput), nil
		},
	}

	// Test Case 1: Default options (exclude dev dependencies)
	opts := scanner.Options{
		IncludeAll: false,
	}

	modules, err := s.GetUpdates(opts)
	if err != nil {
		t.Fatalf("GetUpdates failed: %v", err)
	}

	// Should have requests and flask, but NOT pytest (dev dependency)
	if len(modules) != 2 {
		t.Errorf("expected 2 modules, got %d", len(modules))
	}

	foundRequests := false
	foundFlask := false
	for _, m := range modules {
		if m.Name == "requests" {
			foundRequests = true
			if !m.Direct {
				t.Error("expected requests to be Direct=true")
			}
			if m.DependencyType != "main" {
				t.Errorf("expected dependency type 'main', got %s", m.DependencyType)
			}
			if m.Version != "2.28.0" {
				t.Errorf("expected version 2.28.0, got %s", m.Version)
			}
			if m.Update.Version != "2.31.0" {
				t.Errorf("expected update version 2.31.0, got %s", m.Update.Version)
			}
		}
		if m.Name == "flask" {
			foundFlask = true
			if !m.Direct {
				t.Error("expected flask to be Direct=true")
			}
		}
		if m.Name == "pytest" {
			t.Error("pytest should not be included when IncludeAll=false")
		}
	}

	if !foundRequests {
		t.Error("requests not found")
	}
	if !foundFlask {
		t.Error("flask not found")
	}

	// Test Case 2: IncludeAll = true
	opts.IncludeAll = true
	modules, err = s.GetUpdates(opts)
	if err != nil {
		t.Fatalf("GetUpdates(IncludeAll) failed: %v", err)
	}

	if len(modules) != 3 {
		t.Errorf("expected 3 modules with IncludeAll, got %d", len(modules))
	}

	// Verify pytest is included with correct type
	foundPytest := false
	for _, m := range modules {
		if m.Name == "pytest" {
			foundPytest = true
			if !m.Direct {
				t.Error("expected pytest to be Direct=true")
			}
			if m.DependencyType != "dev" {
				t.Errorf("expected dependency type 'dev', got %s", m.DependencyType)
			}
		}
	}
	if !foundPytest {
		t.Error("pytest not found with IncludeAll=true")
	}
}

func TestGetUpdates_Filter(t *testing.T) {
	tmpDir := t.TempDir()
	pyprojectToml := `[tool.poetry]
name = "test-project"

[tool.poetry.dependencies]
django = "^4.0.0"
flask = "^2.2.0"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte(pyprojectToml), 0644); err != nil {
		t.Fatalf("failed to write pyproject.toml: %v", err)
	}

	mockOutput := `django 4.0.0 5.0.0 Web framework
flask  2.2.0 3.0.0 Web framework
`

	s := &Scanner{
		workDir: tmpDir,
		runPoetryCmd: func(_ ...string) ([]byte, error) {
			return []byte(mockOutput), nil
		},
	}

	opts := scanner.Options{
		Filter: "django",
	}

	modules, err := s.GetUpdates(opts)
	if err != nil {
		t.Fatalf("GetUpdates with filter failed: %v", err)
	}

	if len(modules) != 1 {
		t.Errorf("expected 1 module, got %d", len(modules))
	}

	if modules[0].Name != "django" {
		t.Errorf("expected django, got %s", modules[0].Name)
	}
}

func TestGetUpdates_NoOutdated(t *testing.T) {
	tmpDir := t.TempDir()
	pyprojectToml := `[tool.poetry]
name = "test-project"

[tool.poetry.dependencies]
requests = "^2.31.0"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte(pyprojectToml), 0644); err != nil {
		t.Fatalf("failed to write pyproject.toml: %v", err)
	}

	s := &Scanner{
		workDir: tmpDir,
		runPoetryCmd: func(_ ...string) ([]byte, error) {
			// Simulate error when no outdated packages
			return []byte{}, nil
		},
	}

	opts := scanner.Options{}

	modules, err := s.GetUpdates(opts)
	if err != nil {
		t.Fatalf("GetUpdates failed: %v", err)
	}

	if len(modules) != 0 {
		t.Errorf("expected 0 modules, got %d", len(modules))
	}
}

func TestGetDependencyIndex(t *testing.T) {
	tmpDir := t.TempDir()
	pyprojectToml := `[tool.poetry]
name = "test-project"

[tool.poetry.dependencies]
python = "^3.9"
requests = "^2.28.0"
flask = "^2.2.0"

[tool.poetry.dev-dependencies]
pytest = "^7.0.0"
black = "^23.0.0"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte(pyprojectToml), 0644); err != nil {
		t.Fatalf("failed to write pyproject.toml: %v", err)
	}

	s := NewScanner(tmpDir)

	idx, err := s.GetDependencyIndex()
	if err != nil {
		t.Fatalf("GetDependencyIndex failed: %v", err)
	}

	// Check main dependencies (excluding python)
	for _, name := range []string{"requests", "flask"} {
		info, ok := idx[name]
		if !ok {
			t.Errorf("expected %s in dependency index", name)
		} else {
			if !info.Direct {
				t.Errorf("expected %s to be Direct", name)
			}
			if info.Type != "main" {
				t.Errorf("expected %s type to be 'main', got %s", name, info.Type)
			}
		}
	}

	// Check dev dependencies
	for _, name := range []string{"pytest", "black"} {
		info, ok := idx[name]
		if !ok {
			t.Errorf("expected %s in dependency index", name)
		} else {
			if !info.Direct {
				t.Errorf("expected %s to be Direct", name)
			}
			if info.Type != "dev" {
				t.Errorf("expected %s type to be 'dev', got %s", name, info.Type)
			}
		}
	}

	// python should be excluded
	if _, ok := idx["python"]; ok {
		t.Error("python should not be in dependency index")
	}
}
