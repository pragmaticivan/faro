package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/pragmaticivan/faro/internal/detector"
	"github.com/pragmaticivan/faro/internal/factory"
	"github.com/pragmaticivan/faro/internal/format"
	"github.com/pragmaticivan/faro/internal/scanner"
	"github.com/pragmaticivan/faro/internal/style"
	"github.com/pragmaticivan/faro/internal/tui"
	"github.com/pragmaticivan/faro/internal/updater"
	"github.com/pragmaticivan/faro/internal/vuln"
)

type RunOptions struct {
	Upgrade             bool
	Interactive         bool
	Filter              string
	All                 bool
	Cooldown            int
	FormatFlag          string
	ShowVulnerabilities bool
	Manager             string // Package manager override
}

type Deps struct {
	Out              io.Writer
	Now              func() time.Time
	StartInteractive func(direct, indirect, transitive []scanner.Module, opts tui.Options)
	Scanner          scanner.Scanner // Optional: verify overrides for testing
	Updater          updater.Updater // Optional: verify overrides for testing
}

// checkVulnerabilities checks for vulnerabilities in current and update versions
func checkVulnerabilities(ctx context.Context, modules []scanner.Module, vulnClient vuln.Client) {
	for i := range modules {
		if modules[i].Update != nil {
			// Use Name field, fallback to Path for backward compatibility
			pkgName := modules[i].Name
			if pkgName == "" {
				pkgName = modules[i].Path
			}

			// Check current version
			if currentCounts, err := vulnClient.CheckModule(ctx, pkgName, modules[i].Version); err == nil {
				modules[i].VulnCurrent = scanner.VulnInfo{
					Low:      currentCounts.Low,
					Medium:   currentCounts.Medium,
					High:     currentCounts.High,
					Critical: currentCounts.Critical,
					Total:    currentCounts.Total,
				}
			}

			// Check update version
			if updateCounts, err := vulnClient.CheckModule(ctx, pkgName, modules[i].Update.Version); err == nil {
				modules[i].VulnUpdate = scanner.VulnInfo{
					Low:      updateCounts.Low,
					Medium:   updateCounts.Medium,
					High:     updateCounts.High,
					Critical: updateCounts.Critical,
					Total:    updateCounts.Total,
				}
			}
		}
	}
}

// groupModules splits modules into direct, indirect, and transitive categories
func groupModules(modules []scanner.Module) (direct, indirect, transitive []scanner.Module) {
	for _, m := range modules {
		// Check if it's a direct dependency
		if m.Direct {
			// Further categorize based on dependency type
			switch m.DependencyType {
			case "devDependencies", "dev", "indirect":
				indirect = append(indirect, m)
			default:
				direct = append(direct, m)
			}
		} else {
			// Handle legacy Go fields for backward compatibility
			if m.FromGoMod {
				if m.Indirect {
					indirect = append(indirect, m)
				} else {
					direct = append(direct, m)
				}
			} else {
				transitive = append(transitive, m)
			}
		}
	}
	return direct, indirect, transitive
}

// printLinesFormat outputs modules in simple line format (path@version)
func printLinesFormat(out io.Writer, direct, indirect, transitive []scanner.Module, includeAll bool) {
	all := make([]scanner.Module, 0, len(direct)+len(indirect)+len(transitive))
	all = append(all, direct...)
	all = append(all, indirect...)
	if includeAll {
		all = append(all, transitive...)
	}
	for _, m := range all {
		if m.Update == nil {
			continue
		}
		name := m.Name
		if name == "" {
			name = m.Path // Fallback for backward compatibility
		}
		_, _ = fmt.Fprintf(out, "%s@%s\n", name, m.Update.Version)
	}
}

// printGroupedOutput prints modules organized by group labels
func printGroupedOutput(out io.Writer, group []scanner.Module, maxPathLen int, showVulns bool, showTime bool, now time.Time) {
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	byLabel := make(map[string][]scanner.Module)
	order := make(map[string]int)
	for _, m := range group {
		label := format.GroupLabel(m)
		byLabel[label] = append(byLabel[label], m)
		if _, ok := order[label]; !ok {
			order[label] = format.GroupSortKey(m)
		}
	}
	labels := make([]string, 0, len(byLabel))
	for k := range byLabel {
		labels = append(labels, k)
	}
	sort.Slice(labels, func(i, j int) bool {
		if order[labels[i]] != order[labels[j]] {
			return order[labels[i]] < order[labels[j]]
		}
		return labels[i] < labels[j]
	})

	for _, label := range labels {
		_, _ = fmt.Fprintf(out, "\n%s\n", dim.Render(label))
		for _, m := range byLabel[label] {
			name := m.Name
			if name == "" {
				name = m.Path // Fallback
			}
			line := " " + style.FormatUpdate(name, m.Version, m.Update.Version, maxPathLen)
			if showVulns && m.VulnCurrent.Total > 0 {
				line += " " + formatVulnCounts(m.VulnCurrent, m.VulnUpdate)
			}
			if showTime {
				pt := format.PublishTime(m.Update.Time, now)
				if pt != "" {
					line += "  " + dim.Render(pt)
				}
			}
			_, _ = fmt.Fprintln(out, line)
		}
	}
}

