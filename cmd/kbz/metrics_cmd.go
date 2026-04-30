package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/sambeau/kanbanzai/internal/actionlog"
)

// entityFeatureLookup implements actionlog.StageFeatureLookup using the entity service.
type entityFeatureLookup struct {
	svc entityService
}

// ListFeaturesInRange returns feature metrics data filtered by time range and optional feature ID.
func (l *entityFeatureLookup) ListFeaturesInRange(since, until time.Time, featureID string) ([]actionlog.FeatureMetricsData, error) {
	results, err := l.svc.List("feature")
	if err != nil {
		return nil, fmt.Errorf("list features: %w", err)
	}

	var features []actionlog.FeatureMetricsData
	for _, r := range results {
		if featureID != "" && r.ID != featureID {
			continue
		}

		ts := featureTimestamp(r.State)
		if !ts.IsZero() {
			if (!since.IsZero() && ts.Before(since)) || (!until.IsZero() && ts.After(until)) {
				continue
			}
		}

		rc, _ := r.State["review_cycle"].(int)
		features = append(features, actionlog.FeatureMetricsData{
			FeatureID:    r.ID,
			DisplayID:    r.ID,
			ReviewCycles: rc,
			Transitions:  nil,
		})
	}

	return features, nil
}

// featureTimestamp returns the most recent timestamp available in a feature state map.
// It prefers "updated" over "created".
func featureTimestamp(state map[string]any) time.Time {
	for _, key := range []string{"updated", "created"} {
		if s, _ := state[key].(string); s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

// runMetrics implements the kbz metrics command.
func runMetrics(args []string, deps dependencies) error {
	fs := flag.NewFlagSet("metrics", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var (
		sinceStr  = fs.String("since", "", "Start of time range (RFC 3339 or YYYY-MM-DD)")
		untilStr  = fs.String("until", "", "End of time range (RFC 3339 or YYYY-MM-DD)")
		featureID = fs.String("feature", "", "Filter by feature ID")
		jsonOut   = fs.Bool("json", false, "Output as JSON")
	)

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			fmt.Fprint(deps.stdout, metricsUsageText)
			return nil
		}
		return fmt.Errorf("parse flags: %w", err)
	}

	now := time.Now().UTC()
	since := now.AddDate(0, 0, -30)
	until := now

	if *sinceStr != "" {
		t, err := parseTimeArg(*sinceStr)
		if err != nil {
			return fmt.Errorf("--since: %w", err)
		}
		since = t
	}

	if *untilStr != "" {
		t, err := parseTimeArg(*untilStr)
		if err != nil {
			return fmt.Errorf("--until: %w", err)
		}
		until = t
	}

	input := actionlog.MetricsInput{
		LogsDir:   actionlog.LogsDir(),
		Since:     since,
		Until:     until,
		FeatureID: *featureID,
	}

	lookup := &entityFeatureLookup{svc: deps.newEntityService("")}
	result, err := actionlog.ComputeMetrics(input, lookup)
	if err != nil {
		return fmt.Errorf("compute metrics: %w", err)
	}

	if result.GateFailureRate.Total == 0 && len(result.TimePerStage) == 0 {
		fmt.Fprintln(deps.stdout, "no data found for the specified time range")
		return nil
	}

	if *jsonOut {
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}
		_, err = fmt.Fprintln(deps.stdout, string(b))
		return err
	}

	return printMetricsText(deps.stdout, result, since, until)
}

// printMetricsText renders a human-readable metrics summary.
func printMetricsText(w io.Writer, result *actionlog.MetricsResult, since, until time.Time) error {
	fmt.Fprintf(w, "metrics: %s to %s\n", since.Format("2006-01-02"), until.Format("2006-01-02"))
	fmt.Fprintln(w)
	fmt.Fprintf(w, "gate_failure_rate: %d/%d (%.1f%%)\n",
		result.GateFailureRate.Count,
		result.GateFailureRate.Total,
		result.GateFailureRate.Rate*100,
	)

	if len(result.TimePerStage) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "time_per_stage:")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  stage\tmedian_hours\tp90_hours\tcount")
		for _, s := range result.TimePerStage {
			fmt.Fprintf(tw, "  %s\t%.1f\t%.1f\t%d\n", s.Stage, s.Median, s.P90, s.Count)
		}
		tw.Flush()
	}

	if len(result.RevisionCycleCounts) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "revision_cycles:")
		for _, f := range result.RevisionCycleCounts {
			fmt.Fprintf(w, "  %s: %d\n", f.DisplayID, f.ReviewCycles)
		}
	}

	return nil
}

// parseTimeArg parses a time argument as RFC 3339 or YYYY-MM-DD.
func parseTimeArg(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("unrecognised time format %q; use YYYY-MM-DD or RFC 3339", s)
}

const metricsUsageText = `kanbanzai metrics [flags]

Flags:
  --since <date>     Start of time range (default: 30 days ago)
  --until <date>     End of time range (default: now)
  --feature <id>     Filter by feature ID
  --json             Output as JSON
`
