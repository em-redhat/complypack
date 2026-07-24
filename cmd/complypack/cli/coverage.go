// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/complytime/complypack/internal/cache"
	"github.com/complytime/complypack/internal/config"
	"github.com/complytime/complypack/internal/coverage"
	"github.com/complytime/complypack/internal/evaluator"
	"github.com/complytime/complypack/internal/requirement"
	"github.com/complytime/complypack/internal/source"
	"github.com/spf13/cobra"
)

var (
	styleTitle   = lipgloss.NewStyle().Bold(true)
	styleControl = lipgloss.NewStyle().Bold(true)
	stylePass    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	styleFail    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	styleGap     = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	styleOK      = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	styleWarn    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	styleDim     = lipgloss.NewStyle().Faint(true)
)

func coverageCmd() *cobra.Command {
	var (
		policyName string
		policyDir  string
		configPath string
		cacheDir   string
		evalID     string
		runTests   bool
		output     string
		sources    []string
	)

	cmd := &cobra.Command{
		Use:   "coverage",
		Short: "Generate a coverage report comparing policy requirements against enforcement artifacts",
		Long: `Compare a policy's in-scope assessment requirements against enforcement
artifacts in a directory, producing a structured coverage report.

Requirements are classified into three buckets:
  - Implemented (passing) — enforcement artifact exists, tests pass
  - Implemented (failing) — enforcement artifact exists, tests fail
  - Gap — no enforcement artifact exists

Examples:
  complypack coverage --policy my-policy --policy-dir ./policy --config complypack.yaml
  complypack coverage --policy my-policy --policy-dir ./policy --source oci://ghcr.io/org/catalog:v1
  complypack coverage --policy my-policy --policy-dir ./policy --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCoverage(cmd, coverageRunParams{
				policyName: policyName,
				policyDir:  policyDir,
				configPath: configPath,
				cacheDir:   cacheDir,
				evalID:     evalID,
				runTests:   runTests,
				output:     output,
				sources:    sources,
				stdout:     cmd.OutOrStdout(),
			})
		},
	}

	cmd.Flags().StringVar(&policyName, "policy", "", "Policy name to check coverage for (required)")
	cmd.Flags().StringVar(&policyDir, "policy-dir", "", "Path to directory containing enforcement artifacts (required)")
	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to complypack.yaml config file")
	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", cache.CacheDirHelp)
	cmd.Flags().StringVar(&evalID, "evaluator", "", "Evaluator ID (auto-detected if omitted)")
	cmd.Flags().BoolVar(&runTests, "run-tests", false, "Execute tests for pass/fail enrichment")
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format: text or json")
	cmd.Flags().StringArrayVar(&sources, "source", nil, "Gemara OCI source (repeatable)")

	_ = cmd.MarkFlagRequired("policy")
	_ = cmd.MarkFlagRequired("policy-dir")

	return cmd
}

// coverageRunParams holds parsed CLI parameters for the coverage command.
type coverageRunParams struct {
	policyName string
	policyDir  string
	configPath string
	cacheDir   string
	evalID     string
	runTests   bool
	output     string
	sources    []string
	stdout     io.Writer
}

func runCoverage(cmd *cobra.Command, params coverageRunParams) error {
	ctx := cmd.Context()

	// Resolve cache directory
	resolvedCacheDir, err := cache.ResolveDir(params.cacheDir)
	if err != nil {
		return fmt.Errorf("failed to resolve cache directory: %w", err)
	}

	// Load artifacts from config or flags
	loaded := requirement.NewArtifactSet()
	if len(params.sources) > 0 {
		for _, s := range params.sources {
			src, err := source.LoadArtifacts(ctx, s, false, resolvedCacheDir)
			if err != nil {
				return fmt.Errorf("failed to load artifacts from %s: %w", s, err)
			}
			if err := loaded.Merge(src); err != nil {
				return fmt.Errorf("failed to merge artifacts from %s: %w", s, err)
			}
		}
	} else {
		cfgPath := params.configPath
		if cfgPath == "" {
			cfgPath = "complypack.yaml"
		}
		cfg, err := config.LoadConfig(cfgPath, false, os.Stderr)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		for _, entry := range cfg.Gemara.Sources {
			src, err := source.LoadArtifacts(ctx, entry.Source, entry.PlainHTTP, resolvedCacheDir)
			if err != nil {
				return fmt.Errorf("failed to load artifacts from %s: %w", entry.Source, err)
			}
			if err := loaded.Merge(src); err != nil {
				return fmt.Errorf("failed to merge artifacts from %s: %w", entry.Source, err)
			}
		}
	}

	// Resolve the named policy
	policy, ok := loaded.Policies[params.policyName]
	if !ok {
		return fmt.Errorf("policy %q not found in loaded artifacts", params.policyName)
	}
	rp, err := requirement.ResolvePolicy(*policy, loaded)
	if err != nil {
		return fmt.Errorf("failed to resolve policy %q: %w", params.policyName, err)
	}

	// Resolve evaluator
	evalRegistry := evaluator.DefaultRegistry()
	var eval evaluator.Evaluator
	if params.evalID != "" {
		eval, err = evalRegistry.Get(params.evalID)
		if err != nil {
			return fmt.Errorf("evaluator %q: %w", params.evalID, err)
		}
	} else {
		ids := evalRegistry.IDs()
		if len(ids) == 0 {
			return fmt.Errorf("no evaluators registered")
		}
		if len(ids) > 1 {
			return fmt.Errorf("multiple evaluators available (%s); use --evaluator to select one",
				strings.Join(ids, ", "))
		}
		eval, _ = evalRegistry.Get(ids[0])
	}

	// Run coverage engine
	report, err := coverage.Run(ctx, coverage.Options{
		ResolvedPolicy: rp,
		PolicyDir:      params.policyDir,
		Evaluator:      eval,
		RunTests:       params.runTests,
	})
	if err != nil {
		return fmt.Errorf("coverage analysis failed: %w", err)
	}

	// Format output
	switch params.output {
	case "json":
		return writeJSON(params.stdout, report)
	case "text", "":
		return writeText(params.stdout, report)
	default:
		return fmt.Errorf("unknown output format %q; use text or json", params.output)
	}
}

// writeJSON marshals the report as indented JSON.
func writeJSON(w io.Writer, report *coverage.Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

// writeText formats the report as human-readable text grouped by control.
func writeText(w io.Writer, report *coverage.Report) error {
	fmt.Fprintln(w, styleTitle.Render(fmt.Sprintf("Coverage Report: %s", report.PolicyID)))
	fmt.Fprintln(w, styleDim.Render(strings.Repeat("━", 50)))

	type controlGroup struct {
		controlID    string
		requirements []coverage.RequirementEntry
	}
	groupMap := make(map[string]*controlGroup)
	var groupOrder []string

	for _, req := range report.Requirements {
		cid := req.ControlID
		if cid == "" {
			cid = "(ungrouped)"
		}
		if _, ok := groupMap[cid]; !ok {
			groupMap[cid] = &controlGroup{controlID: cid}
			groupOrder = append(groupOrder, cid)
		}
		groupMap[cid].requirements = append(groupMap[cid].requirements, req)
	}
	sort.Strings(groupOrder)

	for _, cid := range groupOrder {
		g := groupMap[cid]
		fmt.Fprintf(w, "\n  %s\n", styleControl.Render(g.controlID))
		for _, req := range g.requirements {
			fmt.Fprintf(w, "    %s %s\n", statusIndicator(req.Status), req.RequirementID)
		}
	}

	if len(report.Warnings) > 0 {
		fmt.Fprintln(w)
		for _, warn := range report.Warnings {
			fmt.Fprintf(w, "  %s %s\n", styleWarn.Render("⚠"), warn.Message)
		}
	}

	if len(report.Manual) > 0 {
		fmt.Fprintf(w, "\n  %s\n",
			styleDim.Render(fmt.Sprintf("Manual requirements (excluded from coverage): %d", len(report.Manual))))
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, styleDim.Render(strings.Repeat("─", 50)))

	covStyle := coverageStyle(report.Metrics.CoveragePercent)
	fmt.Fprintf(w, "  %s\n", covStyle.Render(
		fmt.Sprintf("%d/%d requirements covered (%.1f%%)",
			report.Metrics.Implemented, report.Metrics.TotalAutomated,
			report.Metrics.CoveragePercent)))
	if report.Metrics.Passing > 0 || report.Metrics.Failing > 0 {
		fmt.Fprintf(w, "  %s: %d  %s: %d\n",
			stylePass.Render("Passing"), report.Metrics.Passing,
			styleFail.Render("Failing"), report.Metrics.Failing)
	}
	if report.Metrics.Gaps > 0 {
		fmt.Fprintf(w, "  %s: %d\n", styleGap.Render("Gaps"), report.Metrics.Gaps)
	}

	return nil
}

// statusIndicator returns a styled text indicator for a requirement status.
func statusIndicator(status coverage.RequirementStatus) string {
	switch status {
	case coverage.StatusImplementedPassing:
		return stylePass.Render("✓ PASS")
	case coverage.StatusImplementedFailing:
		return styleFail.Render("✗ FAIL")
	case coverage.StatusImplemented:
		return styleOK.Render("● OK  ")
	case coverage.StatusGap:
		return styleGap.Render("○ GAP ")
	default:
		return styleDim.Render("  ?   ")
	}
}

func coverageStyle(pct float64) lipgloss.Style {
	switch {
	case pct >= 80:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	case pct >= 50:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	default:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	}
}
