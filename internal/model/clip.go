package model

import (
	"fmt"
	"math"
	"runtime"

	"github.com/bagtoad/imgsort/internal/onnxlib"
	ort "github.com/yalue/onnxruntime_go"
)

// CLIPSession holds a loaded CLIP model ready for inference.
type CLIPSession struct {
	session   *ort.DynamicAdvancedSession
	tokenizer *Tokenizer
}

// NewCLIPSession creates a new CLIP inference session.
// If explicitPath is empty, it tries the embedded library first, then platform defaults.
func NewCLIPSession(explicitPath string) (*CLIPSession, error) {
	var onnxrtLibPath string
	if explicitPath != "" {
		onnxrtLibPath = explicitPath
	} else if extractedPath, err := onnxlib.Extract(); err == nil {
		onnxrtLibPath = extractedPath
	} else {
		onnxrtLibPath = defaultONNXRuntimePath()
	}
	ort.SetSharedLibraryPath(onnxrtLibPath)
	if err := ort.InitializeEnvironment(); err != nil {
		return nil, fmt.Errorf("cannot initialize ONNX Runtime: %w", err)
	}

	modelPath, err := FilePath("model.onnx")
	if err != nil {
		return nil, err
	}

	session, err := ort.NewDynamicAdvancedSession(
		modelPath,
		[]string{"input_ids", "pixel_values", "attention_mask"},
		[]string{"logits_per_image", "logits_per_text"},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot create ONNX session: %w", err)
	}

	tokenizer, err := TokenizerFromModelsDir()
	if err != nil {
		session.Destroy()
		return nil, fmt.Errorf("cannot load tokenizer: %w", err)
	}

	return &CLIPSession{
		session:   session,
		tokenizer: tokenizer,
	}, nil
}

// BaselineCategory is the internal label for the baseline "catch-all" prompt
// used to prevent false-positive classification.
const BaselineCategory = "uncategorized"

// baselinePrompt is the generic prompt that competes with real categories.
// If an image is more similar to this than any specific category, it's skipped.
const baselinePrompt = "a photo"

// Classify runs zero-shot classification on an image against the given categories.
// A baseline "uncategorized" prompt is injected to prevent false positives
// (especially with few categories). Returns a map of category names to their
// similarity scores (after softmax), including the baseline.
func (c *CLIPSession) Classify(imagePath string, categories []string) (map[string]float32, error) {
	// Preprocess image
	pixelValues, err := PreprocessImage(imagePath)
	if err != nil {
		return nil, fmt.Errorf("cannot preprocess image: %w", err)
	}

	// Build prompt list: baseline + real categories
	allLabels := append([]string{BaselineCategory}, categories...)
	numLabels := int64(len(allLabels))

	// Tokenize: baseline gets the generic prompt, others get "a photo of {cat}"
	tokenIDs := make([]int64, 0, len(allLabels)*contextLen)
	tokenIDs = append(tokenIDs, c.tokenizer.Encode(baselinePrompt)...)
	for _, cat := range categories {
		prompt := fmt.Sprintf("a photo of %s", cat)
		tokenIDs = append(tokenIDs, c.tokenizer.Encode(prompt)...)
	}

	// Create attention mask (1 for non-padding, 0 for padding)
	attentionMask := make([]int64, len(tokenIDs))
	for i, id := range tokenIDs {
		if id != 0 {
			attentionMask[i] = 1
		}
	}

	// Create input tensors
	inputIDsTensor, err := ort.NewTensor(ort.NewShape(numLabels, int64(contextLen)), tokenIDs)
	if err != nil {
		return nil, fmt.Errorf("cannot create input_ids tensor: %w", err)
	}
	defer inputIDsTensor.Destroy()

	pixelTensor, err := ort.NewTensor(ort.NewShape(1, 3, int64(clipImageSize), int64(clipImageSize)), pixelValues)
	if err != nil {
		return nil, fmt.Errorf("cannot create pixel_values tensor: %w", err)
	}
	defer pixelTensor.Destroy()

	attentionTensor, err := ort.NewTensor(ort.NewShape(numLabels, int64(contextLen)), attentionMask)
	if err != nil {
		return nil, fmt.Errorf("cannot create attention_mask tensor: %w", err)
	}
	defer attentionTensor.Destroy()

	// Create output tensors
	logitsPerImage, err := ort.NewEmptyTensor[float32](ort.NewShape(1, numLabels))
	if err != nil {
		return nil, fmt.Errorf("cannot create output tensor: %w", err)
	}
	defer logitsPerImage.Destroy()

	logitsPerText, err := ort.NewEmptyTensor[float32](ort.NewShape(numLabels, 1))
	if err != nil {
		return nil, fmt.Errorf("cannot create output tensor: %w", err)
	}
	defer logitsPerText.Destroy()

	// Run inference
	inputs := []ort.Value{inputIDsTensor, pixelTensor, attentionTensor}
	outputs := []ort.Value{logitsPerImage, logitsPerText}
	if err := c.session.Run(inputs, outputs); err != nil {
		return nil, fmt.Errorf("inference failed: %w", err)
	}

	// Extract logits and apply softmax over all labels (including baseline)
	logits := logitsPerImage.GetData()
	probs := softmax(logits)

	// Return all scores including the baseline
	result := make(map[string]float32, len(allLabels))
	for i, label := range allLabels {
		result[label] = probs[i]
	}
	return result, nil
}

// Destroy releases resources held by the CLIP session.
func (c *CLIPSession) Destroy() {
	if c.session != nil {
		c.session.Destroy()
	}
	ort.DestroyEnvironment()
}

func softmax(logits []float32) []float32 {
	max := logits[0]
	for _, v := range logits[1:] {
		if v > max {
			max = v
		}
	}

	sum := float32(0)
	result := make([]float32, len(logits))
	for i, v := range logits {
		result[i] = float32(math.Exp(float64(v - max)))
		sum += result[i]
	}
	for i := range result {
		result[i] /= sum
	}
	return result
}

func defaultONNXRuntimePath() string {
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "/opt/homebrew/lib/libonnxruntime.dylib"
		}
		return "/usr/local/lib/libonnxruntime.dylib"
	case "linux":
		return "/usr/lib/libonnxruntime.so"
	case "windows":
		return "onnxruntime.dll"
	default:
		return "libonnxruntime.so"
	}
}
