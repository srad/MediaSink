package tensorflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// getModelPath finds the correct path to a TensorFlow model
// It handles different working directories by searching for the assets/models directory
func GetModelPath(modelName string) (string, error) {
	// First try: relative path from current working directory
	relativePath := filepath.Join("assets", "models", modelName)
	if _, err := os.Stat(relativePath); err == nil {
		return relativePath, nil
	}

	// Second try: from project root (one level up from detectors)
	// This handles test execution from ./detectors
	upPath := filepath.Join("..", "..", "assets", "models", modelName)
	if _, err := os.Stat(upPath); err == nil {
		return upPath, nil
	}

	// Third try: search from current directory upward for assets/models
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	searchDir := cwd
	for {
		testPath := filepath.Join(searchDir, "assets", "models", modelName)
		if _, err := os.Stat(testPath); err == nil {
			return testPath, nil
		}

		// Move up one directory
		parent := filepath.Dir(searchDir)
		if parent == searchDir {
			// Reached filesystem root
			break
		}
		searchDir = parent
	}

	// If all attempts fail, return error with attempted paths
	return "", fmt.Errorf("model not found: tried %s and searched upward from %s",
		relativePath, cwd)
}

// ParseTensorName splits a tensor name like "operation:0" into operation name and output index
func ParseTensorName(tensorName string) (string, int, error) {
	parts := strings.Split(tensorName, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid tensor name format: %s", tensorName)
	}
	opName := parts[0]
	outputIdx, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, fmt.Errorf("invalid output index in tensor name %s: %w", tensorName, err)
	}
	return opName, outputIdx, nil
}
