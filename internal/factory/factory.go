// Package factory provides scanner and updater factory functions for different package managers.
package factory

import (
	"fmt"

	"github.com/pragmaticivan/faro/internal/detector"
	"github.com/pragmaticivan/faro/internal/scanner"
	"github.com/pragmaticivan/faro/internal/scanner/gomod"
	"github.com/pragmaticivan/faro/internal/scanner/npm"
	"github.com/pragmaticivan/faro/internal/scanner/pip"
	"github.com/pragmaticivan/faro/internal/scanner/pnpm"
	"github.com/pragmaticivan/faro/internal/scanner/poetry"
	"github.com/pragmaticivan/faro/internal/scanner/uv"
	"github.com/pragmaticivan/faro/internal/scanner/yarn"
	"github.com/pragmaticivan/faro/internal/updater"
	gomodUpdater "github.com/pragmaticivan/faro/internal/updater/gomod"
	npmUpdater "github.com/pragmaticivan/faro/internal/updater/npm"
	pipUpdater "github.com/pragmaticivan/faro/internal/updater/pip"
	pnpmUpdater "github.com/pragmaticivan/faro/internal/updater/pnpm"
	poetryUpdater "github.com/pragmaticivan/faro/internal/updater/poetry"
	uvUpdater "github.com/pragmaticivan/faro/internal/updater/uv"
	yarnUpdater "github.com/pragmaticivan/faro/internal/updater/yarn"
	"github.com/pragmaticivan/faro/internal/vuln"
)

// CreateScanner creates a scanner for the specified package manager.
func CreateScanner(pm detector.PackageManager, workDir string) (scanner.Scanner, error) {
	switch pm {
	case detector.Go:
		return gomod.NewScanner(workDir), nil
	case detector.Npm:
		return npm.NewScanner(workDir), nil
	case detector.Yarn:
		return yarn.NewScanner(workDir), nil
	case detector.Pnpm:
		return pnpm.NewScanner(workDir), nil
	case detector.Pip:
		return pip.NewScanner(workDir), nil
	case detector.Poetry:
		return poetry.NewScanner(workDir), nil
	case detector.Uv:
		return uv.NewScanner(workDir), nil
	default:
		return nil, fmt.Errorf("unsupported package manager: %s", pm)
	}
}

// CreateUpdater creates an updater for the specified package manager.
func CreateUpdater(pm detector.PackageManager, workDir string) (updater.Updater, error) {
	switch pm {
	case detector.Go:
		return gomodUpdater.NewUpdater(workDir), nil
	case detector.Npm:
		return npmUpdater.NewUpdater(workDir), nil
	case detector.Yarn:
		return yarnUpdater.NewUpdater(workDir), nil
	case detector.Pnpm:
		return pnpmUpdater.NewUpdater(workDir), nil
	case detector.Pip:
		return pipUpdater.NewUpdater(workDir), nil
	case detector.Poetry:
		return poetryUpdater.NewUpdater(workDir), nil
	case detector.Uv:
		return uvUpdater.NewUpdater(workDir), nil
	default:
		return nil, fmt.Errorf("unsupported package manager: %s", pm)
	}
}

// CreateVulnClient creates a vulnerability client for the specified package manager.
func CreateVulnClient(pm detector.PackageManager) vuln.Client {
	ecosystem := getEcosystem(pm)
	return vuln.NewClientForEcosystem(ecosystem)
}

// getEcosystem maps package managers to OSV ecosystem names.
func getEcosystem(pm detector.PackageManager) string {
	switch pm {
	case detector.Go:
		return "Go"
	case detector.Npm, detector.Yarn, detector.Pnpm:
		return "npm"
	case detector.Pip, detector.Poetry, detector.Uv:
		return "PyPI"
	default:
		return "Go"
	}
}
