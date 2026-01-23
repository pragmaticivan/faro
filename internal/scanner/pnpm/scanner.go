// Package pnpm provides pnpm package manager scanning functionality.
package pnpm

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pragmaticivan/faro/internal/scanner"
)

// Scanner implements scanner.Scanner for pnpm.
type Scanner struct {
	workDir         string
	runPnpmOutdated func() ([]byte, error)
}

// pnpmOutdated represents the structure of `pnpm outdated --json` output.
type pnpmOutdated map[string]pnpmPackageInfo

type pnpmPackageInfo struct {
	Current string `json:"current"`
	Latest  string `json:"latest"`
	Wanted  string `json:"wanted"`
}

type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// NewScanner creates a new pnpm scanner.
func NewScanner(workDir string) *Scanner {
	return &Scanner{
		workDir: workDir,
		runPnpmOutdated: func() ([]byte, error) {
			cmd := exec.Command("pnpm", "outdated", "--json")
			cmd.Dir = workDir
			out, _ := cmd.Output() // pnpm outdated may return non-zero
			return out, nil
		},
	}
}

// GetUpdates returns all pnpm packages that have available updates.
func (s *Scanner) GetUpdates(opts scanner.Options) ([]scanner.Module, error) {
	pkgJSON, err := s.readPackageJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to read package.json: %w", err)
	}

	output, err := s.runPnpmOutdated()
	if err != nil {
		return nil, fmt.Errorf("failed to run pnpm outdated: %w", err)
	}

	if len(output) == 0 {
		return []scanner.Module{}, nil
	}

	var outdated pnpmOutdated
	if err := json.Unmarshal(output, &outdated); err != nil {
		return nil, fmt.Errorf("failed to parse pnpm outdated output: %w", err)
	}

	var modules []scanner.Module
	for name, info := range outdated {
		_, isDirect := pkgJSON.Dependencies[name]
		_, isDevDirect := pkgJSON.DevDependencies[name]

		depType := "dependencies"
		if isDevDirect {
			depType = "devDependencies"
		} else if !isDirect {
			depType = "transitive"
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

// GetDependencyIndex returns a map of pnpm package names to their dependency information.
func (s *Scanner) GetDependencyIndex() (scanner.DependencyIndex, error) {
	pkgJSON, err := s.readPackageJSON()
	if err != nil {
		return nil, err
	}

	idx := make(scanner.DependencyIndex)
	for name := range pkgJSON.Dependencies {
		idx[name] = scanner.DependencyInfo{Direct: true, Type: "dependencies"}
	}
	for name := range pkgJSON.DevDependencies {
		idx[name] = scanner.DependencyInfo{Direct: true, Type: "devDependencies"}
	}
	return idx, nil
}

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
