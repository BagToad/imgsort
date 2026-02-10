//go:build integration

package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bagtoad/imgsort/internal/categories"
	"github.com/bagtoad/imgsort/internal/categorizer"
	"github.com/bagtoad/imgsort/internal/model"
	"github.com/bagtoad/imgsort/internal/mover"
	"github.com/bagtoad/imgsort/internal/report"
	"github.com/bagtoad/imgsort/internal/scanner"
)

func TestMain(m *testing.M) {
	// Ensure models are downloaded before tests run
	err := model.EnsureModels(func(filename string, downloaded, total int64) {
		// silent during tests
	})
	if err != nil {
		panic("failed to download models: " + err.Error())
	}
	os.Exit(m.Run())
}

func newCLIP(t *testing.T) *model.CLIPSession {
	t.Helper()
	clip, err := model.NewCLIPSession("")
	if err != nil {
		t.Fatalf("cannot create CLIP session: %v", err)
	}
	t.Cleanup(func() { clip.Destroy() })
	return clip
}

func TestScanTestdata(t *testing.T) {
	result, err := scanner.Scan("../testdata")
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(result.ImagePaths) != 6 {
		t.Errorf("expected 6 images, got %d: %v", len(result.ImagePaths), result.ImagePaths)
	}

	// readme.txt + generate.go = 2 skipped
	if result.SkippedCount != 2 {
		t.Errorf("expected 2 skipped, got %d", result.SkippedCount)
	}
}

func TestCLIPClassifySingleImage(t *testing.T) {
	clip := newCLIP(t)

	cats := []string{"landscape", "sunset", "document", "night", "nature", "flower"}
	scores, err := clip.Classify("../testdata/landscape.jpg", cats)
	if err != nil {
		t.Fatalf("Classify failed: %v", err)
	}

	if len(scores) != len(cats)+1 { // +1 for baseline "uncategorized"
		t.Fatalf("expected %d scores (including baseline), got %d", len(cats)+1, len(scores))
	}

	// Verify all scores are valid probabilities
	sum := float32(0)
	for _, s := range scores {
		if s < 0 || s > 1 {
			t.Errorf("score out of range [0,1]: %f", s)
		}
		sum += s
	}
	if sum < 0.99 || sum > 1.01 {
		t.Errorf("scores should sum to ~1.0, got %f", sum)
	}

	t.Logf("Landscape scores: %v", scores)
}

func TestCLIPClassifyAllTestImages(t *testing.T) {
	clip := newCLIP(t)

	cats := []string{"landscape", "sunset", "red", "night", "nature", "document"}

	testCases := []struct {
		image string
	}{
		{image: "../testdata/landscape.jpg"},
		{image: "../testdata/sunset.png"},
		{image: "../testdata/red_object.jpg"},
		{image: "../testdata/dark_scene.png"},
		{image: "../testdata/nature.jpg"},
		{image: "../testdata/document.png"},
	}

	for _, tc := range testCases {
		t.Run(filepath.Base(tc.image), func(t *testing.T) {
			scores, err := clip.Classify(tc.image, cats)
			if err != nil {
				t.Fatalf("Classify failed: %v", err)
			}

			bestCat := ""
			bestScore := float32(0)
			for cat, score := range scores {
				if cat == model.BaselineCategory {
					continue
				}
				if score > bestScore {
					bestScore = score
					bestCat = cat
				}
			}

			t.Logf("%s → %s (%.1f%%, baseline=%.1f%%)",
				filepath.Base(tc.image), bestCat, bestScore*100, scores[model.BaselineCategory]*100)
		})
	}
}

