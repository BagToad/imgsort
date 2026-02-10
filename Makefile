.PHONY: build install setup download-models test clean

BINARY_NAME=imgsort
BUILD_DIR=./cmd/imgsort

# Detect platform for ONNX Runtime
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

build:
	go build -o bin/$(BINARY_NAME) $(BUILD_DIR)

install:
	go install $(BUILD_DIR)

test:
	go test ./...

test-integration:
	go test -tags integration ./test/ -v -count=1

clean:
	rm -f $(BINARY_NAME)

setup:
ifeq ($(UNAME_S),Darwin)
	@echo "Installing ONNX Runtime via Homebrew..."
	brew install onnxruntime
else ifeq ($(UNAME_S),Linux)
	@echo "Please install ONNX Runtime:"
	@echo "  Download from https://github.com/microsoft/onnxruntime/releases"
	@echo "  Extract and copy libonnxruntime.so to /usr/lib/"
else
	@echo "Please install ONNX Runtime for your platform:"
	@echo "  https://github.com/microsoft/onnxruntime/releases"
endif

download-models:
	@echo "Models will be downloaded automatically on first run."
	@echo "To pre-download, run: $(BINARY_NAME) --help && $(BINARY_NAME) /tmp --dry-run"
