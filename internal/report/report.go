// Package report generates summary reports of the categorization process.
package report

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"github.com/bagtoad/imgsort/internal/categorizer"
	"github.com/bagtoad/imgsort/internal/mover"
)

// Print writes a summary report to the given writer.
func Print(w io.Writer, results []categorizer.Result, moves []mover.MoveResult, skippedNonImage int, dryRun bool) {
	totalImages := len(results)
	skippedCount := 0
	for _, r := range results {
		if r.Skipped {
			skippedCount++
		}
	}
	categorizedCount := totalImages - skippedCount

	fmt.Fprintln(w)
	if dryRun {
		fmt.Fprintln(w, "=== Dry Run Summary ===")
	} else {
		fmt.Fprintln(w, "=== Summary ===")
	}
	fmt.Fprintf(w, "Images found:        %d\n", totalImages)
	fmt.Fprintf(w, "Images categorized:  %d\n", categorizedCount)
	fmt.Fprintf(w, "Images skipped:      %d\n", skippedCount)
	if skippedNonImage > 0 {
		fmt.Fprintf(w, "Non-image files:     %d\n", skippedNonImage)
	}

	if len(moves) == 0 {
		fmt.Fprintln(w, "\nNo files to move.")
		return
	}

	// Group moves by category
	groups := make(map[string][]mover.MoveResult)
	for _, m := range moves {
		groups[m.Category] = append(groups[m.Category], m)
	}

	// Sort category names
	catNames := make([]string, 0, len(groups))
	for k := range groups {
		catNames = append(catNames, k)
	}
	sort.Strings(catNames)

	fmt.Fprintf(w, "Categories:          %d\n", len(catNames))
	fmt.Fprintln(w)

	verb := "Moved"
	if dryRun {
		verb = "Would move"
	}

	for _, cat := range catNames {
		items := groups[cat]
		fmt.Fprintf(w, "  %s/ (%d files)\n", cat, len(items))
		for _, m := range items {
			fmt.Fprintf(w, "    %s %s → %s\n", verb, filepath.Base(m.SourcePath), m.DestPath)
		}
	}
	fmt.Fprintln(w)
}
