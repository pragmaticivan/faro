// Package gomod provides Go module update functionality.
package gomod

import (
	"fmt"
	"os/exec"

	"github.com/pragmaticivan/faro/internal/scanner"
)

// Updater implements updater.Updater for Go modules.
type Updater struct {
	workDir string
	runCmd  func(name string, args ...string) ([]byte, error)
}

// NewUpdater creates a new Go module updater.
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

// UpdatePackages updates multiple Go modules to their specified versions.
func (u *Updater) UpdatePackages(modules []scanner.Module) error {
	if len(modules) == 0 {
		return nil
	}

	fmt.Printf("Upgrading %d packages...\n", len(modules))

	args := u.buildGoGetArgs(modules)
	if out, err := u.runCmd("go", args...); err != nil {
		return fmt.Errorf("go get failed: %s: %w", string(out), err)
	}

	// Tidy up
	if out, err := u.runCmd("go", "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy failed: %s: %w", string(out), err)
	}

	return nil
}

// UpdateSinglePackage updates a single Go module to its specified version.
func (u *Updater) UpdateSinglePackage(module scanner.Module) error {
	return u.UpdatePackages([]scanner.Module{module})
}

// buildGoGetArgs constructs the arguments for `go get`.
func (u *Updater) buildGoGetArgs(modules []scanner.Module) []string {
	args := []string{"get"}
	for _, m := range modules {
		path := m.Name
		if path == "" {
			path = m.Path // Fallback for legacy compatibility
		}

		if m.Update != nil && m.Update.Version != "" {
			args = append(args, fmt.Sprintf("%s@%s", path, m.Update.Version))
		} else {
			args = append(args, path)
		}
	}
	return args
}
