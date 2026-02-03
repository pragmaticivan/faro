package yarn

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
			"jest": "^29.0.0",
		},
	}
	pkgJSONBytes, _ := json.Marshal(mockPkgJSON)
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), pkgJSONBytes, 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	// Mock yarn outdated output (multiple JSON lines)
	mockOutput := yarnOutdated{
		Type: "table",
		Data: yarnOutdatedTable{
			Head: []string{"Package", "Current", "Wanted", "Latest", "Package Type"},
			Body: [][]string{
				{"react", "18.0.0", "18.2.0", "18.2.0", "dependencies"},
				{"axios", "1.0.0", "1.6.0", "1.6.0", "dependencies"},
				{"jest", "29.0.0", "29.7.0", "29.7.0", "devDependencies"},
				{"@types/node", "18.0.0", "20.0.0", "20.0.0", "dependencies"}, // transitive
			},
		},
	}
	mockOutputLine, _ := json.Marshal(mockOutput)
	mockOutputBytes := append(mockOutputLine, '\n')

	s := &Scanner{
		workDir: tmpDir,
		runYarnOutdated: func() ([]byte, error) {
			return mockOutputBytes, nil
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

	// Should have react and axios, but NOT jest or @types/node
	if len(modules) != 2 {
		t.Errorf("expected 2 modules, got %d", len(modules))
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
		if m.Name == "jest" {
			t.Error("jest should not be included when IncludeAll=false")
		}
		if m.Name == "@types/node" {
			t.Error("@types/node should not be included (transitive)")
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

	// Verify jest is included with correct type
	foundJest := false
	for _, m := range modules {
		if m.Name == "jest" {
			foundJest = true
			if !m.Direct {
				t.Error("expected jest to be Direct=true")
			}
			if m.DependencyType != "devDependencies" {
				t.Errorf("expected dependency type 'devDependencies', got %s", m.DependencyType)
			}
		}
	}
	if !foundJest {
		t.Error("jest not found with IncludeAll=true")
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

	mockOutput := yarnOutdated{
		Type: "table",
		Data: yarnOutdatedTable{
			Body: [][]string{
				{"react", "18.0.0", "18.2.0", "18.2.0", "dependencies"},
				{"react-dom", "18.0.0", "18.2.0", "18.2.0", "dependencies"},
				{"vue", "3.0.0", "3.3.0", "3.3.0", "dependencies"},
			},
		},
	}
	mockOutputLine, _ := json.Marshal(mockOutput)

	s := &Scanner{
		workDir: tmpDir,
		runYarnOutdated: func() ([]byte, error) {
			return append(mockOutputLine, '\n'), nil
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
		runYarnOutdated: func() ([]byte, error) {
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
			"jest":       "^29.0.0",
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
	for _, name := range []string{"jest", "typescript"} {
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

func TestGetUpdates_InsufficientFields(t *testing.T) {
	tmpDir := t.TempDir()
	mockPkgJSON := packageJSON{
		Dependencies: map[string]string{
			"react": "^18.0.0",
		},
	}
	pkgJSONBytes, _ := json.Marshal(mockPkgJSON)
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), pkgJSONBytes, 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	// Mock output with insufficient fields (less than 4 columns)
	mockOutput := yarnOutdated{
		Type: "table",
		Data: yarnOutdatedTable{
			Body: [][]string{
				{"react", "18.0.0"}, // Only 2 fields, should be skipped
			},
		},
	}
	mockOutputLine, _ := json.Marshal(mockOutput)

	s := &Scanner{
		workDir: tmpDir,
		runYarnOutdated: func() ([]byte, error) {
			return append(mockOutputLine, '\n'), nil
		},
	}

	opts := scanner.Options{}

	modules, err := s.GetUpdates(opts)
	if err != nil {
		t.Fatalf("GetUpdates failed: %v", err)
	}

	if len(modules) != 0 {
		t.Errorf("expected 0 modules (invalid row), got %d", len(modules))
	}
}
