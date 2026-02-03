package pip

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pragmaticivan/faro/internal/scanner"
)

func TestGetUpdates(t *testing.T) {
	// Create temp directory with requirements.txt
	tmpDir := t.TempDir()
	requirementsTxt := `requests==2.28.0
flask==2.2.0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte(requirementsTxt), 0644); err != nil {
		t.Fatalf("failed to write requirements.txt: %v", err)
	}

	// Mock pip outdated output
	mockOutdated := pipOutdated{
		{Name: "requests", Version: "2.28.0", Latest: "2.31.0", Type: "wheel"},
		{Name: "flask", Version: "2.2.0", Latest: "3.0.0", Type: "wheel"},
		{Name: "werkzeug", Version: "2.2.0", Latest: "3.0.0", Type: "wheel"}, // transitive
	}
	outdatedBytes, _ := json.Marshal(mockOutdated)

	s := &Scanner{
		workDir: tmpDir,
		runPipCmd: func(args ...string) ([]byte, error) {
			return outdatedBytes, nil
		},
	}

	// Test Case 1: Default options (only direct dependencies)
	opts := scanner.Options{
		IncludeAll: false,
	}

	modules, err := s.GetUpdates(opts)
	if err != nil {
		t.Fatalf("GetUpdates failed: %v", err)
	}

	if len(modules) != 2 {
		t.Errorf("expected 2 modules, got %d", len(modules))
	}

	// Verify direct dependencies
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
		if m.Name == "werkzeug" {
			t.Error("werkzeug should not be included when IncludeAll=false")
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

	// Verify werkzeug is now included
	foundWerkzeug := false
	for _, m := range modules {
		if m.Name == "werkzeug" {
			foundWerkzeug = true
			if m.Direct {
				t.Error("expected werkzeug to be Direct=false")
			}
			if m.DependencyType != "transitive" {
				t.Errorf("expected dependency type 'transitive', got %s", m.DependencyType)
			}
		}
	}
	if !foundWerkzeug {
		t.Error("werkzeug not found with IncludeAll=true")
	}
}

func TestGetUpdates_Filter(t *testing.T) {
	tmpDir := t.TempDir()
	requirementsTxt := `requests==2.28.0
django==4.0.0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte(requirementsTxt), 0644); err != nil {
		t.Fatalf("failed to write requirements.txt: %v", err)
	}

	mockOutdated := pipOutdated{
		{Name: "requests", Version: "2.28.0", Latest: "2.31.0", Type: "wheel"},
		{Name: "django", Version: "4.0.0", Latest: "5.0.0", Type: "wheel"},
	}
	outdatedBytes, _ := json.Marshal(mockOutdated)

	s := &Scanner{
		workDir: tmpDir,
		runPipCmd: func(args ...string) ([]byte, error) {
			return outdatedBytes, nil
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

func TestGetUpdates_EmptyRequirements(t *testing.T) {
	tmpDir := t.TempDir()
	requirementsTxt := ``
	if err := os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte(requirementsTxt), 0644); err != nil {
		t.Fatalf("failed to write requirements.txt: %v", err)
	}

	mockOutdated := pipOutdated{
		{Name: "pip", Version: "21.0.0", Latest: "23.0.0", Type: "wheel"},
	}
	outdatedBytes, _ := json.Marshal(mockOutdated)

	s := &Scanner{
		workDir: tmpDir,
		runPipCmd: func(args ...string) ([]byte, error) {
			return outdatedBytes, nil
		},
	}

	opts := scanner.Options{
		IncludeAll: true,
	}

	modules, err := s.GetUpdates(opts)
	if err != nil {
		t.Fatalf("GetUpdates failed: %v", err)
	}

	if len(modules) != 1 {
		t.Errorf("expected 1 module, got %d", len(modules))
	}

	if modules[0].Name != "pip" {
		t.Errorf("expected pip, got %s", modules[0].Name)
	}
}

func TestGetDependencyIndex(t *testing.T) {
	tmpDir := t.TempDir()
	requirementsTxt := `requests==2.28.0
flask==2.2.0
# A comment
django>=4.0.0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte(requirementsTxt), 0644); err != nil {
		t.Fatalf("failed to write requirements.txt: %v", err)
	}

	s := NewScanner(tmpDir)

	idx, err := s.GetDependencyIndex()
	if err != nil {
		t.Fatalf("GetDependencyIndex failed: %v", err)
	}

	expectedDeps := []string{"requests", "flask", "django"}
	for _, dep := range expectedDeps {
		if info, ok := idx[dep]; !ok {
			t.Errorf("expected %s in dependency index", dep)
		} else {
			if !info.Direct {
				t.Errorf("expected %s to be Direct", dep)
			}
			if info.Type != "main" {
				t.Errorf("expected %s type to be 'main', got %s", dep, info.Type)
			}
		}
	}
}