// printSimpleOutput prints modules in simple list format
func printSimpleOutput(out io.Writer, group []scanner.Module, maxPathLen int, showVulns bool, showTime bool, now time.Time) {
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	for _, m := range group {
		name := m.Name
		if name == "" {
			name = m.Path // Fallback
		}
		line := " " + style.FormatUpdate(name, m.Version, m.Update.Version, maxPathLen)
		if showVulns && m.VulnCurrent.Total > 0 {
			line += " " + formatVulnCounts(m.VulnCurrent, m.VulnUpdate)
		}
		if showTime {
			pt := format.PublishTime(m.Update.Time, now)
			if pt != "" {
				line += "  " + dim.Render(pt)
			}
		}
		_, _ = fmt.Fprintln(out, line)
	}
}

// printGroup outputs a titled group of modules
func printGroup(out io.Writer, title string, group []scanner.Module, maxPathLen int, grouped bool, showVulns bool, showTime bool, now time.Time) {
	if len(group) == 0 {
		return
	}
	_, _ = fmt.Fprintf(out, "\n%s\n", title)

	if grouped {
		printGroupedOutput(out, group, maxPathLen, showVulns, showTime, now)
	} else {
		printSimpleOutput(out, group, maxPathLen, showVulns, showTime, now)
	}
}

// calculateMaxPathLen finds the longest module path for alignment
func calculateMaxPathLen(direct, indirect, transitive []scanner.Module) int {
	maxPathLen := 0
	for _, group := range [][]scanner.Module{direct, indirect, transitive} {
		for _, m := range group {
			name := m.Name
			if name == "" {
				name = m.Path
			}
			if len(name) > maxPathLen {
				maxPathLen = len(name)
			}
		}
	}
	return maxPathLen
}

