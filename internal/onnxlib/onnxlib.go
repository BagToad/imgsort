// Package onnxlib extracts the embedded ONNX Runtime shared library to a
// temporary directory and returns its path. This allows the binary to be
// fully self-contained with no external runtime dependencies.
package onnxlib

import (
	"fmt"
	"os"
	"path/filepath"
)

// Extract writes the embedded ONNX Runtime shared library to a temporary
// directory and returns its full path.
func Extract() (string, error) {
	if len(libraryData) == 0 {
		return "", fmt.Errorf("no embedded ONNX Runtime library for this platform")
	}

	dir, err := os.MkdirTemp("", "imgsort-onnxrt-*")
	if err != nil {
		return "", fmt.Errorf("cannot create temp dir: %w", err)
	}

	libPath := filepath.Join(dir, libraryName)
	if err := os.WriteFile(libPath, libraryData, 0755); err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("cannot write library: %w", err)
	}

	return libPath, nil
}
