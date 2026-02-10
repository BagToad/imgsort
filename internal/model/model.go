// Package model handles CLIP ONNX model downloading, loading, and inference.
package model

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const hfBaseURL = "https://huggingface.co/Xenova/clip-vit-base-patch32/resolve/main"

// ModelFile describes a file to download.
type ModelFile struct {
	Name   string
	URL    string
	SHA256 string // expected hash (empty = skip verification)
}

// RequiredFiles defines all files needed for CLIP inference.
var RequiredFiles = []ModelFile{
	{
		Name: "model.onnx",
		URL:  hfBaseURL + "/onnx/model.onnx",
	},
	{
		Name: "vocab.json",
		URL:  hfBaseURL + "/vocab.json",
	},
	{
		Name: "merges.txt",
		URL:  hfBaseURL + "/merges.txt",
	},
}

// ModelsDir returns the path to the model storage directory (~/.imgsort/models/).
func ModelsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".imgsort", "models"), nil
}

// EnsureModels checks that all required files exist, downloading any that are missing.
func EnsureModels(progressFn func(filename string, downloaded, total int64)) error {
	dir, err := ModelsDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("cannot create models directory: %w", err)
	}

	for _, m := range RequiredFiles {
		path := filepath.Join(dir, m.Name)
		if _, err := os.Stat(path); err == nil {
			continue // already downloaded
		}

		if err := downloadFile(path, m.URL, m.SHA256, func(downloaded, total int64) {
			if progressFn != nil {
				progressFn(m.Name, downloaded, total)
			}
		}); err != nil {
			os.Remove(path) // clean up partial download
			return fmt.Errorf("failed to download %s: %w", m.Name, err)
		}
	}
	return nil
}

// FilePath returns the full path to a named file in the models directory.
func FilePath(name string) (string, error) {
	dir, err := ModelsDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("file not found: %s (run imgsort to download)", name)
	}
	return path, nil
}

func downloadFile(destPath, url, expectedHash string, progressFn func(downloaded, total int64)) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	tmpPath := destPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("cannot create file: %w", err)
	}
	defer func() {
		f.Close()
		os.Remove(tmpPath) // clean up if not renamed
	}()

	hasher := sha256.New()
	writer := io.MultiWriter(f, hasher)

	var downloaded int64
	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := writer.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("write error: %w", writeErr)
			}
			downloaded += int64(n)
			if progressFn != nil {
				progressFn(downloaded, resp.ContentLength)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read error: %w", readErr)
		}
	}

	f.Close()

	// Verify hash if provided
	if expectedHash != "" {
		actualHash := hex.EncodeToString(hasher.Sum(nil))
		if actualHash != expectedHash {
			return fmt.Errorf("SHA256 mismatch: expected %s, got %s", expectedHash, actualHash)
		}
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("cannot finalize download: %w", err)
	}
	return nil
}
