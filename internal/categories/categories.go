// Package categories provides the default and custom category lists for classification.
package categories

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultCategories is the built-in list of common photo categories.
var DefaultCategories = []string{
	// People & Social
	"people", "portrait", "selfie", "group photo", "baby", "wedding", "family",
	// Animals
	"dog", "cat", "bird", "wildlife", "pet", "fish", "insect",
	// Nature & Landscapes
	"landscape", "mountain", "forest", "ocean", "lake", "river", "waterfall",
	"desert", "field", "garden", "park", "sunrise", "sunset", "sky", "clouds",
	// Urban & Architecture
	"city", "building", "skyscraper", "bridge", "street", "house", "church",
	"castle", "monument", "ruins",
	// Food & Drink
	"food", "dessert", "coffee", "cocktail", "fruit", "meal",
	// Travel & Transport
	"car", "airplane", "boat", "train", "bicycle", "motorcycle", "road",
	"airport", "harbor",
	// Activities & Sports
	"sports", "hiking", "swimming", "skiing", "concert", "festival", "party",
	// Art & Creative
	"art", "painting", "sculpture", "graffiti", "illustration", "calligraphy",
	// Indoor & Objects
	"indoor", "furniture", "electronics", "book", "toy", "instrument",
	"clothing", "jewelry",
	// Documents & Screenshots
	"document", "screenshot", "whiteboard", "diagram", "chart", "map", "sign",
	"receipt", "menu",
	// Miscellaneous
	"flower", "tree", "night", "fireworks", "snow", "rain", "fog",
	"abstract", "pattern", "texture", "macro", "aerial",
}

// configPath returns the path to the user's custom categories file.
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".imgsort", "categories.txt"), nil
}

// LoadCustomCategories reads categories from ~/.imgsort/categories.txt.
// Returns nil if the file does not exist.
func LoadCustomCategories() ([]string, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot open categories file: %w", err)
	}
	defer f.Close()

	var categories []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			categories = append(categories, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading categories file: %w", err)
	}

	return categories, nil
}

// Resolve returns the final list of categories to use for classification.
// Priority: CLI flag > custom file > defaults.
func Resolve(cliCategories []string) ([]string, error) {
	if len(cliCategories) > 0 {
		return cliCategories, nil
	}

	custom, err := LoadCustomCategories()
	if err != nil {
		return nil, err
	}
	if len(custom) > 0 {
		return custom, nil
	}

	return DefaultCategories, nil
}
