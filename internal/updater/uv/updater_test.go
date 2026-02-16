package uv

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

	if updater.runUvCmd == nil {
		t.Error("runUvCmd function should not be nil")
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
		{Name: "requests", Update: &scanner.UpdateInfo{Version: "2.28.1"}},
		{Name: "flask", Update: &scanner.UpdateInfo{Version: "2.2.0"}},
	}

	var capturedCommands []string
	updater := &Updater{
		workDir: "/test/dir",
		runUvCmd: func(args ...string) ([]byte, error) {
			capturedCommands = append(capturedCommands, "uv "+strings.Join(args, " "))
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

	expectedFirst := "uv pip install requests==2.28.1"
	if capturedCommands[0] != expectedFirst {
		t.Errorf("expected command %q, got %q", expectedFirst, capturedCommands[0])
	}

	expectedSecond := "uv pip install flask==2.2.0"
	if capturedCommands[1] != expectedSecond {
		t.Errorf("expected command %q, got %q", expectedSecond, capturedCommands[1])
	}
}

func TestUpdatePackages_WithoutVersion(t *testing.T) {
	modules := []scanner.Module{
		{Name: "requests"},
	}

	var capturedCommands []string
	updater := &Updater{
		workDir: "/test/dir",
		runUvCmd: func(args ...string) ([]byte, error) {
			capturedCommands = append(capturedCommands, "uv "+strings.Join(args, " "))
			return []byte("success"), nil
		},
	}

	err := updater.UpdatePackages(modules)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := "uv pip install requests"
	if capturedCommands[0] != expected {
		t.Errorf("expected command %q, got %q", expected, capturedCommands[0])
	}
}

func TestUpdatePackages_InstallFails(t *testing.T) {
	modules := []scanner.Module{
		{Name: "requests", Update: &scanner.UpdateInfo{Version: "2.28.1"}},
	}

	updater := &Updater{
		workDir: "/test/dir",
		runUvCmd: func(args ...string) ([]byte, error) {
			return []byte("uv pip install failed"), errors.New("exit 1")
		},
	}

	err := updater.UpdatePackages(modules)
	if err == nil {
		t.Fatal("expected error when uv pip install fails")
	}

	if !strings.Contains(err.Error(), "uv pip install failed") {
		t.Errorf("expected error to contain 'uv pip install failed', got %v", err)
	}
}

func TestUpdateSinglePackage(t *testing.T) {
	module := scanner.Module{
		Name:   "requests",
		Update: &scanner.UpdateInfo{Version: "2.28.1"},
	}

	var capturedCommands []string
	updater := &Updater{
		workDir: "/test/dir",
		runUvCmd: func(args ...string) ([]byte, error) {
			capturedCommands = append(capturedCommands, "uv "+strings.Join(args, " "))
			return []byte("success"), nil
		},
	}

	err := updater.UpdateSinglePackage(module)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := "uv pip install requests==2.28.1"
	if capturedCommands[0] != expected {
		t.Errorf("expected command %q, got %q", expected, capturedCommands[0])
	}
}
