// Package updater provides interfaces for updating dependencies across different package managers.
package updater

import "github.com/pragmaticivan/faro/internal/scanner"

// Updater is the interface that all package manager updaters must implement.
type Updater interface {
	// UpdatePackages updates multiple packages to their specified versions.
	// It returns an error if any update fails.
	UpdatePackages(modules []scanner.Module) error

	// UpdateSinglePackage updates a single package to its specified version.
	UpdateSinglePackage(module scanner.Module) error
}
