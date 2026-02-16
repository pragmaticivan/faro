package npm

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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

	expectedProd := "npm install --save express@4.18.2"
	if capturedCommands[0] != expectedProd {
		t.Errorf("expected command %q, got %q", expectedProd, capturedCommands[0])
	}

	expectedDev := "npm install --save-dev jest@29.3.1"
	if capturedCommands[1] != expectedDev {
		t.Errorf("expected command %q, got %q", expectedDev, capturedCommands[1])
	}
}

func TestUpdatePackages_ProductionOnly(t *testing.T) {
	modules := []scanner.Module{
		{Name: "express", DependencyType: "dependencies", Update: &scanner.UpdateInfo{Version: "4.18.2"}},
		{Name: "lodash", DependencyType: "dependencies", Update: &scanner.UpdateInfo{Version: "4.17.21"}},
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

	expected := "npm install --save express@4.18.2 lodash@4.17.21"
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

	expected := "npm install --save-dev jest@29.3.1"
	if capturedCommands[0] != expected {
		t.Errorf("expected command %q, got %q", expected, capturedCommands[0])
	}
}

func TestUpdatePackages_WithoutVersions(t *testing.T) {
	modules := []scanner.Module{
		{Name: "express", DependencyType: "dependencies"},
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

	expected := "npm install --save express"
	if capturedCommands[0] != expected {
		t.Errorf("expected command %q, got %q", expected, capturedCommands[0])
	}
}

func TestUpdatePackages_ProductionFails(t *testing.T) {
	modules := []scanner.Module{
		{Name: "express", DependencyType: "dependencies", Update: &scanner.UpdateInfo{Version: "4.18.2"}},
	}

	updater := &Updater{
		workDir: "/test/dir",
		runCmd: func(name string, args ...string) ([]byte, error) {
			if args[0] == "install" && args[1] == "--save" {
				return []byte("npm install failed"), errors.New("exit 1")
			}
			return []byte("success"), nil
		},
	}

	err := updater.UpdatePackages(modules)
	if err == nil {
		t.Fatal("expected error when npm install fails")
	}

	if !strings.Contains(err.Error(), "npm install failed") {
		t.Errorf("expected error to contain 'npm install failed', got %v", err)
	}
}

func TestUpdatePackages_DevFails(t *testing.T) {
	modules := []scanner.Module{
		{Name: "jest", DependencyType: "devDependencies", Update: &scanner.UpdateInfo{Version: "29.3.1"}},
	}

	updater := &Updater{
		workDir: "/test/dir",
		runCmd: func(name string, args ...string) ([]byte, error) {
			if args[0] == "install" && args[1] == "--save-dev" {
				return []byte("npm install --save-dev failed"), errors.New("exit 1")
			}
			return []byte("success"), nil
		},
	}

	err := updater.UpdatePackages(modules)
	if err == nil {
		t.Fatal("expected error when npm install --save-dev fails")
	}

	if !strings.Contains(err.Error(), "npm install --save-dev failed") {
		t.Errorf("expected error to contain 'npm install --save-dev failed', got %v", err)
	}
}

func TestUpdateSinglePackage(t *testing.T) {
	module := scanner.Module{
		Name:           "express",
		DependencyType: "dependencies",
		Update:         &scanner.UpdateInfo{Version: "4.18.2"},
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

	expected := "npm install --save express@4.18.2"
	if capturedCommands[0] != expected {
		t.Errorf("expected command %q, got %q", expected, capturedCommands[0])
	}
}

func TestUpdatePackageJSON(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "npm-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	pkgJSON := map[string]interface{}{
		"name":    "test-package",
		"version": "1.0.0",
		"dependencies": map[string]interface{}{
			"express": "^4.18.0",
			"lodash":  "^4.17.20",
		},
		"devDependencies": map[string]interface{}{
			"jest": "^29.0.0",
		},
	}

	data, err := json.MarshalIndent(pkgJSON, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal package.json: %v", err)
	}

	pkgPath := filepath.Join(tempDir, "package.json")
	if err := os.WriteFile(pkgPath, data, 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	modules := []scanner.Module{
		{Name: "express", DependencyType: "dependencies", Update: &scanner.UpdateInfo{Version: "4.18.2"}},
		{Name: "jest", DependencyType: "devDependencies", Update: &scanner.UpdateInfo{Version: "29.3.1"}},
	}

	var capturedCommands []string
	updater := &Updater{
		workDir: tempDir,
		runCmd: func(name string, args ...string) ([]byte, error) {
			capturedCommands = append(capturedCommands, name+" "+strings.Join(args, " "))
			return []byte("success"), nil
		},
	}

	err = updater.UpdatePackageJSON(modules)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	updatedData, err := os.ReadFile(pkgPath)
	if err != nil {
		t.Fatalf("failed to read updated package.json: %v", err)
	}

	var updatedPkg map[string]interface{}
	if err := json.Unmarshal(updatedData, &updatedPkg); err != nil {
		t.Fatalf("failed to parse updated package.json: %v", err)
	}

	deps := updatedPkg["dependencies"].(map[string]interface{})
	if deps["express"] != "^4.18.2" {
		t.Errorf("expected express version ^4.18.2, got %v", deps["express"])
	}
	if deps["lodash"] != "^4.17.20" {
		t.Errorf("expected lodash version ^4.17.20, got %v", deps["lodash"])
	}

	devDeps := updatedPkg["devDependencies"].(map[string]interface{})
	if devDeps["jest"] != "^29.3.1" {
		t.Errorf("expected jest version ^29.3.1, got %v", devDeps["jest"])
	}

	if len(capturedCommands) != 1 || capturedCommands[0] != "npm install" {
		t.Errorf("expected 'npm install' to be called, got: %v", capturedCommands)
	}
}

func TestUpdatePackageJSON_ReadError(t *testing.T) {
	updater := NewUpdater("/nonexistent/dir")
	modules := []scanner.Module{
		{Name: "express", Update: &scanner.UpdateInfo{Version: "4.18.2"}},
	}

	err := updater.UpdatePackageJSON(modules)
	if err == nil {
		t.Fatal("expected error when reading nonexistent package.json")
	}

	if !strings.Contains(err.Error(), "failed to read package.json") {
		t.Errorf("expected error to contain 'failed to read package.json', got %v", err)
	}
}
