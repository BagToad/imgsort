// Package categorizer provides zero-shot image classification using CLIP embeddings.
package categorizer

import (
	"fmt"
	"log"

	"github.com/bagtoad/imgsort/internal/model"
)

// Result holds the categorization result for a single image.
type Result struct {
	Path       string
	Category   string
	Confidence float32
	Skipped    bool
}

// Categorize classifies a list of images against the given categories using
// the provided CLIP session. Images below the confidence threshold or where the
// baseline "uncategorized" prompt wins are skipped.
func Categorize(
	clip *model.CLIPSession,
	imagePaths []string,
	categories []string,
	threshold float64,
	progressFn func(current, total int),
) ([]Result, error) {
	if len(categories) == 0 {
		return nil, fmt.Errorf("no categories provided")
	}

	results := make([]Result, 0, len(imagePaths))

	for i, imgPath := range imagePaths {
		if progressFn != nil {
			progressFn(i+1, len(imagePaths))
		}

		scores, err := clip.Classify(imgPath, categories)
		if err != nil {
			log.Printf("Warning: skipping %s: %v", imgPath, err)
			results = append(results, Result{Path: imgPath, Skipped: true})
			continue
		}

		// Find the best real category (excluding the baseline)
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

		// Skip if the baseline "uncategorized" prompt scored higher than the best real category
		baselineScore := scores[model.BaselineCategory]
		if baselineScore >= bestScore {
			log.Printf("Warning: skipping %s (no category matched better than baseline; best was %q at %.1f%%)",
				imgPath, bestCat, bestScore*100)
			results = append(results, Result{Path: imgPath, Skipped: true})
			continue
		}

		if float64(bestScore) < threshold {
			log.Printf("Warning: skipping %s (best match %q at %.1f%% confidence, below %.1f%% threshold)",
				imgPath, bestCat, bestScore*100, threshold*100)
			results = append(results, Result{Path: imgPath, Skipped: true})
			continue
		}

		results = append(results, Result{
			Path:       imgPath,
			Category:   bestCat,
			Confidence: bestScore,
		})
	}

	return results, nil
}

// GroupByCategory groups categorization results by category name.
func GroupByCategory(results []Result) map[string][]Result {
	groups := make(map[string][]Result)
	for _, r := range results {
		if !r.Skipped {
			groups[r.Category] = append(groups[r.Category], r)
		}
	}
	return groups
}