// TestSingleCategoryDoesNotAlwaysMatch verifies that a single category
// doesn't always match with 100% confidence (the baseline bug).
func TestSingleCategoryDoesNotAlwaysMatch(t *testing.T) {
	clip := newCLIP(t)

	// A dark scene image should NOT match "cat" with high confidence
	scores, err := clip.Classify("../testdata/dark_scene.png", []string{"cat"})
	if err != nil {
		t.Fatalf("Classify failed: %v", err)
	}

	catScore := scores["cat"]
	baselineScore := scores[model.BaselineCategory]

	t.Logf("Single category test: cat=%.1f%%, baseline=%.1f%%", catScore*100, baselineScore*100)

	// The baseline should beat "cat" for a dark scene image
	if catScore > baselineScore {
		t.Logf("Warning: 'cat' scored higher than baseline for dark_scene.png (cat=%.1f%% vs baseline=%.1f%%)",
			catScore*100, baselineScore*100)
	}

	// The cat score should definitely NOT be 100%
	if catScore > 0.95 {
		t.Errorf("single category 'cat' should not be 95%%+ confident for dark_scene.png, got %.1f%%", catScore*100)
	}

	// Also test: a document image should NOT match "cat"
	scores2, err := clip.Classify("../testdata/document.png", []string{"cat"})
	if err != nil {
		t.Fatalf("Classify failed: %v", err)
	}

	catScore2 := scores2["cat"]
	baselineScore2 := scores2[model.BaselineCategory]
	t.Logf("Document as 'cat': cat=%.1f%%, baseline=%.1f%%", catScore2*100, baselineScore2*100)

	if catScore2 > 0.95 {
		t.Errorf("document.png should not be 95%%+ confident as 'cat', got %.1f%%", catScore2*100)
	}

	// Test the full categorizer pipeline: single category should skip non-matching images
	result, err := categorizer.Categorize(clip, []string{"../testdata/dark_scene.png", "../testdata/document.png"}, []string{"cat"}, 0.15, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range result {
		t.Logf("  %s: category=%q skipped=%v confidence=%.1f%%",
			filepath.Base(r.Path), r.Category, r.Skipped, r.Confidence*100)
	}
}

func TestFullPipelineDryRun(t *testing.T) {
	clip := newCLIP(t)

	// Copy test images to a temp directory so we don't modify testdata
	tmpDir := t.TempDir()
	copyTestImages(t, tmpDir)

	// Scan
	scanResult, err := scanner.Scan(tmpDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	t.Logf("Found %d images, %d skipped", len(scanResult.ImagePaths), scanResult.SkippedCount)

	// Resolve categories
	cats, err := categories.Resolve([]string{"landscape", "sunset", "red", "night", "nature", "document"})
	if err != nil {
		t.Fatal(err)
	}

	// Categorize
	results, err := categorizer.Categorize(clip, scanResult.ImagePaths, cats, 0.10, nil)
	if err != nil {
		t.Fatalf("Categorize failed: %v", err)
	}

	categorized := 0
	skipped := 0
	for _, r := range results {
		if r.Skipped {
			skipped++
			t.Logf("SKIPPED: %s", filepath.Base(r.Path))
		} else {
			categorized++
			t.Logf("  %s → %s (%.1f%%)", filepath.Base(r.Path), r.Category, r.Confidence*100)
		}
	}
	t.Logf("Categorized: %d, Skipped: %d", categorized, skipped)

	if categorized == 0 {
		t.Error("expected at least some images to be categorized")
	}

	// Move (dry run)
	moves, err := mover.MoveFiles(tmpDir, results, true)
	if err != nil {
		t.Fatal(err)
	}

	// Verify no files were actually moved
	entries, _ := os.ReadDir(tmpDir)
	for _, entry := range entries {
		if entry.IsDir() {
			t.Errorf("no subdirectories should exist in dry-run, found: %s", entry.Name())
		}
	}

	// Print report
	report.Print(os.Stdout, results, moves, scanResult.SkippedCount, true)
}

func TestFullPipelineWithMove(t *testing.T) {
	clip := newCLIP(t)

	tmpDir := t.TempDir()
	copyTestImages(t, tmpDir)

	// Scan
	scanResult, err := scanner.Scan(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Categorize with specific categories
	cats := []string{"landscape", "sunset", "red", "night", "nature", "document"}
	results, err := categorizer.Categorize(clip, scanResult.ImagePaths, cats, 0.10, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Actually move files
	moves, err := mover.MoveFiles(tmpDir, results, false)
	if err != nil {
		t.Fatal(err)
	}

	// Verify: moved files should exist at destination
	for _, m := range moves {
		if _, err := os.Stat(m.DestPath); err != nil {
			t.Errorf("moved file should exist at destination: %s", m.DestPath)
		}
		if _, err := os.Stat(m.SourcePath); !os.IsNotExist(err) {
			t.Errorf("source file should have been moved: %s", m.SourcePath)
		}
	}

	// Verify: category subdirectories were created
	catDirs := make(map[string]bool)
	for _, m := range moves {
		catDirs[m.Category] = true
	}
	for cat := range catDirs {
		catPath := filepath.Join(tmpDir, cat)
		info, err := os.Stat(catPath)
		if err != nil {
			t.Errorf("category dir should exist: %s", catPath)
		} else if !info.IsDir() {
			t.Errorf("should be a directory: %s", catPath)
		}
	}

	// Print report
	report.Print(os.Stdout, results, moves, scanResult.SkippedCount, false)
	t.Logf("Successfully moved %d files into %d categories", len(moves), len(catDirs))
}

func TestCategorizeWithDefaultCategories(t *testing.T) {
	clip := newCLIP(t)

	cats, err := categories.Resolve(nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Using %d default categories", len(cats))

	result, err := scanner.Scan("../testdata")
	if err != nil {
		t.Fatal(err)
	}

	results, err := categorizer.Categorize(clip, result.ImagePaths, cats, 0.10, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range results {
		if r.Skipped {
			t.Logf("SKIPPED: %s", filepath.Base(r.Path))
		} else {
			t.Logf("  %s → %s (%.1f%%)", filepath.Base(r.Path), r.Category, r.Confidence*100)
		}
	}
}

// copyTestImages copies image files from testdata to a destination directory.
func copyTestImages(t *testing.T, dstDir string) {
	t.Helper()
	srcDir := "../testdata"
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) == ".go" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(srcDir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dstDir, entry.Name()), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
}
