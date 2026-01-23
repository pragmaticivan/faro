// Package gomod provides Go module scanning functionality.
package gomod

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pragmaticivan/faro/internal/cooldown"
	"github.com/pragmaticivan/faro/internal/gomod"
	"github.com/pragmaticivan/faro/internal/scanner"
)

// Scanner implements scanner.Scanner for Go modules.
type Scanner struct {
	workDir        string
	goModPath      string
	listAllModules func() ([]byte, error)
}

// goModule is the internal representation from `go list` output.
type goModule struct {
	Path     string    `json:"Path"`
	Version  string    `json:"Version"`
	Time     string    `json:"Time"`
	Update   *goModule `json:"Update"`
	Indirect bool      `json:"Indirect"`
}

// NewScanner creates a new Go module scanner.
func NewScanner(workDir string) *Scanner {
	return &Scanner{
		workDir:   workDir,
		goModPath: filepath.Join(workDir, "go.mod"),
		listAllModules: func() ([]byte, error) {
			cmd := exec.Command("go", "list", "-m", "-u", "-json", "all")
			cmd.Dir = workDir
			return cmd.Output()
		},
	}
}

// GetUpdates returns all Go modules that have available updates.
func (s *Scanner) GetUpdates(opts scanner.Options) ([]scanner.Module, error) {
	idx, err := gomod.ReadRequireIndex(s.goModPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read go.mod: %w", err)
	}

	var filterRegex *regexp.Regexp
	if opts.Filter != "" {
		compiled, err := regexp.Compile(opts.Filter)
		if err != nil {
			return nil, fmt.Errorf("invalid filter pattern: %w", err)
		}
		filterRegex = compiled
	}

	output, err := s.listAllModules()
	if err != nil {
		return nil, fmt.Errorf("failed to run go list: %w", err)
	}

	goModules, err := decodeGoListModules(output)
	if err != nil {
		return nil, err
	}

	return s.annotateAndFilter(goModules, idx, opts, filterRegex, time.Now()), nil
}

// GetDependencyIndex returns a map of Go module paths to their dependency information.
func (s *Scanner) GetDependencyIndex() (scanner.DependencyIndex, error) {
	idx, err := gomod.ReadRequireIndex(s.goModPath)
	if err != nil {
		return nil, err
	}

	depIdx := make(scanner.DependencyIndex)
	for path, indirect := range idx {
		depType := "direct"
		if indirect {
			depType = "indirect"
		}
		depIdx[path] = scanner.DependencyInfo{
			Direct: !indirect,
			Type:   depType,
		}
	}
	return depIdx, nil
}

// decodeGoListModules decodes the JSON stream output from `go list -m -u -json all`.
func decodeGoListModules(data []byte) ([]goModule, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	var modules []goModule
	for decoder.More() {
		var m goModule
		if err := decoder.Decode(&m); err != nil {
			return nil, fmt.Errorf("failed to decode json: %w", err)
		}
		modules = append(modules, m)
	}
	return modules, nil
}

// annotateAndFilter applies go.mod classification and filters modules based on opts.
func (s *Scanner) annotateAndFilter(
	modules []goModule,
	idx gomod.RequireIndex,
	opts scanner.Options,
	filterRegex *regexp.Regexp,
	now time.Time,
) []scanner.Module {
	out := make([]scanner.Module, 0, len(modules))
	for _, m := range modules {
		if m.Update == nil {
			continue
		}

		// Override classification based on go.mod
		fromGoMod := false
		indirect := m.Indirect
		depType := "transitive"
		if idxIndirect, ok := idx[m.Path]; ok {
			fromGoMod = true
			indirect = idxIndirect
			if indirect {
				depType = "indirect"
			} else {
				depType = "direct"
			}
		}

		// Filter out transitive dependencies if not including all
		if !opts.IncludeAll && !fromGoMod {
			continue
		}

		// Apply filter
		if opts.Filter != "" {
			match := strings.Contains(m.Path, opts.Filter)
			if !match && filterRegex != nil {
				match = filterRegex.MatchString(m.Path)
			}
			if !match {
				continue
			}
		}

		// Apply cooldown
		if opts.CooldownDays > 0 {
			if !cooldown.Eligible(m.Update.Time, opts.CooldownDays, now) {
				continue
			}
		}

		// Convert to scanner.Module
		module := scanner.Module{
			Name:           m.Path,
			Version:        m.Version,
			Time:           m.Time,
			Direct:         !indirect,
			DependencyType: depType,
			// Legacy fields for backward compatibility
			Path:      m.Path,
			Indirect:  indirect,
			FromGoMod: fromGoMod,
		}
		if m.Update != nil {
			module.Update = &scanner.UpdateInfo{
				Version: m.Update.Version,
				Time:    m.Update.Time,
			}
		}
		out = append(out, module)
	}
	return out
}
