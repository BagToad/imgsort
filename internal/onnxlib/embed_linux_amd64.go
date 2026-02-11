//go:build embed_onnx && linux && amd64

package onnxlib

import _ "embed"

//go:embed libonnxruntime.so
var libraryData []byte

const libraryName = "libonnxruntime.so"
