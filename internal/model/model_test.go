package model

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

func TestPreprocessImage(t *testing.T) {
	// Create a simple 100x100 red test image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	f, err := os.CreateTemp("", "test_*.png")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()

	tensor, err := PreprocessImage(f.Name())
	if err != nil {
		t.Fatalf("PreprocessImage failed: %v", err)
	}

	expectedLen := 3 * clipImageSize * clipImageSize
	if len(tensor) != expectedLen {
		t.Errorf("expected tensor length %d, got %d", expectedLen, len(tensor))
	}

	// R channel should be positive (red pixel normalized)
	// (1.0 - 0.48145466) / 0.26862954 ≈ 1.93
	rVal := tensor[0] // first pixel of R channel
	if rVal < 1.5 || rVal > 2.5 {
		t.Errorf("unexpected R channel value: %f (expected ~1.93)", rVal)
	}

	// G channel should be negative (0.0 normalized)
	// (0.0 - 0.4578275) / 0.26130258 ≈ -1.75
	gVal := tensor[clipImageSize * clipImageSize] // first pixel of G channel
	if gVal > -1.0 || gVal < -2.5 {
		t.Errorf("unexpected G channel value: %f (expected ~-1.75)", gVal)
	}
}

func TestPreprocessImageNonSquare(t *testing.T) {
	// Create a 200x100 image (landscape orientation)
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			img.Set(x, y, color.RGBA{R: 128, G: 128, B: 128, A: 255})
		}
	}

	f, err := os.CreateTemp("", "test_landscape_*.png")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()

	tensor, err := PreprocessImage(f.Name())
	if err != nil {
		t.Fatalf("PreprocessImage failed: %v", err)
	}

	expectedLen := 3 * clipImageSize * clipImageSize
	if len(tensor) != expectedLen {
		t.Errorf("expected tensor length %d, got %d", expectedLen, len(tensor))
	}
}

func TestCenterCrop(t *testing.T) {
	// Wide image
	wide := image.NewRGBA(image.Rect(0, 0, 300, 100))
	cropped := centerCrop(wide)
	bounds := cropped.Bounds()
	if bounds.Dx() != bounds.Dy() {
		t.Errorf("center crop should produce square: got %dx%d", bounds.Dx(), bounds.Dy())
	}
	if bounds.Dx() != 100 {
		t.Errorf("expected 100x100, got %dx%d", bounds.Dx(), bounds.Dy())
	}

	// Tall image
	tall := image.NewRGBA(image.Rect(0, 0, 100, 300))
	cropped = centerCrop(tall)
	bounds = cropped.Bounds()
	if bounds.Dx() != bounds.Dy() {
		t.Errorf("center crop should produce square: got %dx%d", bounds.Dx(), bounds.Dy())
	}
	if bounds.Dx() != 100 {
		t.Errorf("expected 100x100, got %dx%d", bounds.Dx(), bounds.Dy())
	}

	// Already square
	square := image.NewRGBA(image.Rect(0, 0, 100, 100))
	cropped = centerCrop(square)
	if cropped != square {
		t.Error("square image should be returned unchanged")
	}
}

func TestSoftmax(t *testing.T) {
	logits := []float32{1.0, 2.0, 3.0}
	probs := softmax(logits)

	// Sum should be ~1.0
	sum := float32(0)
	for _, p := range probs {
		sum += p
	}
	if sum < 0.99 || sum > 1.01 {
		t.Errorf("softmax sum should be 1.0, got %f", sum)
	}

	// probs should be in ascending order
	if probs[0] >= probs[1] || probs[1] >= probs[2] {
		t.Errorf("softmax probabilities should be ascending: %v", probs)
	}
}
