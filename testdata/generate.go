// This program generates test images for integration testing.
// Each image uses colors and patterns that CLIP can distinguish.
//
//go:build ignore

package main

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

func main() {
	dir := "testdata"
	os.MkdirAll(dir, 0755)

	// A blue sky with green ground — landscape-like
	generateSkyGround(filepath.Join(dir, "landscape.jpg"))

	// A warm orange/red image — sunset-like
	generateSunset(filepath.Join(dir, "sunset.png"))

	// A bright red image — like a red flower or object
	generateSolidColor(filepath.Join(dir, "red_object.jpg"), color.RGBA{220, 30, 30, 255})

	// A dark image — night-like
	generateSolidColor(filepath.Join(dir, "dark_scene.png"), color.RGBA{15, 15, 30, 255})

	// A green nature-like gradient
	generateNature(filepath.Join(dir, "nature.jpg"))

	// A white/gray document-like image with "text" lines
	generateDocument(filepath.Join(dir, "document.png"))

	// A non-image file for skip testing
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not an image"), 0644)
}

func generateSkyGround(path string) {
	img := image.NewRGBA(image.Rect(0, 0, 224, 224))
	for y := 0; y < 224; y++ {
		for x := 0; x < 224; x++ {
			if y < 112 {
				// Sky blue gradient
				b := uint8(180 + y/3)
				img.Set(x, y, color.RGBA{100, 150, b, 255})
			} else {
				// Green ground
				g := uint8(100 + (224-y)/3)
				img.Set(x, y, color.RGBA{50, g, 30, 255})
			}
		}
	}
	saveJPEG(path, img)
}

func generateSunset(path string) {
	img := image.NewRGBA(image.Rect(0, 0, 224, 224))
	for y := 0; y < 224; y++ {
		for x := 0; x < 224; x++ {
			r := uint8(255 - y/3)
			g := uint8(100 + int(80*math.Sin(float64(y)/30)))
			b := uint8(50 + y/4)
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}
	savePNG(path, img)
}

func generateSolidColor(path string, c color.RGBA) {
	img := image.NewRGBA(image.Rect(0, 0, 224, 224))
	for y := 0; y < 224; y++ {
		for x := 0; x < 224; x++ {
			img.Set(x, y, c)
		}
	}
	if filepath.Ext(path) == ".png" {
		savePNG(path, img)
	} else {
		saveJPEG(path, img)
	}
}

func generateNature(path string) {
	img := image.NewRGBA(image.Rect(0, 0, 224, 224))
	for y := 0; y < 224; y++ {
		for x := 0; x < 224; x++ {
			g := uint8(80 + int(80*math.Sin(float64(x)/20)*math.Cos(float64(y)/25)))
			r := uint8(40 + int(30*math.Sin(float64(y)/30)))
			img.Set(x, y, color.RGBA{r, g, 20, 255})
		}
	}
	saveJPEG(path, img)
}

func generateDocument(path string) {
	img := image.NewRGBA(image.Rect(0, 0, 224, 224))
	// White background
	for y := 0; y < 224; y++ {
		for x := 0; x < 224; x++ {
			img.Set(x, y, color.RGBA{245, 245, 245, 255})
		}
	}
	// Dark horizontal lines simulating text
	for line := 0; line < 12; line++ {
		y := 20 + line*16
		lineWidth := 140 + (line%3)*20
		for x := 20; x < 20+lineWidth && x < 210; x++ {
			for dy := 0; dy < 3; dy++ {
				if y+dy < 224 {
					img.Set(x, y+dy, color.RGBA{40, 40, 40, 255})
				}
			}
		}
	}
	savePNG(path, img)
}

func saveJPEG(path string, img image.Image) {
	f, _ := os.Create(path)
	defer f.Close()
	jpeg.Encode(f, img, &jpeg.Options{Quality: 90})
}

func savePNG(path string, img image.Image) {
	f, _ := os.Create(path)
	defer f.Close()
	png.Encode(f, img)
}
