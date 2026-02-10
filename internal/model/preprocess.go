package model

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"image/color"
	"math"
	"os"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

const clipImageSize = 224

// CLIP normalization constants
var (
	clipMean = [3]float32{0.48145466, 0.4578275, 0.40821073}
	clipStd  = [3]float32{0.26862954, 0.26130258, 0.27577711}
)

// PreprocessImage loads an image file and returns a float32 tensor in
// [1, 3, 224, 224] CHW format, normalized for CLIP.
func PreprocessImage(path string) ([]float32, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open image: %w", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("cannot decode image: %w", err)
	}

	// Center crop to square
	img = centerCrop(img)

	// Resize to 224x224 using bilinear interpolation
	img = resize(img, clipImageSize, clipImageSize)

	// Convert to CHW float32 tensor with normalization
	return imageToTensor(img), nil
}

// centerCrop crops the image to a square from the center.
func centerCrop(img image.Image) image.Image {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	if w == h {
		return img
	}

	var cropRect image.Rectangle
	if w > h {
		offset := (w - h) / 2
		cropRect = image.Rect(bounds.Min.X+offset, bounds.Min.Y, bounds.Min.X+offset+h, bounds.Max.Y)
	} else {
		offset := (h - w) / 2
		cropRect = image.Rect(bounds.Min.X, bounds.Min.Y+offset, bounds.Max.X, bounds.Min.Y+offset+w)
	}

	cropped := image.NewRGBA(image.Rect(0, 0, cropRect.Dx(), cropRect.Dy()))
	for y := 0; y < cropRect.Dy(); y++ {
		for x := 0; x < cropRect.Dx(); x++ {
			cropped.Set(x, y, img.At(cropRect.Min.X+x, cropRect.Min.Y+y))
		}
	}
	return cropped
}

// resize performs bilinear interpolation to resize an image.
func resize(img image.Image, width, height int) image.Image {
	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	dst := image.NewRGBA(image.Rect(0, 0, width, height))

	xRatio := float64(srcW) / float64(width)
	yRatio := float64(srcH) / float64(height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := float64(x)*xRatio + float64(bounds.Min.X)
			srcY := float64(y)*yRatio + float64(bounds.Min.Y)

			x0 := int(math.Floor(srcX))
			y0 := int(math.Floor(srcY))
			x1 := x0 + 1
			y1 := y0 + 1

			if x1 >= bounds.Max.X {
				x1 = bounds.Max.X - 1
			}
			if y1 >= bounds.Max.Y {
				y1 = bounds.Max.Y - 1
			}

			xFrac := srcX - float64(x0)
			yFrac := srcY - float64(y0)

			r00, g00, b00, a00 := img.At(x0, y0).RGBA()
			r10, g10, b10, a10 := img.At(x1, y0).RGBA()
			r01, g01, b01, a01 := img.At(x0, y1).RGBA()
			r11, g11, b11, a11 := img.At(x1, y1).RGBA()

			r := bilinear(float64(r00), float64(r10), float64(r01), float64(r11), xFrac, yFrac)
			g := bilinear(float64(g00), float64(g10), float64(g01), float64(g11), xFrac, yFrac)
			b := bilinear(float64(b00), float64(b10), float64(b01), float64(b11), xFrac, yFrac)
			a := bilinear(float64(a00), float64(a10), float64(a01), float64(a11), xFrac, yFrac)

			dst.Set(x, y, color.RGBA64{
				R: uint16(r),
				G: uint16(g),
				B: uint16(b),
				A: uint16(a),
			})
		}
	}
	return dst
}

func bilinear(c00, c10, c01, c11, xFrac, yFrac float64) float64 {
	return c00*(1-xFrac)*(1-yFrac) + c10*xFrac*(1-yFrac) +
		c01*(1-xFrac)*yFrac + c11*xFrac*yFrac
}

// imageToTensor converts an image to a [1, 3, 224, 224] CHW float32 tensor,
// normalized with CLIP mean and std.
func imageToTensor(img image.Image) []float32 {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	tensor := make([]float32, 3*h*w)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()

			// Convert from uint16 [0, 65535] to float32 [0, 1], then normalize
			rf := float32(r) / 65535.0
			gf := float32(g) / 65535.0
			bf := float32(b) / 65535.0

			idx := y*w + x
			tensor[0*h*w+idx] = (rf - clipMean[0]) / clipStd[0] // R channel
			tensor[1*h*w+idx] = (gf - clipMean[1]) / clipStd[1] // G channel
			tensor[2*h*w+idx] = (bf - clipMean[2]) / clipStd[2] // B channel
		}
	}

	return tensor
}
