package onnx

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetModelPath finds the path to an ONNX model file (<modelName>.onnx).
// It searches for assets/models/<modelName>.onnx starting from the working
// directory and walking upward to the filesystem root.
func GetModelPath(modelName string) (string, error) {
	filename := modelName + ".onnx"

	// First try: relative path from current working directory
	relativePath := filepath.Join("assets", "models", filename)
	if _, err := os.Stat(relativePath); err == nil {
		return relativePath, nil
	}

	// Second try: one level up (handles tests run from a sub-package)
	upPath := filepath.Join("..", "..", "assets", "models", filename)
	if _, err := os.Stat(upPath); err == nil {
		return upPath, nil
	}

	// Third try: walk upward from the working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	searchDir := cwd
	for {
		testPath := filepath.Join(searchDir, "assets", "models", filename)
		if _, err := os.Stat(testPath); err == nil {
			return testPath, nil
		}

		parent := filepath.Dir(searchDir)
		if parent == searchDir {
			break
		}
		searchDir = parent
	}

	return "", fmt.Errorf("model not found: tried %s and searched upward from %s",
		relativePath, cwd)
}
