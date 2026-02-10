# imgsort

Sort images into category folders using a local CLIP AI model.

`imgsort` uses OpenAI's CLIP model (running locally via ONNX Runtime) to automatically categorize images and sort them into named subfolders. No cloud APIs, no internet required after initial setup.

## Usage

```bash
imgsort <directory> [--dry-run] [--categories <categories>] [--confidence <threshold>]
```

### Examples

```bash
# Sort images using built-in categories
imgsort ~/Photos

# Preview what would happen without moving files
imgsort ~/Photos --dry-run

# Sort into specific categories only
imgsort ~/Photos --categories "landscape,portrait,food,animals"

# Adjust confidence threshold (default: 0.15)
imgsort ~/Photos --confidence 0.3
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--dry-run` | `false` | Show categorization results without moving files |
| `--categories` | built-in defaults | Comma-separated list of categories |
| `--confidence` | `0.15` | Minimum confidence threshold (0.0-1.0) |

## How It Works

1. Scans the target directory for image files (JPEG, PNG, GIF, BMP, WebP, TIFF)
2. Downloads the CLIP ViT-B/32 model on first run (~600MB, stored in `~/.imgsort/models/`)
3. For each image, computes similarity against all candidate categories using zero-shot classification
4. Moves images into category-named subfolders (or prints a preview with `--dry-run`)

## Custom Categories

By default, imgsort uses a built-in list of ~90 common photo categories. You can customize this:

- **CLI flag:** `--categories "cat1,cat2,cat3"` — uses only these categories
- **Config file:** Create `~/.imgsort/categories.txt` with one category per line

## Prerequisites

- **Go 1.25+**
- **C compiler** (for CGo / ONNX Runtime bindings)
- **ONNX Runtime** shared library

### Install ONNX Runtime

**macOS:**
```bash
brew install onnxruntime
```

**Linux:**
```bash
# Download from https://github.com/microsoft/onnxruntime/releases
# Extract and copy libonnxruntime.so to /usr/lib/
```

## Build

```bash
make setup    # Install ONNX Runtime (macOS only)
make build    # Build the binary
make install  # Install to $GOPATH/bin
make test     # Run tests
```

## Supported Image Formats

JPEG, PNG, GIF, BMP, WebP, TIFF

## License

See [LICENSE](LICENSE) file.
