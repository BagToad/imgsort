package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bagtoad/imgsort/internal/categorizer"
	"github.com/bagtoad/imgsort/internal/mover"
)

func TestPrintReport(t *testing.T) {
	results := []categorizer.Result{
		{Path: "/imgs/beach.jpg", Category: "landscape", Confidence: 0.8},
		{Path: "/imgs/cat.png", Category: "animals", Confidence: 0.9},
		{Path: "/imgs/blur.jpg", Skipped: true},
	}

	moves := []mover.MoveResult{
		{SourcePath: "/imgs/beach.jpg", DestPath: "/imgs/landscape/beach.jpg", Category: "landscape"},
		{SourcePath: "/imgs/cat.png", DestPath: "/imgs/animals/cat.png", Category: "animals"},
	}

	var buf bytes.Buffer
	Print(&buf, results, moves, 5, false)

	output := buf.String()

	// Check key parts of the report
	checks := []string{
		"Images found:        3",
		"Images categorized:  2",
		"Images skipped:      1",
		"Non-image files:     5",
		"Categories:          2",
		"landscape/ (1 files)",
		"animals/ (1 files)",
		"Moved",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("report missing %q\nFull output:\n%s", check, output)
		}
	}
}

func TestPrintReportDryRun(t *testing.T) {
	results := []categorizer.Result{
		{Path: "/imgs/beach.jpg", Category: "landscape", Confidence: 0.8},
	}

	moves := []mover.MoveResult{
		{SourcePath: "/imgs/beach.jpg", DestPath: "/imgs/landscape/beach.jpg", Category: "landscape"},
	}

	var buf bytes.Buffer
	Print(&buf, results, moves, 0, true)

	output := buf.String()

	if !strings.Contains(output, "Dry Run Summary") {
		t.Errorf("expected dry run header in output:\n%s", output)
	}
	if !strings.Contains(output, "Would move") {
		t.Errorf("expected 'Would move' in dry run output:\n%s", output)
	}
}

func TestPrintReportEmpty(t *testing.T) {
	var buf bytes.Buffer
	Print(&buf, nil, nil, 0, false)

	output := buf.String()
	if !strings.Contains(output, "No files to move") {
		t.Errorf("expected empty message in output:\n%s", output)
	}
}
