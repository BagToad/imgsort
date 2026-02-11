//go:build embed_onnx && windows && amd64

package onnxlib

import _ "embed"

//go:embed onnxruntime.dll
var libraryData []byte

const libraryName = "onnxruntime.dll"
