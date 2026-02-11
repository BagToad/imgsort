//go:build !embed_onnx

package onnxlib

// No embedded library — fall back to system-installed ONNX Runtime.
var libraryData []byte

const libraryName = ""
