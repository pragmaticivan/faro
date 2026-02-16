package pnpm

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
		{Name: "express", Version: "4.18.0", DependencyType: "dependencies", Update: &scanner.UpdateInfo{Version: "4.18.2"}},
		{Name: "jest", Version: "29.0.0", DependencyType: "devDependencies", Update: &scanner.UpdateInfo{Version: "29.3.1"}},
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

	expectedProd := "pnpm add express@4.18.2"
	if capturedCommands[0] != expectedProd {
		t.Errorf("expected command %q, got %q", expectedProd, capturedCommands[0])
	}

	expectedDev := "pnpm add --save-dev jest@29.3.1"
	if capturedCommands[1] != expectedDev {
		t.Errorf("expected command %q, got %q", expectedDev, capturedCommands[1])
	}
}

func TestUpdatePackages_ProductionOnly(t *testing.T) {
	modules := []scanner.Module{
		{Name: "express", DependencyType: "dependencies", Update: &scanner.UpdateInfo{Version: "4.18.2"}},
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

	if len(capturedCommands) != 1 {
		t.Fatalf("expected 1 command, got %d: %v", len(capturedCommands), capturedCommands)
	}

	expected := "pnpm add express@4.18.2"
	if capturedCommands[0] != expected {
		t.Errorf("expected command %q, got %q", expected, capturedCommands[0])
	}
}

func TestUpdatePackages_DevOnly(t *testing.T) {
	modules := []scanner.Module{
		{Name: "jest", DependencyType: "devDependencies", Update: &scanner.UpdateInfo{Version: "29.3.1"}},
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

	if len(capturedCommands) != 1 {
		t.Fatalf("expected 1 command, got %d: %v", len(capturedCommands), capturedCommands)
	}

	expected := "pnpm add --save-dev jest@29.3.1"
	if capturedCommands[0] != expected {
		t.Errorf("expected command %q, got %q", expected, capturedCommands[0])
	}
}

func TestUpdatePackages_AddFails(t *testing.T) {
	modules := []scanner.Module{
		{Name: "express", DependencyType: "dependencies", Update: &scanner.UpdateInfo{Version: "4.18.2"}},
	}

	updater := &Updater{
		workDir: "/test/dir",
		runCmd: func(name string, args ...string) ([]byte, error) {
			return []byte("pnpm add failed"), errors.New("exit 1")
		},
	}

	err := updater.UpdatePackages(modules)
	if err == nil {
		t.Fatal("expected error when pnpm add fails")
	}

	if !strings.Contains(err.Error(), "pnpm add failed") {
		t.Errorf("expected error to contain 'pnpm add failed', got %v", err)
	}
}

func TestUpdatePackages_AddDevFails(t *testing.T) {
	modules := []scanner.Module{
		{Name: "jest", DependencyType: "devDependencies", Update: &scanner.UpdateInfo{Version: "29.3.1"}},
	}

	updater := &Updater{
		workDir: "/test/dir",
		runCmd: func(name string, args ...string) ([]byte, error) {
			return []byte("pnpm add --save-dev failed"), errors.New("exit 1")
		},
	}

	err := updater.UpdatePackages(modules)
	if err == nil {
		t.Fatal("expected error when pnpm add --save-dev fails")
	}

	if !strings.Contains(err.Error(), "pnpm add --save-dev failed") {
		t.Errorf("expected error to contain 'pnpm add --save-dev failed', got %v", err)
	}
}
