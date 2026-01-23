// Package detector provides automatic package manager detection based on project files.
package detector

import (
	"fmt"
	"os"
	"path/filepath"
)

// PackageManager represents a supported package manager.
type PackageManager string

const (
	Go     PackageManager = "go"
	Npm    PackageManager = "npm"
	Yarn   PackageManager = "yarn"
	Pnpm   PackageManager = "pnpm"
	Pip    PackageManager = "pip"
	Poetry PackageManager = "poetry"
	Uv     PackageManager = "uv"
)

// DetectionResult contains information about a detected package manager.
type DetectionResult struct {
	Manager    PackageManager
	ConfigFile string
	LockFile   string
}

// detector represents a package manager detection rule.
type detector struct {
	manager    PackageManager
	files      []string // Files that must exist
	configFile string   // Primary config file
	lockFile   string   // Lock file (if any)
	priority   int      // Lower = higher priority
}

var detectors = []detector{
	{
		manager:    Go,
		files:      []string{"go.mod"},
		configFile: "go.mod",
		lockFile:   "go.sum",
		priority:   1,
	},
	{
		manager:    Pnpm,
		files:      []string{"pnpm-lock.yaml"},
		configFile: "package.json",
		lockFile:   "pnpm-lock.yaml",
		priority:   2,
	},
	{
		manager:    Yarn,
		files:      []string{"yarn.lock"},
		configFile: "package.json",
		lockFile:   "yarn.lock",
		priority:   3,
	},
	{
		manager:    Npm,
		files:      []string{"package-lock.json"},
		configFile: "package.json",
		lockFile:   "package-lock.json",
		priority:   4,
	},
	{
		manager:    Poetry,
		files:      []string{"poetry.lock", "pyproject.toml"},
		configFile: "pyproject.toml",
		lockFile:   "poetry.lock",
		priority:   5,
	},
	{
		manager:    Uv,
		files:      []string{"uv.lock"},
		configFile: "pyproject.toml",
		lockFile:   "uv.lock",
		priority:   6,
	},
	{
		manager:    Pip,
		files:      []string{"requirements.txt"},
		configFile: "requirements.txt",
		lockFile:   "",
		priority:   7,
	},
}

// Detect scans the given directory for package manager files and returns all detected managers.
// Results are sorted by priority (most specific first).
func Detect(dir string) ([]DetectionResult, error) {
	var results []DetectionResult

	for _, d := range detectors {
		allExist := true
		for _, file := range d.files {
			path := filepath.Join(dir, file)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				allExist = false
				break
			}
		}

		if allExist {
			results = append(results, DetectionResult{
				Manager:    d.manager,
				ConfigFile: d.configFile,
				LockFile:   d.lockFile,
			})
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no supported package manager detected in %s", dir)
	}

	return results, nil
}

// DetectSingle detects a single package manager, preferring the highest priority match.
// If multiple managers are detected, it returns the first one based on priority.
func DetectSingle(dir string) (DetectionResult, error) {
	results, err := Detect(dir)
	if err != nil {
		return DetectionResult{}, err
	}
	return results[0], nil
}

// Validate checks if a given package manager name is supported.
func Validate(manager string) (PackageManager, error) {
	pm := PackageManager(manager)
	switch pm {
	case Go, Npm, Yarn, Pnpm, Pip, Poetry, Uv:
		return pm, nil
	default:
		return "", fmt.Errorf("unsupported package manager: %s (supported: go, npm, yarn, pnpm, pip, poetry, uv)", manager)
	}
}

// String returns the string representation of PackageManager.
func (pm PackageManager) String() string {
	return string(pm)
}
