// Package pip provides pip package manager scanning functionality.
package pip

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pragmaticivan/faro/internal/scanner"
)

// Scanner implements scanner.Scanner for pip.
type Scanner struct {
	workDir   string
	runPipCmd func(args ...string) ([]byte, error)
}

// pipOutdated represents the structure of `pip list --outdated --format json` output.
type pipOutdated []pipPackageInfo

type pipPackageInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Latest  string `json:"latest_version"`
	Type    string `json:"latest_filetype"`
}

// NewScanner creates a new pip scanner.
func NewScanner(workDir string) *Scanner {
	return &Scanner{
		workDir: workDir,
		runPipCmd: func(args ...string) ([]byte, error) {
			cmd := exec.Command("pip", args...)
			cmd.Dir = workDir
			return cmd.Output()
		},
	}
}

// GetUpdates returns all pip packages that have available updates.
func (s *Scanner) GetUpdates(opts scanner.Options) ([]scanner.Module, error) {
	// Read requirements.txt to determine direct dependencies
	directDeps, err := s.readRequirementsTxt()
	if err != nil {
		return nil, fmt.Errorf("failed to read requirements.txt: %w", err)
	}

	// Get outdated packages from pip
	output, err := s.runPipCmd("list", "--outdated", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to run pip list --outdated: %w", err)
	}

	var outdated pipOutdated
	if err := json.Unmarshal(output, &outdated); err != nil {
		return nil, fmt.Errorf("failed to parse pip output: %w", err)
	}

	var modules []scanner.Module
	for _, info := range outdated {
		_, isDirect := directDeps[strings.ToLower(info.Name)]

		// Filter transitive if not including all
		if !opts.IncludeAll && !isDirect {
			continue
		}

		// Apply filter
		if opts.Filter != "" && !strings.Contains(strings.ToLower(info.Name), strings.ToLower(opts.Filter)) {
			continue
		}

		depType := "main"
		if !isDirect {
			depType = "transitive"
		}

		module := scanner.Module{
			Name:           info.Name,
			Version:        info.Version,
			Direct:         isDirect,
			DependencyType: depType,
			Update: &scanner.UpdateInfo{
				Version: info.Latest,
			},
		}
		modules = append(modules, module)
	}

	return modules, nil
}

// GetDependencyIndex returns a map of pip package names to their dependency information.
func (s *Scanner) GetDependencyIndex() (scanner.DependencyIndex, error) {
	directDeps, err := s.readRequirementsTxt()
	if err != nil {
		return nil, err
	}

	idx := make(scanner.DependencyIndex)
	for name := range directDeps {
		idx[name] = scanner.DependencyInfo{Direct: true, Type: "main"}
	}
	return idx, nil
}

// readRequirementsTxt reads requirements.txt and returns a map of package names.
func (s *Scanner) readRequirementsTxt() (map[string]bool, error) {
	path := filepath.Join(s.workDir, "requirements.txt")
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]bool), nil
		}
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	deps := make(map[string]bool)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse package name (handle version specs like package==1.0.0, package>=1.0.0, etc.)
		parts := strings.FieldsFunc(line, func(r rune) bool {
			return r == '=' || r == '>' || r == '<' || r == '~' || r == '!'
		})
		if len(parts) > 0 {
			pkgName := strings.TrimSpace(parts[0])
			deps[strings.ToLower(pkgName)] = true
		}
	}

	return deps, scanner.Err()
}
