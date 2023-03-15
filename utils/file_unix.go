//go:build unix || linux

package utils

import (
	"os"
	"syscall"
)

func CreateLockFile(fileName string) (*os.File, error) {
	file, err := os.OpenFile(fileName, os.O_WRONLY, 0666)
	if err != nil {
		return nil, err
	}
	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		return nil, err
	}
	return file, nil
}
