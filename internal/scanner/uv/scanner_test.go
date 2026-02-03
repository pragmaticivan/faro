package uv

import (
	"encoding/json"
	"testing"

	"github.com/pragmaticivan/faro/internal/scanner"
)

func TestGetUpdates(t *testing.T) {
	// Mock uv pip list --outdated output
	mockOutdated := uvOutdated{
		{Name: "requests", Version: "2.28.0", Latest: "2.31.0"},
		{Name: "flask", Version: "2.2.0", Latest: "3.0.0"},
		{Name: "django", Version: "4.0.0", Latest: "5.0.0"},
	}
	outdatedBytes, _ := json.Marshal(mockOutdated)

	s := &Scanner{
		workDir: ".",
		runUvCmd: func(args ...string) ([]byte, error) {
			return outdatedBytes, nil
		},
	}

	// Test Case 1: Default options
	opts := scanner.Options{
		IncludeAll: false,
	}

	modules, err := s.GetUpdates(opts)
	if err != nil {
		t.Fatalf("GetUpdates failed: %v", err)
	}

	if len(modules) != 3 {
		t.Errorf("expected 3 modules, got %d", len(modules))
	}

	// Verify packages
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
		}
	}

	if !foundRequests {
		t.Error("requests not found")
	}
	if !foundFlask {
		t.Error("flask not found")
	}
}

func TestGetUpdates_Filter(t *testing.T) {
	mockOutdated := uvOutdated{
		{Name: "requests", Version: "2.28.0", Latest: "2.31.0"},
		{Name: "django", Version: "4.0.0", Latest: "5.0.0"},
		{Name: "flask", Version: "2.2.0", Latest: "3.0.0"},
	}
	outdatedBytes, _ := json.Marshal(mockOutdated)

	s := &Scanner{
		workDir: ".",
		runUvCmd: func(args ...string) ([]byte, error) {
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

func TestGetUpdates_CaseInsensitiveFilter(t *testing.T) {
	mockOutdated := uvOutdated{
		{Name: "Django", Version: "4.0.0", Latest: "5.0.0"},
		{Name: "Flask", Version: "2.2.0", Latest: "3.0.0"},
	}
	outdatedBytes, _ := json.Marshal(mockOutdated)

	s := &Scanner{
		workDir: ".",
		runUvCmd: func(args ...string) ([]byte, error) {
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

	if modules[0].Name != "Django" {
		t.Errorf("expected Django, got %s", modules[0].Name)
	}
}

func TestGetUpdates_EmptyOutdated(t *testing.T) {
	mockOutdated := uvOutdated{}
	outdatedBytes, _ := json.Marshal(mockOutdated)

	s := &Scanner{
		workDir: ".",
		runUvCmd: func(args ...string) ([]byte, error) {
			return outdatedBytes, nil
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
	mockPackages := []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}{
		{Name: "requests", Version: "2.31.0"},
		{Name: "flask", Version: "3.0.0"},
		{Name: "django", Version: "5.0.0"},
	}
	packagesBytes, _ := json.Marshal(mockPackages)

	s := &Scanner{
		workDir: ".",
		runUvCmd: func(args ...string) ([]byte, error) {
			return packagesBytes, nil
		},
	}

	idx, err := s.GetDependencyIndex()
	if err != nil {
		t.Fatalf("GetDependencyIndex failed: %v", err)
	}

	expectedPackages := []string{"requests", "flask", "django"}
	for _, name := range expectedPackages {
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
}
