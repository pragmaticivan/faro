package gomod

import (
	"errors"
	"strings"
	"testing"

	"github.com/pragmaticivan/faro/internal/scanner"
)

func TestNewUpdater(t *testing.T) {
	workDir := "/test/dir"
	updater := NewUpdater(workDir)

	if updater.workDir != workDir {
		t.Errorf("expected workDir %s, got %s", workDir, updater.workDir)
	}

	if updater.runCmd == nil {
		t.Error("runCmd function should not be nil")
	}
}

func TestUpdatePackages_EmptyModules(t *testing.T) {
	updater := NewUpdater("/test/dir")
	err := updater.UpdatePackages([]scanner.Module{})

	if err != nil {
		t.Errorf("expected no error for empty modules, got %v", err)
	}
}

func TestUpdatePackages_Success(t *testing.T) {
	modules := []scanner.Module{
		{Name: "github.com/pkg/errors", Version: "0.9.0", Update: &scanner.UpdateInfo{Version: "0.9.1"}},
		{Name: "github.com/stretchr/testify", Version: "1.8.0", Update: &scanner.UpdateInfo{Version: "1.8.1"}},
	}

	var capturedCommands []string
	updater := &Updater{
		workDir: "/test/dir",
		runCmd: func(name string, args ...string) ([]byte, error) {
			capturedCommands = append(capturedCommands, name+" "+strings.Join(args, " "))
			return []byte("success"), nil
		},
	}

	err := updater.UpdatePackages(modules)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(capturedCommands) != 2 {
		t.Fatalf("expected 2 commands, got %d: %v", len(capturedCommands), capturedCommands)
	}

	expectedGoGet := "go get github.com/pkg/errors@0.9.1 github.com/stretchr/testify@1.8.1"
	if capturedCommands[0] != expectedGoGet {
		t.Errorf("expected command %q, got %q", expectedGoGet, capturedCommands[0])
	}

	expectedTidy := "go mod tidy"
	if capturedCommands[1] != expectedTidy {
		t.Errorf("expected command %q, got %q", expectedTidy, capturedCommands[1])
	}
}

func TestUpdatePackages_WithoutVersions(t *testing.T) {
	modules := []scanner.Module{
		{Name: "github.com/pkg/errors"},
	}

	var capturedCommands []string
	updater := &Updater{
		workDir: "/test/dir",
		runCmd: func(name string, args ...string) ([]byte, error) {
			capturedCommands = append(capturedCommands, name+" "+strings.Join(args, " "))
			return []byte("success"), nil
		},
	}

	err := updater.UpdatePackages(modules)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedGoGet := "go get github.com/pkg/errors"
	if capturedCommands[0] != expectedGoGet {
		t.Errorf("expected command %q, got %q", expectedGoGet, capturedCommands[0])
	}
}

func TestUpdatePackages_GoGetFails(t *testing.T) {
	modules := []scanner.Module{
		{Name: "github.com/pkg/errors", Update: &scanner.UpdateInfo{Version: "0.9.1"}},
	}

	updater := &Updater{
		workDir: "/test/dir",
		runCmd: func(name string, args ...string) ([]byte, error) {
			if name == "go" && args[0] == "get" {
				return []byte("go get failed"), errors.New("exit 1")
			}
			return []byte("success"), nil
		},
	}

	err := updater.UpdatePackages(modules)
	if err == nil {
		t.Fatal("expected error when go get fails")
	}

	if !strings.Contains(err.Error(), "go get failed") {
		t.Errorf("expected error to contain 'go get failed', got %v", err)
	}
}

func TestUpdatePackages_GoModTidyFails(t *testing.T) {
	modules := []scanner.Module{
		{Name: "github.com/pkg/errors", Update: &scanner.UpdateInfo{Version: "0.9.1"}},
	}

	updater := &Updater{
		workDir: "/test/dir",
		runCmd: func(name string, args ...string) ([]byte, error) {
			if name == "go" && args[0] == "mod" && args[1] == "tidy" {
				return []byte("tidy failed"), errors.New("exit 1")
			}
			return []byte("success"), nil
		},
	}

	err := updater.UpdatePackages(modules)
	if err == nil {
		t.Fatal("expected error when go mod tidy fails")
	}

	if !strings.Contains(err.Error(), "go mod tidy failed") {
		t.Errorf("expected error to contain 'go mod tidy failed', got %v", err)
	}
}

func TestUpdateSinglePackage(t *testing.T) {
	module := scanner.Module{
		Name:   "github.com/pkg/errors",
		Update: &scanner.UpdateInfo{Version: "0.9.1"},
	}

	var capturedCommands []string
	updater := &Updater{
		workDir: "/test/dir",
		runCmd: func(name string, args ...string) ([]byte, error) {
			capturedCommands = append(capturedCommands, name+" "+strings.Join(args, " "))
			return []byte("success"), nil
		},
	}

	err := updater.UpdateSinglePackage(module)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedGoGet := "go get github.com/pkg/errors@0.9.1"
	if capturedCommands[0] != expectedGoGet {
		t.Errorf("expected command %q, got %q", expectedGoGet, capturedCommands[0])
	}
}

func TestBuildGoGetArgs(t *testing.T) {
	tests := []struct {
		name     string
		modules  []scanner.Module
		expected []string
	}{
		{
			name: "modules with versions",
			modules: []scanner.Module{
				{Name: "github.com/pkg/errors", Update: &scanner.UpdateInfo{Version: "0.9.1"}},
				{Name: "github.com/stretchr/testify", Update: &scanner.UpdateInfo{Version: "1.8.1"}},
			},
			expected: []string{"get", "github.com/pkg/errors@0.9.1", "github.com/stretchr/testify@1.8.1"},
		},
		{
			name: "modules without versions",
			modules: []scanner.Module{
				{Name: "github.com/pkg/errors"},
			},
			expected: []string{"get", "github.com/pkg/errors"},
		},
		{
			name: "modules with empty update info",
			modules: []scanner.Module{
				{Name: "github.com/pkg/errors", Update: &scanner.UpdateInfo{}},
			},
			expected: []string{"get", "github.com/pkg/errors"},
		},
		{
			name: "modules with Path fallback",
			modules: []scanner.Module{
				{Path: "github.com/pkg/errors", Update: &scanner.UpdateInfo{Version: "0.9.1"}},
			},
			expected: []string{"get", "github.com/pkg/errors@0.9.1"},
		},
		{
			name: "mixed modules",
			modules: []scanner.Module{
				{Name: "github.com/pkg/errors", Update: &scanner.UpdateInfo{Version: "0.9.1"}},
				{Name: "github.com/stretchr/testify"},
			},
			expected: []string{"get", "github.com/pkg/errors@0.9.1", "github.com/stretchr/testify"},
		},
	}

	updater := NewUpdater("/test/dir")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := updater.buildGoGetArgs(tt.modules)

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d args, got %d: %v", len(tt.expected), len(result), result)
			}

			for i, arg := range result {
				if arg != tt.expected[i] {
					t.Errorf("arg %d: expected %q, got %q", i, tt.expected[i], arg)
				}
			}
		})
	}
}
