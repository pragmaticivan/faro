// Package npm provides npm package manager update functionality.
package npm

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pragmaticivan/faro/internal/scanner"
)

// Updater implements updater.Updater for npm.
type Updater struct {
	workDir string
	runCmd  func(name string, args ...string) ([]byte, error)
}

// NewUpdater creates a new npm updater.
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

// UpdatePackages updates multiple npm packages to their specified versions.
func (u *Updater) UpdatePackages(modules []scanner.Module) error {
	if len(modules) == 0 {
		return nil
	}

	fmt.Printf("Upgrading %d packages...\n", len(modules))

	// Group by dependency type
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

	// Install production dependencies
	if len(deps) > 0 {
		args := append([]string{"install", "--save"}, deps...)
		if out, err := u.runCmd("npm", args...); err != nil {
			return fmt.Errorf("npm install failed: %s: %w", string(out), err)
		}
	}

	// Install dev dependencies
	if len(devDeps) > 0 {
		args := append([]string{"install", "--save-dev"}, devDeps...)
		if out, err := u.runCmd("npm", args...); err != nil {
			return fmt.Errorf("npm install --save-dev failed: %s: %w", string(out), err)
		}
	}

	return nil
}

// UpdateSinglePackage updates a single npm package to its specified version.
func (u *Updater) UpdateSinglePackage(module scanner.Module) error {
	return u.UpdatePackages([]scanner.Module{module})
}

// UpdatePackageJSON directly updates package.json with new versions (alternative approach).
func (u *Updater) UpdatePackageJSON(modules []scanner.Module) error {
	pkgPath := filepath.Join(u.workDir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return fmt.Errorf("failed to read package.json: %w", err)
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return fmt.Errorf("failed to parse package.json: %w", err)
	}

	// Update versions
	for _, m := range modules {
		if m.Update == nil {
			continue
		}

		version := m.Update.Version
		if version[0] != '^' && version[0] != '~' && version[0] != '>' && version[0] != '<' {
			// Preserve semantic versioning prefix from original if present
			version = "^" + version
		}

		switch m.DependencyType {
		case "dependencies":
			if deps, ok := pkg["dependencies"].(map[string]interface{}); ok {
				deps[m.Name] = version
			}
		case "devDependencies":
			if deps, ok := pkg["devDependencies"].(map[string]interface{}); ok {
				deps[m.Name] = version
			}
		}
	}

	// Write updated package.json
	updatedData, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal package.json: %w", err)
	}

	if err := os.WriteFile(pkgPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write package.json: %w", err)
	}

	// Run npm install to update lockfile
	if out, err := u.runCmd("npm", "install"); err != nil {
		return fmt.Errorf("npm install failed after updating package.json: %s: %w", string(out), err)
	}

	return nil
}
