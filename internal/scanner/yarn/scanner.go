// Package yarn provides yarn package manager scanning functionality.
package yarn

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pragmaticivan/faro/internal/scanner"
)

// Scanner implements scanner.Scanner for yarn.
type Scanner struct {
	workDir         string
	runYarnOutdated func() ([]byte, error)
}

// yarnOutdated represents the structure of `yarn outdated --json` output.
type yarnOutdated struct {
	Type string            `json:"type"`
	Data yarnOutdatedTable `json:"data,omitempty"`
}

type yarnOutdatedTable struct {
	Head []string   `json:"head"`
	Body [][]string `json:"body"`
}

// NewScanner creates a new yarn scanner.
func NewScanner(workDir string) *Scanner {
	return &Scanner{
		workDir: workDir,
		runYarnOutdated: func() ([]byte, error) {
			cmd := exec.Command("yarn", "outdated", "--json")
			cmd.Dir = workDir
			out, _ := cmd.Output() // yarn outdated may return non-zero
			return out, nil
		},
	}
}

// GetUpdates returns all yarn packages that have available updates.
func (s *Scanner) GetUpdates(opts scanner.Options) ([]scanner.Module, error) {
	pkgJSON, err := s.readPackageJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to read package.json: %w", err)
	}

	output, err := s.runYarnOutdated()
	if err != nil {
		return nil, fmt.Errorf("failed to run yarn outdated: %w", err)
	}

	if len(output) == 0 {
		return []scanner.Module{}, nil
	}

	var modules []scanner.Module
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		var entry yarnOutdated
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		if entry.Type == "table" && len(entry.Data.Body) > 0 {
			for _, row := range entry.Data.Body {
				if len(row) < 4 {
					continue
				}

				name := row[0]
				current := row[1]
				latest := row[3]

				_, isDirect := pkgJSON.Dependencies[name]
				_, isDevDirect := pkgJSON.DevDependencies[name]

				depType := "dependencies"
				if isDevDirect {
					depType = "devDependencies"
				} else if !isDirect {
					depType = "transitive"
				}

				if !opts.IncludeAll && depType == "devDependencies" {
					continue
				}

				if !opts.IncludeAll && depType == "transitive" {
					continue
				}

				if opts.Filter != "" && !strings.Contains(name, opts.Filter) {
					continue
				}

				module := scanner.Module{
					Name:           name,
					Version:        current,
					Direct:         isDirect || isDevDirect,
					DependencyType: depType,
					Update: &scanner.UpdateInfo{
						Version: latest,
					},
				}

				modules = append(modules, module)
			}
		}
	}

	return modules, nil
}

// GetDependencyIndex returns a map of yarn package names to their dependency information.
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

type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
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