func Run(opts RunOptions, deps Deps) error {
	if deps.Out == nil {
		return fmt.Errorf("missing deps.Out")
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}

	// Detect or validate package manager
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	var pm detector.PackageManager
	if opts.Manager != "" {
		// Use explicit manager
		pm, err = detector.Validate(opts.Manager)
		if err != nil {
			return err
		}
	} else {
		// Auto-detect
		result, err := detector.DetectSingle(workDir)
		if err != nil {
			return fmt.Errorf("failed to detect package manager: %w\nSpecify one with --manager flag", err)
		}
		pm = result.Manager
	}

	// Create scanner and updater for the detected package manager
	var pkgScanner scanner.Scanner
	if deps.Scanner != nil {
		pkgScanner = deps.Scanner
	} else {
		pkgScanner, err = factory.CreateScanner(pm, workDir)
		if err != nil {
			return err
		}
	}

	formats, err := format.ParseFlag(opts.FormatFlag)
	if err != nil {
		return err
	}

	if !formats.Lines {
		_, _ = fmt.Fprintf(deps.Out, "Using package manager: %s\n", pm)
		_, _ = fmt.Fprintln(deps.Out, "Checking for updates...")
	}

	// Get updates using the package-specific scanner
	modules, err := pkgScanner.GetUpdates(scanner.Options{
		Filter:       opts.Filter,
		IncludeAll:   opts.All,
		CooldownDays: opts.Cooldown,
		WorkDir:      workDir,
	})
	if err != nil {
		return err
	}

	if len(modules) == 0 {
		if !formats.Lines {
			_, _ = fmt.Fprintln(deps.Out, "All dependencies match the latest package versions :)")
		}
		return nil
	}

	// Check vulnerabilities if requested
	if opts.ShowVulnerabilities {
		if !formats.Lines {
			_, _ = fmt.Fprintln(deps.Out, "Checking vulnerabilities...")
		}
		vulnClient := factory.CreateVulnClient(pm)
		ctx := context.Background()
		checkVulnerabilities(ctx, modules, vulnClient)
	}

	direct, indirect, transitive := groupModules(modules)

	// Adapt group labels based on package manager
	directLabel, indirectLabel, transitiveLabel := getGroupLabels(pm)

	if opts.Interactive {
		if deps.StartInteractive == nil {
			return fmt.Errorf("missing deps.StartInteractive")
		}
		// Create updater for interactive mode
		var updaterInstance updater.Updater
		if deps.Updater != nil {
			updaterInstance = deps.Updater
		} else {
			updaterInstance, err = factory.CreateUpdater(pm, workDir)
			if err != nil {
				return fmt.Errorf("failed to create updater: %w", err)
			}
		}
		deps.StartInteractive(direct, indirect, transitive, tui.Options{
			FormatGroup:     formats.Group,
			FormatTime:      formats.Time,
			Updater:         updaterInstance,
			DirectLabel:     directLabel,
			IndirectLabel:   indirectLabel,
			TransitiveLabel: transitiveLabel,
		})
		return nil
	}

	if formats.Lines {
		printLinesFormat(deps.Out, direct, indirect, transitive, opts.All)
		return nil
	}

	_, _ = fmt.Fprintln(deps.Out, "\nAvailable updates:")

	maxPathLen := calculateMaxPathLen(direct, indirect, transitive)
	now := deps.Now()

	printGroup(deps.Out, directLabel, direct, maxPathLen, formats.Group, opts.ShowVulnerabilities, formats.Time, now)
	printGroup(deps.Out, indirectLabel, indirect, maxPathLen, formats.Group, opts.ShowVulnerabilities, formats.Time, now)
	if opts.All {
		printGroup(deps.Out, transitiveLabel, transitive, maxPathLen, formats.Group, opts.ShowVulnerabilities, formats.Time, now)
	}

	packagesToUpdate := make([]scanner.Module, 0, len(direct)+len(indirect)+len(transitive))
	packagesToUpdate = append(packagesToUpdate, direct...)
	packagesToUpdate = append(packagesToUpdate, indirect...)
	if opts.All {
		packagesToUpdate = append(packagesToUpdate, transitive...)
	}

	if opts.Upgrade {
		var updaterInstance updater.Updater
		if deps.Updater != nil {
			updaterInstance = deps.Updater
		} else {
			updaterInstance, err = factory.CreateUpdater(pm, workDir)
			if err != nil {
				return err
			}
		}

		_, _ = fmt.Fprintln(deps.Out, "\nUpgrading...")
		if err := updaterInstance.UpdatePackages(packagesToUpdate); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(deps.Out, "Done.")
		return nil
	}

	_, _ = fmt.Fprintln(deps.Out, "\nRun with -u to upgrade, or -i for interactive mode.")
	return nil
}

// getGroupLabels returns appropriate group labels based on the package manager.
func getGroupLabels(pm detector.PackageManager) (direct, indirect, transitive string) {
	switch pm {
	case detector.Go:
		return "Direct dependencies (go.mod)",
			"Indirect dependencies (go.mod // indirect)",
			"Transitive (not in go.mod)"
	case detector.Npm, detector.Yarn, detector.Pnpm:
		return "Dependencies (package.json)",
			"DevDependencies (package.json)",
			"Transitive"
	case detector.Pip:
		return "Main dependencies (requirements.txt)",
			"Transitive",
			"Transitive"
	case detector.Poetry, detector.Uv:
		return "Main dependencies",
			"Dev dependencies",
			"Transitive"
	default:
		return "Direct dependencies",
			"Indirect dependencies",
			"Transitive"
	}
}

// formatVulnCounts creates a compact string showing vulnerability transitions
// e.g., "[L (1), M (2), H (2)] → [L (0)]" or just "[L (1), M (2)]" if no update info
func formatVulnCounts(current, update scanner.VulnInfo) string {
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	currentStr := style.FormatVulnInfo(current)
	if currentStr == "" {
		return ""
	}

	updateStr := style.FormatVulnInfo(update)

	// Show transition with arrow
	fixed := current.Total - update.Total

	if fixed > 0 {
		// Vulnerabilities were fixed
		if updateStr == "" {
			return fmt.Sprintf("%s → %s", currentStr, green.Render(fmt.Sprintf("✓ (fixes %d)", fixed)))
		}
		return fmt.Sprintf("%s → %s %s", currentStr, updateStr, green.Render(fmt.Sprintf("(fixes %d)", fixed)))
	} else if fixed < 0 {
		// More vulnerabilities in update
		return fmt.Sprintf("%s → %s %s", currentStr, updateStr, red.Render(fmt.Sprintf("(+%d)", -fixed)))
	} else if update.Total > 0 {
		// Same count but might be different types
		return fmt.Sprintf("%s → %s", currentStr, updateStr)
	}

	// No change or no update checked
	return currentStr
}
