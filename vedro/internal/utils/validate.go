package utils

import (
	"path/filepath"
	"strings"
)

func ValidatePath(root, bucket, key string) bool {
	cleanPath := filepath.Clean(filepath.Join(root, bucket, key))
	return strings.HasPrefix(cleanPath, filepath.Clean(root))
}

func GetFullPath(root, bucket, key string) string {
	return filepath.Clean(filepath.Join(root, bucket, key))
}