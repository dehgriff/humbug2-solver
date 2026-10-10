//go:build !darwin && !linux

package main

import "os"

func makeTerminalRaw(file *os.File) (func() error, bool, error) {
	return nil, false, nil
}
