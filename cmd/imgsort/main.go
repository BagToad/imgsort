package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/bagtoad/imgsort/internal/categories"
	"github.com/bagtoad/imgsort/internal/categorizer"
	"github.com/bagtoad/imgsort/internal/model"
	"github.com/bagtoad/imgsort/internal/mover"
	"github.com/bagtoad/imgsort/internal/report"
	"github.com/bagtoad/imgsort/internal/scanner"
	"github.com/spf13/cobra"
)

func main() {
	var dryRun bool
	var categoriesFlag string
	var confidence float64

	rootCmd := &cobra.Command{
		Use:   "imgsort <directory>",
		Short: "Sort images into category folders using a local CLIP AI model",
		Long: `imgsort uses a locally-running CLIP model to categorize images
in a directory and sort them into category-named subfolders.

Images are classified using zero-shot classification against either
a built-in set of common categories, a custom categories file
(~/.imgsort/categories.txt), or categories provided via --categories.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(args[0], dryRun, categoriesFlag, confidence)
		},
	}

	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without moving files")
	rootCmd.Flags().StringVar(&categoriesFlag, "categories", "", "Comma-separated list of categories to classify into")
	rootCmd.Flags().Float64Var(&confidence, "confidence", 0.15, "Minimum confidence threshold for classification (0.0-1.0)")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(dir string, dryRun bool, categoriesFlag string, confidence float64) error {
	// Validate directory
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("cannot access directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	// Resolve categories
	var cliCats []string
	if categoriesFlag != "" {
		for _, c := range strings.Split(categoriesFlag, ",") {
			c = strings.TrimSpace(c)
			if c != "" {
				cliCats = append(cliCats, c)
			}
		}
	}
	cats, err := categories.Resolve(cliCats)
	if err != nil {
		return fmt.Errorf("cannot resolve categories: %w", err)
	}
	fmt.Printf("Using %d categories\n", len(cats))

	// Scan directory
	fmt.Printf("Scanning %s...\n", dir)
	scanResult, err := scanner.Scan(dir)
	if err != nil {
		return err
	}
	fmt.Printf("Found %d images (%d non-image files skipped)\n", len(scanResult.ImagePaths), scanResult.SkippedCount)

	// Ensure models are downloaded
	fmt.Println("Checking AI model...")
	err = model.EnsureModels(func(filename string, downloaded, total int64) {
		if total > 0 {
			pct := float64(downloaded) / float64(total) * 100
			fmt.Printf("\rDownloading %s... %.0f%%", filename, pct)
		} else {
			fmt.Printf("\rDownloading %s... %d bytes", filename, downloaded)
		}
	})
	if err != nil {
		return fmt.Errorf("model setup failed: %w", err)
	}

	// Create CLIP session
	fmt.Println("Loading CLIP model...")
	clip, err := model.NewCLIPSession("")
	if err != nil {
		return fmt.Errorf("cannot load CLIP model: %w", err)
	}
	defer clip.Destroy()

	// Categorize images
	fmt.Println("Categorizing images...")
	results, err := categorizer.Categorize(clip, scanResult.ImagePaths, cats, confidence,
		func(current, total int) {
			fmt.Printf("\rProcessing image %d/%d...", current, total)
		},
	)
	if err != nil {
		return err
	}
	fmt.Println() // newline after progress

	// Move files
	if dryRun {
		fmt.Println("Dry run mode — no files will be moved")
	}
	moves, err := mover.MoveFiles(dir, results, dryRun)
	if err != nil {
		return err
	}

	// Print report
	report.Print(os.Stdout, results, moves, scanResult.SkippedCount, dryRun)

	return nil
}
