package pnpm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pragmaticivan/faro/internal/scanner"
)

func TestGetUpdates(t *testing.T) {
	// Create temp directory with package.json
	tmpDir := t.TempDir()
	mockPkgJSON := packageJSON{
		Dependencies: map[string]string{
			"react": "^18.0.0",
			"axios": "^1.0.0",
		},
		DevDependencies: map[string]string{
			"vitest": "^0.34.0",
		},
	}
	pkgJSONBytes, _ := json.Marshal(mockPkgJSON)
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), pkgJSONBytes, 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	// Mock pnpm outdated output
	mockOutdated := pnpmOutdated{
		"react": {
			Current: "18.0.0",
			Latest:  "18.2.0",
			Wanted:  "18.2.0",
		},
		"axios": {
			Current: "1.0.0",
			Latest:  "1.6.0",
			Wanted:  "1.6.0",
		},
		"vitest": {
			Current: "0.34.0",
			Latest:  "1.0.0",
			Wanted:  "0.34.6",
		},
		"@types/node": { // transitive
			Current: "18.0.0",
			Latest:  "20.0.0",
			Wanted:  "20.0.0",
		},
	}
	outdatedBytes, _ := json.Marshal(mockOutdated)

	s := &Scanner{
		workDir: tmpDir,
		runPnpmOutdated: func() ([]byte, error) {
			return outdatedBytes, nil
		},
	}

	// Test Case 1: Default options (exclude dev dependencies, include transitive)
	opts := scanner.Options{
		IncludeAll: false,
	}

	modules, err := s.GetUpdates(opts)
	if err != nil {
		t.Fatalf("GetUpdates failed: %v", err)
	}

	// Should have react, axios, and @types/node (transitive), but NOT vitest (dev)
	if len(modules) != 3 {
		t.Errorf("expected 3 modules, got %d", len(modules))
	}

	foundReact := false
	foundAxios := false
	for _, m := range modules {
		if m.Name == "react" {
			foundReact = true
			if !m.Direct {
				t.Error("expected react to be Direct=true")
			}
			if m.DependencyType != "dependencies" {
				t.Errorf("expected dependency type 'dependencies', got %s", m.DependencyType)
			}
			if m.Version != "18.0.0" {
				t.Errorf("expected version 18.0.0, got %s", m.Version)
			}
			if m.Update.Version != "18.2.0" {
				t.Errorf("expected update version 18.2.0, got %s", m.Update.Version)
			}
		}
		if m.Name == "axios" {
			foundAxios = true
			if !m.Direct {
				t.Error("expected axios to be Direct=true")
			}
		}
		if m.Name == "vitest" {
			t.Error("vitest should not be included when IncludeAll=false")
		}
		if m.Name == "@types/node" {
			// @types/node is transitive and should be included
			if m.Direct {
				t.Error("expected @types/node to be Direct=false")
			}
			if m.DependencyType != "transitive" {
				t.Errorf("expected dependency type 'transitive', got %s", m.DependencyType)
			}
		}
	}

	if !foundReact {
		t.Error("react not found")
	}
	if !foundAxios {
		t.Error("axios not found")
	}

	// Test Case 2: IncludeAll = true
	opts.IncludeAll = true
	modules, err = s.GetUpdates(opts)
	if err != nil {
		t.Fatalf("GetUpdates(IncludeAll) failed: %v", err)
	}

	if len(modules) != 4 {
		t.Errorf("expected 4 modules with IncludeAll, got %d", len(modules))
	}

	// Verify vitest is included with correct type
	foundVitest := false
	for _, m := range modules {
		if m.Name == "vitest" {
			foundVitest = true
			if !m.Direct {
				t.Error("expected vitest to be Direct=true")
			}
			if m.DependencyType != "devDependencies" {
				t.Errorf("expected dependency type 'devDependencies', got %s", m.DependencyType)
			}
		}
	}
	if !foundVitest {
		t.Error("vitest not found with IncludeAll=true")
	}
}

func TestGetUpdates_Filter(t *testing.T) {
	tmpDir := t.TempDir()
	mockPkgJSON := packageJSON{
		Dependencies: map[string]string{
			"react":     "^18.0.0",
			"react-dom": "^18.0.0",
			"vue":       "^3.0.0",
		},
	}
	pkgJSONBytes, _ := json.Marshal(mockPkgJSON)
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), pkgJSONBytes, 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	mockOutdated := pnpmOutdated{
		"react":     {Current: "18.0.0", Latest: "18.2.0", Wanted: "18.2.0"},
		"react-dom": {Current: "18.0.0", Latest: "18.2.0", Wanted: "18.2.0"},
		"vue":       {Current: "3.0.0", Latest: "3.3.0", Wanted: "3.3.0"},
	}
	outdatedBytes, _ := json.Marshal(mockOutdated)

	s := &Scanner{
		workDir: tmpDir,
		runPnpmOutdated: func() ([]byte, error) {
			return outdatedBytes, nil
		},
	}

	opts := scanner.Options{
		Filter: "react",
	}

	modules, err := s.GetUpdates(opts)
	if err != nil {
		t.Fatalf("GetUpdates with filter failed: %v", err)
	}

	if len(modules) != 2 {
		t.Errorf("expected 2 modules (react and react-dom), got %d", len(modules))
	}

	for _, m := range modules {
		if m.Name != "react" && m.Name != "react-dom" {
			t.Errorf("unexpected module %s with filter 'react'", m.Name)
		}
	}
}

func TestGetUpdates_EmptyOutdated(t *testing.T) {
	tmpDir := t.TempDir()
	mockPkgJSON := packageJSON{
		Dependencies: map[string]string{
			"react": "^18.2.0",
		},
	}
	pkgJSONBytes, _ := json.Marshal(mockPkgJSON)
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), pkgJSONBytes, 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	s := &Scanner{
		workDir: tmpDir,
		runPnpmOutdated: func() ([]byte, error) {
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
	mockPkgJSON := packageJSON{
		Dependencies: map[string]string{
			"react": "^18.0.0",
			"axios": "^1.0.0",
		},
		DevDependencies: map[string]string{
			"vitest":     "^0.34.0",
			"typescript": "^5.0.0",
		},
	}
	pkgJSONBytes, _ := json.Marshal(mockPkgJSON)
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), pkgJSONBytes, 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	s := NewScanner(tmpDir)

	idx, err := s.GetDependencyIndex()
	if err != nil {
		t.Fatalf("GetDependencyIndex failed: %v", err)
	}

	// Check production dependencies
	for _, name := range []string{"react", "axios"} {
		info, ok := idx[name]
		if !ok {
			t.Errorf("expected %s in dependency index", name)
		} else {
			if !info.Direct {
				t.Errorf("expected %s to be Direct", name)
			}
			if info.Type != "dependencies" {
				t.Errorf("expected %s type to be 'dependencies', got %s", name, info.Type)
			}
		}
	}

	// Check dev dependencies
	for _, name := range []string{"vitest", "typescript"} {
		info, ok := idx[name]
		if !ok {
			t.Errorf("expected %s in dependency index", name)
		} else {
			if !info.Direct {
				t.Errorf("expected %s to be Direct", name)
			}
			if info.Type != "devDependencies" {
				t.Errorf("expected %s type to be 'devDependencies', got %s", name, info.Type)
			}
		}
	}
}
