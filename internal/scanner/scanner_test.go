package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScan(t *testing.T) {
	dir := t.TempDir()

	// Create test files
	imageFiles := []string{"photo.jpg", "image.png", "pic.gif", "shot.bmp", "web.webp", "scan.tiff"}
	nonImageFiles := []string{"readme.txt", "data.csv", "script.sh"}

	for _, f := range imageFiles {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("fake"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range nonImageFiles {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("fake"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create a subdirectory (should be ignored)
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}

	result, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(result.ImagePaths) != len(imageFiles) {
		t.Errorf("expected %d images, got %d", len(imageFiles), len(result.ImagePaths))
	}

	if result.SkippedCount != len(nonImageFiles) {
		t.Errorf("expected %d skipped, got %d", len(nonImageFiles), result.SkippedCount)
	}
}

func TestScanCaseInsensitive(t *testing.T) {
	dir := t.TempDir()

	files := []string{"PHOTO.JPG", "Image.PNG", "pic.JPEG"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("fake"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	result, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(result.ImagePaths) != len(files) {
		t.Errorf("expected %d images, got %d", len(files), len(result.ImagePaths))
	}
}

func TestScanNoImages(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Scan(dir)
	if err == nil {
		t.Error("expected error for directory with no images")
	}
}

func TestScanNonexistentDir(t *testing.T) {
	_, err := Scan("/nonexistent/path/12345")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestScanNotADir(t *testing.T) {
	f, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.Close()

	_, err = Scan(f.Name())
	if err == nil {
		t.Error("expected error for file (not directory)")
	}
}

func TestScanSkipsHiddenFiles(t *testing.T) {
	dir := t.TempDir()

	// Visible image
	if err := os.WriteFile(filepath.Join(dir, "photo.jpg"), []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}
	// Hidden image files (should be ignored entirely, not even counted as skipped)
	if err := os.WriteFile(filepath.Join(dir, ".hidden.jpg"), []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".DS_Store"), []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(result.ImagePaths) != 1 {
		t.Errorf("expected 1 image, got %d: %v", len(result.ImagePaths), result.ImagePaths)
	}
	if result.SkippedCount != 0 {
		t.Errorf("expected 0 skipped (hidden files should be ignored), got %d", result.SkippedCount)
	}
}
