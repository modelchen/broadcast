//go:build windows

package utils

import (
	"os"
)

func CreateLockFile(fileName string) (*os.File, error) {
	if _, err := os.Stat(fileName); err == nil {
		err = os.Remove(fileName)
		if err != nil {
			return nil, err
		}
	}
	return os.OpenFile(fileName, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666)
}
