package poetry

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

	if updater.runPoetryCmd == nil {
		t.Error("runPoetryCmd function should not be nil")
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
		{Name: "requests", DependencyType: "main", Update: &scanner.UpdateInfo{Version: "2.28.1"}},
		{Name: "pytest", DependencyType: "dev", Update: &scanner.UpdateInfo{Version: "7.2.0"}},
	}

	var capturedCommands []string
	updater := &Updater{
		workDir: "/test/dir",
		runPoetryCmd: func(args ...string) ([]byte, error) {
			capturedCommands = append(capturedCommands, "poetry "+strings.Join(args, " "))
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

	expectedMain := "poetry add requests@2.28.1"
	if capturedCommands[0] != expectedMain {
		t.Errorf("expected command %q, got %q", expectedMain, capturedCommands[0])
	}

	expectedDev := "poetry add --group dev pytest@7.2.0"
	if capturedCommands[1] != expectedDev {
		t.Errorf("expected command %q, got %q", expectedDev, capturedCommands[1])
	}
}

func TestUpdatePackages_MainOnly(t *testing.T) {
	modules := []scanner.Module{
		{Name: "requests", DependencyType: "main", Update: &scanner.UpdateInfo{Version: "2.28.1"}},
	}

	var capturedCommands []string
	updater := &Updater{
		workDir: "/test/dir",
		runPoetryCmd: func(args ...string) ([]byte, error) {
			capturedCommands = append(capturedCommands, "poetry "+strings.Join(args, " "))
			return []byte("success"), nil
		},
	}

	err := updater.UpdatePackages(modules)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(capturedCommands) != 1 {
		t.Fatalf("expected 1 command, got %d: %v", len(capturedCommands), capturedCommands)
	}

	expected := "poetry add requests@2.28.1"
	if capturedCommands[0] != expected {
		t.Errorf("expected command %q, got %q", expected, capturedCommands[0])
	}
}

func TestUpdatePackages_DevOnly(t *testing.T) {
	modules := []scanner.Module{
		{Name: "pytest", DependencyType: "dev", Update: &scanner.UpdateInfo{Version: "7.2.0"}},
	}

	var capturedCommands []string
	updater := &Updater{
		workDir: "/test/dir",
		runPoetryCmd: func(args ...string) ([]byte, error) {
			capturedCommands = append(capturedCommands, "poetry "+strings.Join(args, " "))
			return []byte("success"), nil
		},
	}

	err := updater.UpdatePackages(modules)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(capturedCommands) != 1 {
		t.Fatalf("expected 1 command, got %d: %v", len(capturedCommands), capturedCommands)
	}

	expected := "poetry add --group dev pytest@7.2.0"
	if capturedCommands[0] != expected {
		t.Errorf("expected command %q, got %q", expected, capturedCommands[0])
	}
}

func TestUpdatePackages_AddFails(t *testing.T) {
	modules := []scanner.Module{
		{Name: "requests", DependencyType: "main", Update: &scanner.UpdateInfo{Version: "2.28.1"}},
	}

	updater := &Updater{
		workDir: "/test/dir",
		runPoetryCmd: func(args ...string) ([]byte, error) {
			return []byte("poetry add failed"), errors.New("exit 1")
		},
	}

	err := updater.UpdatePackages(modules)
	if err == nil {
		t.Fatal("expected error when poetry add fails")
	}

	if !strings.Contains(err.Error(), "poetry add failed") {
		t.Errorf("expected error to contain 'poetry add failed', got %v", err)
	}
}

func TestUpdateSinglePackage(t *testing.T) {
	module := scanner.Module{
		Name:           "requests",
		DependencyType: "main",
		Update:         &scanner.UpdateInfo{Version: "2.28.1"},
	}

	var capturedCommands []string
	updater := &Updater{
		workDir: "/test/dir",
		runPoetryCmd: func(args ...string) ([]byte, error) {
			capturedCommands = append(capturedCommands, "poetry "+strings.Join(args, " "))
			return []byte("success"), nil
		},
	}

	err := updater.UpdateSinglePackage(module)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := "poetry add requests@2.28.1"
	if capturedCommands[0] != expected {
		t.Errorf("expected command %q, got %q", expected, capturedCommands[0])
	}
}
