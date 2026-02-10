// Package mover handles moving image files into category subfolders.
package mover

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bagtoad/imgsort/internal/categorizer"
)

// MoveResult records what happened to a single file.
type MoveResult struct {
	SourcePath string
	DestPath   string
	Category   string
}

// MoveFiles moves categorized images into category subfolders within baseDir.
// If dryRun is true, no files are moved but results are still returned.
func MoveFiles(baseDir string, results []categorizer.Result, dryRun bool) ([]MoveResult, error) {
	groups := categorizer.GroupByCategory(results)
	var moveResults []MoveResult

	for category, items := range groups {
		catDir := filepath.Join(baseDir, category)

		if !dryRun {
			if err := os.MkdirAll(catDir, 0755); err != nil {
				return nil, fmt.Errorf("cannot create category folder %q: %w", catDir, err)
			}
		}

		for _, item := range items {
			destPath := filepath.Join(catDir, filepath.Base(item.Path))
			destPath = resolveConflict(destPath, dryRun)

			if !dryRun {
				if err := os.Rename(item.Path, destPath); err != nil {
					return nil, fmt.Errorf("cannot move %s to %s: %w", item.Path, destPath, err)
				}
			}

			moveResults = append(moveResults, MoveResult{
				SourcePath: item.Path,
				DestPath:   destPath,
				Category:   category,
			})
		}
	}

	return moveResults, nil
}

// resolveConflict appends a numeric suffix if a file already exists at destPath.
func resolveConflict(destPath string, dryRun bool) string {
	if dryRun {
		return destPath
	}

	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		return destPath
	}

	ext := filepath.Ext(destPath)
	base := strings.TrimSuffix(destPath, ext)

	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s_%d%s", base, i, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}
