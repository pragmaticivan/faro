// Package poetry provides Poetry package manager update functionality.
package poetry

import (
	"fmt"
	"os/exec"

	"github.com/pragmaticivan/faro/internal/scanner"
)

// Updater implements updater.Updater for Poetry.
type Updater struct {
	workDir      string
	runPoetryCmd func(args ...string) ([]byte, error)
}

// NewUpdater creates a new Poetry updater.
func NewUpdater(workDir string) *Updater {
	return &Updater{
		workDir: workDir,
		runPoetryCmd: func(args ...string) ([]byte, error) {
			cmd := exec.Command("poetry", args...)
			cmd.Dir = workDir
			return cmd.CombinedOutput()
		},
	}
}

// UpdatePackages updates multiple Poetry packages to their specified versions.
func (u *Updater) UpdatePackages(modules []scanner.Module) error {
	if len(modules) == 0 {
		return nil
	}

	fmt.Printf("Upgrading %d packages...\n", len(modules))

	for _, m := range modules {
		pkgSpec := m.Name
		if m.Update != nil && m.Update.Version != "" {
			pkgSpec = fmt.Sprintf("%s@%s", m.Name, m.Update.Version)
		}

		var args []string
		if m.DependencyType == "dev" {
			args = []string{"add", "--group", "dev", pkgSpec}
		} else {
			args = []string{"add", pkgSpec}
		}

		if out, err := u.runPoetryCmd(args...); err != nil {
			return fmt.Errorf("poetry add failed: %s: %w", string(out), err)
		}
	}

	return nil
}

// UpdateSinglePackage updates a single Poetry package to its specified version.
func (u *Updater) UpdateSinglePackage(module scanner.Module) error {
	return u.UpdatePackages([]scanner.Module{module})
}
