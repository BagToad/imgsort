package mover

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bagtoad/imgsort/internal/categorizer"
)

func TestMoveFiles(t *testing.T) {
	dir := t.TempDir()

	// Create test image files
	files := []string{"beach.jpg", "city.jpg", "food.png"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("fake image"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	results := []categorizer.Result{
		{Path: filepath.Join(dir, "beach.jpg"), Category: "landscape", Confidence: 0.8},
		{Path: filepath.Join(dir, "city.jpg"), Category: "city", Confidence: 0.7},
		{Path: filepath.Join(dir, "food.png"), Category: "food", Confidence: 0.9},
	}

	moves, err := MoveFiles(dir, results, false)
	if err != nil {
		t.Fatal(err)
	}

	if len(moves) != 3 {
		t.Errorf("expected 3 moves, got %d", len(moves))
	}

	// Verify files were moved
	for _, m := range moves {
		if _, err := os.Stat(m.DestPath); err != nil {
			t.Errorf("destination file missing: %s", m.DestPath)
		}
		if _, err := os.Stat(m.SourcePath); !os.IsNotExist(err) {
			t.Errorf("source file should no longer exist: %s", m.SourcePath)
		}
	}

	// Verify category dirs were created
	for _, cat := range []string{"landscape", "city", "food"} {
		catDir := filepath.Join(dir, cat)
		info, err := os.Stat(catDir)
		if err != nil {
			t.Errorf("category dir missing: %s", catDir)
		} else if !info.IsDir() {
			t.Errorf("%s is not a directory", catDir)
		}
	}
}

func TestMoveFilesDryRun(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "test.jpg"), []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	results := []categorizer.Result{
		{Path: filepath.Join(dir, "test.jpg"), Category: "nature", Confidence: 0.5},
	}

	moves, err := MoveFiles(dir, results, true)
	if err != nil {
		t.Fatal(err)
	}

	if len(moves) != 1 {
		t.Errorf("expected 1 move result, got %d", len(moves))
	}

	// File should still be at original location
	if _, err := os.Stat(filepath.Join(dir, "test.jpg")); err != nil {
		t.Error("file should not have been moved in dry-run")
	}

	// Category dir should not exist
	if _, err := os.Stat(filepath.Join(dir, "nature")); !os.IsNotExist(err) {
		t.Error("category dir should not exist in dry-run")
	}
}

func TestMoveFilesConflict(t *testing.T) {
	dir := t.TempDir()

	// Create a file and a pre-existing destination
	if err := os.WriteFile(filepath.Join(dir, "photo.jpg"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	catDir := filepath.Join(dir, "nature")
	if err := os.MkdirAll(catDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(catDir, "photo.jpg"), []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}

	results := []categorizer.Result{
		{Path: filepath.Join(dir, "photo.jpg"), Category: "nature", Confidence: 0.5},
	}

	moves, err := MoveFiles(dir, results, false)
	if err != nil {
		t.Fatal(err)
	}

	if len(moves) != 1 {
		t.Fatalf("expected 1 move, got %d", len(moves))
	}

	// Should have been renamed to photo_1.jpg
	expected := filepath.Join(catDir, "photo_1.jpg")
	if moves[0].DestPath != expected {
		t.Errorf("expected dest %s, got %s", expected, moves[0].DestPath)
	}

	// Both files should exist
	if _, err := os.Stat(filepath.Join(catDir, "photo.jpg")); err != nil {
		t.Error("original file should still exist")
	}
	if _, err := os.Stat(expected); err != nil {
		t.Error("renamed file should exist")
	}
}

func TestMoveFilesSkipped(t *testing.T) {
	dir := t.TempDir()

	results := []categorizer.Result{
		{Path: "/fake/path.jpg", Skipped: true},
	}

	moves, err := MoveFiles(dir, results, false)
	if err != nil {
		t.Fatal(err)
	}

	if len(moves) != 0 {
		t.Errorf("expected 0 moves for skipped files, got %d", len(moves))
	}
}
