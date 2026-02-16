package pip

import (
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
	// Setup temporary directory with requirements.txt
	tempDir, err := os.MkdirTemp("", "pip-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	reqContent := "requests==2.28.0\nflask==2.2.0\n"
	reqPath := filepath.Join(tempDir, "requirements.txt")
	if err := os.WriteFile(reqPath, []byte(reqContent), 0644); err != nil {
		t.Fatalf("failed to write requirements.txt: %v", err)
	}

	modules := []scanner.Module{
		{Name: "requests", Update: &scanner.UpdateInfo{Version: "2.28.1"}},
		{Name: "flask", Update: &scanner.UpdateInfo{Version: "2.2.2"}},
	}

	var capturedCommands []string
	updater := &Updater{
		workDir: tempDir,
		runCmd: func(name string, args ...string) ([]byte, error) {
			capturedCommands = append(capturedCommands, name+" "+strings.Join(args, " "))
			return []byte("success"), nil
		},
	}

	err = updater.UpdatePackages(modules)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify pip install commands (order matters in implementation)
	if len(capturedCommands) != 2 {
		t.Fatalf("expected 2 commands, got %d: %v", len(capturedCommands), capturedCommands)
	}

	expectedFirst := "pip install requests==2.28.1"
	if capturedCommands[0] != expectedFirst {
		t.Errorf("expected command %q, got %q", expectedFirst, capturedCommands[0])
	}

	expectedSecond := "pip install flask==2.2.2"
	if capturedCommands[1] != expectedSecond {
		t.Errorf("expected command %q, got %q", expectedSecond, capturedCommands[1])
	}

	// Verify requirements.txt updated
	updatedReq, err := os.ReadFile(reqPath)
	if err != nil {
		t.Fatalf("failed to read updated requirements.txt: %v", err)
	}

	expectedContent := "requests==2.28.1\nflask==2.2.2\n"
	if string(updatedReq) != expectedContent {
		t.Errorf("expected requirements.txt content:\n%q\ngot:\n%q", expectedContent, string(updatedReq))
	}
}

func TestUpdatePackages_InstallFails(t *testing.T) {
	updater := &Updater{
		workDir: "/test/dir",
		runCmd: func(_ string, _ ...string) ([]byte, error) {
			return []byte("pip failed"), errors.New("exit 1")
		},
	}

	modules := []scanner.Module{
		{Name: "requests", Update: &scanner.UpdateInfo{Version: "2.28.1"}},
	}

	err := updater.UpdatePackages(modules)
	if err == nil {
		t.Fatal("expected error when pip install fails")
	}

	if !strings.Contains(err.Error(), "pip install") {
		t.Errorf("expected error to contain 'pip install', got %v", err)
	}
}

func TestUpdatePackages_RequirementsTxtMissing(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pip-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	updater := &Updater{
		workDir: tempDir,
		runCmd: func(_ string, _ ...string) ([]byte, error) {
			return []byte("success"), nil
		},
	}

	modules := []scanner.Module{
		{Name: "requests", Update: &scanner.UpdateInfo{Version: "2.28.1"}},
	}

	// Should fail because requirements.txt is missing
	err = updater.UpdatePackages(modules)
	if err == nil {
		t.Fatal("expected error when requirements.txt is missing")
	}
}

func TestUpdateRequirementsTxt_PreservesCommentsAndFormatting(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pip-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	initialContent := `# Project requirements
requests==2.28.0
# Core utils
flask>=2.0.0
gunicorn
`
	reqPath := filepath.Join(tempDir, "requirements.txt")
	if err := os.WriteFile(reqPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("failed to write requirements.txt: %v", err)
	}

	modules := []scanner.Module{
		{Name: "requests", Update: &scanner.UpdateInfo{Version: "2.28.1"}},
		{Name: "flask", Update: &scanner.UpdateInfo{Version: "2.2.2"}},
	}

	updater := NewUpdater(tempDir)
	// We call updateRequirementsTxt via private method access through reflection or just test UpdatePackages which calls it
	// Since updateRequirementsTxt is unexported, we test via UpdatePackages but we need empty runCmd
	updater.runCmd = func(_ string, _ ...string) ([]byte, error) {
		return []byte("success"), nil
	}

	err = updater.UpdatePackages(modules)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	updatedReq, err := os.ReadFile(reqPath)
	if err != nil {
		t.Fatalf("failed to read updated requirements.txt: %v", err)
	}

	expectedContent := `# Project requirements
requests==2.28.1
# Core utils
flask==2.2.2
gunicorn
`
	if string(updatedReq) != expectedContent {
		t.Errorf("expected requirements.txt content:\n%q\ngot:\n%q", expectedContent, string(updatedReq))
	}
}
