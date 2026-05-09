package mcp

import (
	"fmt"
	"strings"

	"github.com/sambeau/kanbanzai/internal/binding"
	"github.com/sambeau/kanbanzai/internal/validate"
)

// BindingLoadableHealthChecker returns an AdditionalHealthChecker that calls
// LoadBindingFile and reports any load errors as binding_loadable warnings.
// AC-007: malformed binding produces a warning; AC-008: valid binding reports ok.
func BindingLoadableHealthChecker(bindingPath string) AdditionalHealthChecker {
	return func() (*validate.HealthReport, error) {
		report := &validate.HealthReport{
			Summary: validate.HealthSummary{
				EntitiesByType: make(map[string]int),
			},
		}

		if bindingPath == "" {
			report.Warnings = append(report.Warnings, validate.ValidationWarning{
				EntityType: "binding_loadable",
				Message:    "binding file path is not configured",
			})
			report.Summary.WarningCount++
			return report, nil
		}

		_, errs := binding.LoadBindingFile(bindingPath)
		if len(errs) > 0 {
			msgs := make([]string, len(errs))
			for i, e := range errs {
				msgs[i] = e.Error()
			}
			report.Warnings = append(report.Warnings, validate.ValidationWarning{
				EntityType: "binding_loadable",
				Message:    fmt.Sprintf("stage-bindings.yaml failed to load: %s", strings.Join(msgs, "; ")),
			})
			report.Summary.WarningCount++
		}

		return report, nil
	}
}
