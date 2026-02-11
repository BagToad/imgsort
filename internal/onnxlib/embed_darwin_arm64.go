//go:build embed_onnx && darwin && arm64

package onnxlib

import _ "embed"

//go:embed libonnxruntime.dylib
var libraryData []byte

const libraryName = "libonnxruntime.dylib"
