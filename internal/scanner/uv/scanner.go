// Package uv provides uv package manager scanning functionality.
package uv

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/pragmaticivan/faro/internal/scanner"
)

// Scanner implements scanner.Scanner for uv.
type Scanner struct {
	workDir  string
	runUvCmd func(args ...string) ([]byte, error)
}

// uvOutdated represents the structure of `uv pip list --outdated --format json` output.
type uvOutdated []uvPackageInfo

type uvPackageInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Latest  string `json:"latest_version"`
}

// NewScanner creates a new uv scanner.
func NewScanner(workDir string) *Scanner {
	return &Scanner{
		workDir: workDir,
		runUvCmd: func(args ...string) ([]byte, error) {
			cmd := exec.Command("uv", args...)
			cmd.Dir = workDir
			return cmd.Output()
		},
	}
}

// GetUpdates returns all uv packages that have available updates.
func (s *Scanner) GetUpdates(opts scanner.Options) ([]scanner.Module, error) {
	// Get outdated packages from uv
	output, err := s.runUvCmd("pip", "list", "--outdated", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to run uv pip list --outdated: %w", err)
	}

	var outdated uvOutdated
	if err := json.Unmarshal(output, &outdated); err != nil {
		return nil, fmt.Errorf("failed to parse uv output: %w", err)
	}

	var modules []scanner.Module
	for _, info := range outdated {
		// Apply filter
		if opts.Filter != "" && !strings.Contains(strings.ToLower(info.Name), strings.ToLower(opts.Filter)) {
			continue
		}

		module := scanner.Module{
			Name:           info.Name,
			Version:        info.Version,
			Direct:         true, // uv doesn't distinguish in list output
			DependencyType: "main",
			Update: &scanner.UpdateInfo{
				Version: info.Latest,
			},
		}
		modules = append(modules, module)
	}

	return modules, nil
}

// GetDependencyIndex returns a map of uv package names to their dependency information.
func (s *Scanner) GetDependencyIndex() (scanner.DependencyIndex, error) {
	// uv pip list shows installed packages
	output, err := s.runUvCmd("pip", "list", "--format", "json")
	if err != nil {
		return nil, err
	}

	var packages []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(output, &packages); err != nil {
		return nil, err
	}

	idx := make(scanner.DependencyIndex)
	for _, pkg := range packages {
		idx[pkg.Name] = scanner.DependencyInfo{Direct: true, Type: "main"}
	}
	return idx, nil
}
