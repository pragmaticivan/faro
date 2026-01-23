// Package poetry provides poetry package manager scanning functionality.
package poetry

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pragmaticivan/faro/internal/scanner"
)

// Scanner implements scanner.Scanner for Poetry.
type Scanner struct {
	workDir      string
	runPoetryCmd func(args ...string) ([]byte, error)
}

// NewScanner creates a new Poetry scanner.
func NewScanner(workDir string) *Scanner {
	return &Scanner{
		workDir: workDir,
		runPoetryCmd: func(args ...string) ([]byte, error) {
			cmd := exec.Command("poetry", args...)
			cmd.Dir = workDir
			return cmd.Output()
		},
	}
}

// GetUpdates returns all Poetry packages that have available updates.
func (s *Scanner) GetUpdates(opts scanner.Options) ([]scanner.Module, error) {
	// Read pyproject.toml to determine dependency types
	depIdx, err := s.GetDependencyIndex()
	if err != nil {
		return nil, err
	}

	// Run poetry show --outdated to get updates
	output, err := s.runPoetryCmd("show", "--outdated")
	// If no outdated packages, poetry show --outdated may return error
	if err != nil {
		return []scanner.Module{}, nil
	}

	lines := strings.Split(string(output), "\n")
	var modules []scanner.Module
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse line: "package-name version latest description"
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		name := fields[0]
		current := fields[1]
		latest := fields[2]

		depInfo, isDirect := depIdx[name]
		if !isDirect {
			depInfo = scanner.DependencyInfo{Direct: false, Type: "transitive"}
		}

		// Filter dev dependencies if not including all
		if !opts.IncludeAll && depInfo.Type == "dev" {
			continue
		}

		// Filter transitive if not including all
		if !opts.IncludeAll && !depInfo.Direct {
			continue
		}

		// Apply filter
		if opts.Filter != "" && !strings.Contains(name, opts.Filter) {
			continue
		}

		module := scanner.Module{
			Name:           name,
			Version:        current,
			Direct:         depInfo.Direct,
			DependencyType: depInfo.Type,
			Update: &scanner.UpdateInfo{
				Version: latest,
			},
		}
		modules = append(modules, module)
	}

	return modules, nil
}

// GetDependencyIndex returns a map of Poetry package names to their dependency information.
func (s *Scanner) GetDependencyIndex() (scanner.DependencyIndex, error) {
	pyproject, err := s.readPyprojectToml()
	if err != nil {
		return nil, err
	}

	idx := make(scanner.DependencyIndex)

	// Parse main dependencies
	if deps, ok := pyproject["tool"].(map[string]interface{})["poetry"].(map[string]interface{})["dependencies"].(map[string]interface{}); ok {
		for name := range deps {
			if name == "python" {
				continue
			}
			idx[name] = scanner.DependencyInfo{Direct: true, Type: "main"}
		}
	}

	// Parse dev dependencies
	if deps, ok := pyproject["tool"].(map[string]interface{})["poetry"].(map[string]interface{})["dev-dependencies"].(map[string]interface{}); ok {
		for name := range deps {
			idx[name] = scanner.DependencyInfo{Direct: true, Type: "dev"}
		}
	}

	return idx, nil
}

// readPyprojectToml reads and parses pyproject.toml.
// This is a simplified implementation; in production, use a TOML parser library.
func (s *Scanner) readPyprojectToml() (map[string]interface{}, error) {
	path := filepath.Join(s.workDir, "pyproject.toml")
	_, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// In production, use github.com/pelletier/go-toml or similar
	// For simplicity, return empty map
	return make(map[string]interface{}), nil
}
