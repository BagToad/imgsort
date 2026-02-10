package categories

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveWithCLICategories(t *testing.T) {
	cli := []string{"cats", "dogs", "birds"}
	result, err := Resolve(cli)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 categories, got %d", len(result))
	}
	if result[0] != "cats" {
		t.Errorf("expected 'cats', got %q", result[0])
	}
}

func TestResolveDefaults(t *testing.T) {
	result, err := Resolve(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != len(DefaultCategories) {
		t.Errorf("expected %d default categories, got %d", len(DefaultCategories), len(result))
	}
}

func TestLoadCustomCategories(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	dir := filepath.Join(tmpHome, ".imgsort")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	content := "nature\n# this is a comment\nanimals\n  food  \n\narchitecture\n"
	if err := os.WriteFile(filepath.Join(dir, "categories.txt"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cats, err := LoadCustomCategories()
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"nature", "animals", "food", "architecture"}
	if len(cats) != len(expected) {
		t.Fatalf("expected %d categories, got %d: %v", len(expected), len(cats), cats)
	}
	for i, c := range expected {
		if cats[i] != c {
			t.Errorf("category %d: expected %q, got %q", i, c, cats[i])
		}
	}
}

func TestLoadCustomCategoriesNoFile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	cats, err := LoadCustomCategories()
	if err != nil {
		t.Fatal(err)
	}
	if cats != nil {
		t.Errorf("expected nil for missing file, got %v", cats)
	}
}

func TestDefaultCategoriesNotEmpty(t *testing.T) {
	if len(DefaultCategories) < 50 {
		t.Errorf("expected at least 50 default categories, got %d", len(DefaultCategories))
	}
}
