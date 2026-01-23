// Package pip provides pip package manager update functionality.
package pip

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pragmaticivan/faro/internal/scanner"
)

// Updater implements updater.Updater for pip.
type Updater struct {
	workDir string
	runCmd  func(name string, args ...string) ([]byte, error)
}

// NewUpdater creates a new pip updater.
func NewUpdater(workDir string) *Updater {
	return &Updater{
		workDir: workDir,
		runCmd: func(name string, args ...string) ([]byte, error) {
			cmd := exec.Command(name, args...)
			cmd.Dir = workDir
			return cmd.CombinedOutput()
		},
	}
}

// UpdatePackages updates multiple pip packages to their specified versions.
func (u *Updater) UpdatePackages(modules []scanner.Module) error {
	if len(modules) == 0 {
		return nil
	}

	fmt.Printf("Upgrading %d packages...\n", len(modules))

	// Install packages
	for _, m := range modules {
		pkgSpec := m.Name
		if m.Update != nil && m.Update.Version != "" {
			pkgSpec = fmt.Sprintf("%s==%s", m.Name, m.Update.Version)
		}

		if out, err := u.runCmd("pip", "install", pkgSpec); err != nil {
			return fmt.Errorf("pip install %s failed: %s: %w", pkgSpec, string(out), err)
		}
	}

	// Update requirements.txt
	if err := u.updateRequirementsTxt(modules); err != nil {
		return fmt.Errorf("failed to update requirements.txt: %w", err)
	}

	return nil
}

// UpdateSinglePackage updates a single pip package to its specified version.
func (u *Updater) UpdateSinglePackage(module scanner.Module) error {
	return u.UpdatePackages([]scanner.Module{module})
}

// updateRequirementsTxt updates the requirements.txt file with new versions.
func (u *Updater) updateRequirementsTxt(modules []scanner.Module) error {
	reqPath := filepath.Join(u.workDir, "requirements.txt")

	// Read existing requirements
	file, err := os.Open(reqPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	updateMap := make(map[string]string)
	for _, m := range modules {
		if m.Update != nil {
			updateMap[strings.ToLower(m.Name)] = m.Update.Version
		}
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			lines = append(lines, line)
			continue
		}

		// Parse package name
		parts := strings.FieldsFunc(trimmed, func(r rune) bool {
			return r == '=' || r == '>' || r == '<' || r == '~' || r == '!'
		})

		if len(parts) > 0 {
			pkgName := strings.TrimSpace(parts[0])
			if newVersion, ok := updateMap[strings.ToLower(pkgName)]; ok {
				lines = append(lines, fmt.Sprintf("%s==%s", pkgName, newVersion))
				continue
			}
		}

		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Write updated requirements
	return os.WriteFile(reqPath, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}
