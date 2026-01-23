// Package uv provides uv package manager update functionality.
package uv

import (
	"fmt"
	"os/exec"

	"github.com/pragmaticivan/faro/internal/scanner"
)

// Updater implements updater.Updater for uv.
type Updater struct {
	workDir  string
	runUvCmd func(args ...string) ([]byte, error)
}

// NewUpdater creates a new uv updater.
func NewUpdater(workDir string) *Updater {
	return &Updater{
		workDir: workDir,
		runUvCmd: func(args ...string) ([]byte, error) {
			cmd := exec.Command("uv", args...)
			cmd.Dir = workDir
			return cmd.CombinedOutput()
		},
	}
}

// UpdatePackages updates multiple uv packages to their specified versions.
func (u *Updater) UpdatePackages(modules []scanner.Module) error {
	if len(modules) == 0 {
		return nil
	}

	fmt.Printf("Upgrading %d packages...\n", len(modules))

	for _, m := range modules {
		pkgSpec := m.Name
		if m.Update != nil && m.Update.Version != "" {
			pkgSpec = fmt.Sprintf("%s==%s", m.Name, m.Update.Version)
		}

		args := []string{"pip", "install", pkgSpec}
		if out, err := u.runUvCmd(args...); err != nil {
			return fmt.Errorf("uv pip install failed: %s: %w", string(out), err)
		}
	}

	return nil
}

// UpdateSinglePackage updates a single uv package to its specified version.
func (u *Updater) UpdateSinglePackage(module scanner.Module) error {
	return u.UpdatePackages([]scanner.Module{module})
}
