package utils

import (
	"os"
	"path/filepath"
)

func GetCurrentPath() string {
	if ex, err := os.Executable(); err == nil {
		return filepath.Dir(ex) + "/"
	}
	return ""
}
