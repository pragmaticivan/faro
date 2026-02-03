// Package poetry provides poetry package manager scanning functionality.
package poetry

import (
	"bufio"
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

		// Parse line: "package-name (!) current latest [description]"
		// The (!) indicator is present for all outdated packages
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		name := fields[0]
		// Skip the (!) indicator
		current := fields[2]
		latest := fields[3]

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
	deps, devDeps, err := s.readPyprojectToml()
	if err != nil {
		return nil, err
	}

	idx := make(scanner.DependencyIndex)

	// Parse main dependencies
	for name := range deps {
		if name == "python" {
			continue
		}
		idx[name] = scanner.DependencyInfo{Direct: true, Type: "main"}
	}

	// Parse dev dependencies
	for name := range devDeps {
		idx[name] = scanner.DependencyInfo{Direct: true, Type: "dev"}
	}

	return idx, nil
}

// readPyprojectToml reads and parses pyproject.toml dependencies.
// This is a simplified TOML parser that only extracts dependency names.
func (s *Scanner) readPyprojectToml() (deps map[string]bool, devDeps map[string]bool, err error) {
	path := filepath.Join(s.workDir, "pyproject.toml")
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	deps = make(map[string]bool)
	devDeps = make(map[string]bool)

	scanner := bufio.NewScanner(file)
	var inDependencies, inDevDependencies bool

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Check for section headers
		if strings.HasPrefix(line, "[tool.poetry.dependencies]") {
			inDependencies = true
			inDevDependencies = false
			continue
		} else if strings.HasPrefix(line, "[tool.poetry.dev-dependencies]") || strings.HasPrefix(line, "[tool.poetry.group.dev.dependencies]") {
			inDevDependencies = true
			inDependencies = false
			continue
		} else if strings.HasPrefix(line, "[") {
			// New section started
			inDependencies = false
			inDevDependencies = false
			continue
		}

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse dependency line (format: package = "version")
		if inDependencies || inDevDependencies {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				pkgName := strings.TrimSpace(parts[0])
				if inDependencies {
					deps[pkgName] = true
				} else {
					devDeps[pkgName] = true
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}

	return deps, devDeps, nil
}
