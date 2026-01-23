// Package yarn provides yarn package manager update functionality.
package yarn

import (
	"fmt"
	"os/exec"

	"github.com/pragmaticivan/faro/internal/scanner"
)

// Updater implements updater.Updater for yarn.
type Updater struct {
	workDir string
	runCmd  func(name string, args ...string) ([]byte, error)
}

// NewUpdater creates a new yarn updater.
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

// UpdatePackages updates multiple yarn packages to their specified versions.
func (u *Updater) UpdatePackages(modules []scanner.Module) error {
	if len(modules) == 0 {
		return nil
	}

	fmt.Printf("Upgrading %d packages...\n", len(modules))

	deps := make([]string, 0)
	devDeps := make([]string, 0)
	for _, m := range modules {
		pkgSpec := m.Name
		if m.Update != nil && m.Update.Version != "" {
			pkgSpec = fmt.Sprintf("%s@%s", m.Name, m.Update.Version)
		}

		if m.DependencyType == "devDependencies" {
			devDeps = append(devDeps, pkgSpec)
		} else {
			deps = append(deps, pkgSpec)
		}
	}

	if len(deps) > 0 {
		args := append([]string{"add"}, deps...)
		if out, err := u.runCmd("yarn", args...); err != nil {
			return fmt.Errorf("yarn add failed: %s: %w", string(out), err)
		}
	}

	if len(devDeps) > 0 {
		args := append([]string{"add", "--dev"}, devDeps...)
		if out, err := u.runCmd("yarn", args...); err != nil {
			return fmt.Errorf("yarn add --dev failed: %s: %w", string(out), err)
		}
	}

	return nil
}

// UpdateSinglePackage updates a single yarn package to its specified version.
func (u *Updater) UpdateSinglePackage(module scanner.Module) error {
	return u.UpdatePackages([]scanner.Module{module})
}
