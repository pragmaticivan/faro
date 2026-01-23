// Package npm provides npm package manager scanning functionality.
package npm

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pragmaticivan/faro/internal/scanner"
)

// Scanner implements scanner.Scanner for npm.
type Scanner struct {
	workDir        string
	runNpmOutdated func() ([]byte, error)
}

// packageJSON represents the structure of package.json.
type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// npmOutdated represents the structure of `npm outdated --json` output.
type npmOutdated map[string]npmPackageInfo

type npmPackageInfo struct {
	Current  string `json:"current"`
	Wanted   string `json:"wanted"`
	Latest   string `json:"latest"`
	Location string `json:"location"`
	Type     string `json:"type"` // "dependencies" or "devDependencies"
}

// NewScanner creates a new npm scanner.
func NewScanner(workDir string) *Scanner {
	return &Scanner{
		workDir: workDir,
		runNpmOutdated: func() ([]byte, error) {
			cmd := exec.Command("npm", "outdated", "--json")
			cmd.Dir = workDir
			// npm outdated returns exit code 1 when there are outdated packages
			// So we ignore the error and just get the output
			out, _ := cmd.Output()
			return out, nil
		},
	}
}

// GetUpdates returns all npm packages that have available updates.
func (s *Scanner) GetUpdates(opts scanner.Options) ([]scanner.Module, error) {
	// Read package.json to determine dependency types
	pkgJSON, err := s.readPackageJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to read package.json: %w", err)
	}

	// Get outdated packages from npm
	output, err := s.runNpmOutdated()
	if err != nil {
		return nil, fmt.Errorf("failed to run npm outdated: %w", err)
	}

	if len(output) == 0 {
		return []scanner.Module{}, nil
	}

	var outdated npmOutdated
	if err := json.Unmarshal(output, &outdated); err != nil {
		return nil, fmt.Errorf("failed to parse npm outdated output: %w", err)
	}

	var modules []scanner.Module
	for name, info := range outdated {
		// Determine if it's a direct dependency
		_, isDirect := pkgJSON.Dependencies[name]
		_, isDevDirect := pkgJSON.DevDependencies[name]

		depType := info.Type
		if depType == "" {
			if isDirect {
				depType = "dependencies"
			} else if isDevDirect {
				depType = "devDependencies"
			} else {
				depType = "transitive"
			}
		}

		// Filter devDependencies if not including all
		if !opts.IncludeAll && depType == "devDependencies" {
			continue
		}

		// Apply filter
		if opts.Filter != "" && !strings.Contains(name, opts.Filter) {
			continue
		}

		module := scanner.Module{
			Name:           name,
			Version:        info.Current,
			Direct:         isDirect || isDevDirect,
			DependencyType: depType,
			Update: &scanner.UpdateInfo{
				Version: info.Latest,
			},
		}
		modules = append(modules, module)
	}

	return modules, nil
}

// GetDependencyIndex returns a map of npm package names to their dependency information.
func (s *Scanner) GetDependencyIndex() (scanner.DependencyIndex, error) {
	pkgJSON, err := s.readPackageJSON()
	if err != nil {
		return nil, err
	}

	idx := make(scanner.DependencyIndex)
	for name := range pkgJSON.Dependencies {
		idx[name] = scanner.DependencyInfo{
			Direct: true,
			Type:   "dependencies",
		}
	}
	for name := range pkgJSON.DevDependencies {
		idx[name] = scanner.DependencyInfo{
			Direct: true,
			Type:   "devDependencies",
		}
	}
	return idx, nil
}

// readPackageJSON reads and parses package.json.
func (s *Scanner) readPackageJSON() (*packageJSON, error) {
	path := filepath.Join(s.workDir, "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}
	return &pkg, nil
}
