// Package scanner provides directory scanning and image file filtering.
package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SupportedExtensions contains the set of image file extensions we process.
var SupportedExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".bmp":  true,
	".webp": true,
	".tiff": true,
	".tif":  true,
}

// Result holds the output of scanning a directory.
type Result struct {
	ImagePaths   []string
	SkippedCount int
}

// Scan walks the given directory (non-recursive) and returns image file paths
// and a count of skipped non-image files.
func Scan(dir string) (*Result, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot access directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot read directory: %w", err)
	}

	result := &Result{}
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if SupportedExtensions[ext] {
			result.ImagePaths = append(result.ImagePaths, filepath.Join(dir, entry.Name()))
		} else {
			result.SkippedCount++
		}
	}

	if len(result.ImagePaths) == 0 {
		return nil, fmt.Errorf("no image files found in %s", dir)
	}

	return result, nil
}
