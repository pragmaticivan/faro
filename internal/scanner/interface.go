// Package scanner provides interfaces and types for dependency scanning across different package managers.
package scanner

import "time"

// Scanner is the interface that all package manager scanners must implement.
type Scanner interface {
	// GetUpdates returns all modules that have available updates.
	GetUpdates(opts Options) ([]Module, error)

	// GetDependencyIndex returns a map of package names to their dependency information.
	GetDependencyIndex() (DependencyIndex, error)
}

// DependencyIndex maps package names to their classification.
type DependencyIndex map[string]DependencyInfo

// DependencyInfo contains metadata about a dependency.
type DependencyInfo struct {
	Direct bool   // Whether this is a direct dependency
	Type   string // Type: "production", "dev", "optional", "peer", "indirect", etc.
}

// Module represents a package/module with version information (ecosystem-agnostic).
type Module struct {
	// Name is the package name/path (e.g., "github.com/pkg/errors" for Go, "express" for npm)
	Name string `json:"name"`

	// Version is the current version
	Version string `json:"version"`

	// Time is when the current version was published (RFC3339 format)
	Time string `json:"time,omitempty"`

	// Update contains the available update information (nil if no update available)
	Update *UpdateInfo `json:"update,omitempty"`

	// Direct indicates if this is a direct dependency (vs transitive/indirect)
	Direct bool `json:"direct"`

	// DependencyType describes the type of dependency:
	// Go: "direct", "indirect", "transitive"
	// npm/yarn/pnpm: "dependencies", "devDependencies", "peerDependencies", "optionalDependencies"
	// Python: "main", "dev", "optional"
	DependencyType string `json:"dependencyType"`

	// VulnCurrent holds vulnerability counts for the current version
	VulnCurrent VulnInfo `json:"-"`

	// VulnUpdate holds vulnerability counts for the update version
	VulnUpdate VulnInfo `json:"-"`

	// Legacy fields for backward compatibility with Go scanner
	Path      string `json:"Path,omitempty"`     // Alias for Name (Go compatibility)
	Indirect  bool   `json:"Indirect,omitempty"` // Go-specific
	FromGoMod bool   `json:"-"`                  // Go-specific
}

// UpdateInfo contains information about an available update.
type UpdateInfo struct {
	Version string `json:"version"`
	Time    string `json:"time,omitempty"`
}

// VulnInfo contains vulnerability information for a module version.
type VulnInfo struct {
	Low      int `json:"low"`
	Medium   int `json:"medium"`
	High     int `json:"high"`
	Critical int `json:"critical"`
	Total    int `json:"total"`
}

// Options configures dependency discovery across all scanners.
type Options struct {
	// Filter is a substring or regex pattern to filter package names
	Filter string

	// IncludeAll determines what additional dependencies to include:
	// - Go: include transitive dependencies not in go.mod
	// - npm/yarn/pnpm: include devDependencies
	// - Python: include all dependency groups
	IncludeAll bool

	// CooldownDays filters out versions published within the last N days
	CooldownDays int

	// WorkDir is the working directory for the scanner
	WorkDir string
}

// MaxPathLength calculates the maximum name length for formatting.
func MaxPathLength(modules []Module) int {
	max := 0
	for _, m := range modules {
		name := m.Name
		if name == "" {
			name = m.Path // Fallback for Go compatibility
		}
		if len(name) > max {
			max = len(name)
		}
	}
	return max
}

// FilterModules applies filtering and cooldown logic to modules.
func FilterModules(modules []Module, filter string, cooldownDays int, now time.Time) []Module {
	if filter == "" && cooldownDays == 0 {
		return modules
	}

	result := make([]Module, 0, len(modules))
	for _, m := range modules {
		// Apply filter
		if filter != "" {
			name := m.Name
			if name == "" {
				name = m.Path
			}
			if !contains(name, filter) {
				continue
			}
		}

		// Apply cooldown
		if cooldownDays > 0 && m.Update != nil {
			updateTime, err := time.Parse(time.RFC3339, m.Update.Time)
			if err == nil {
				age := int(now.Sub(updateTime).Hours() / 24)
				if age < cooldownDays {
					continue
				}
			}
		}

		result = append(result, m)
	}
	return result
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
