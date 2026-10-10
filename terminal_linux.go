//go:build linux

package main

import (
	"os"
	"syscall"
	"unsafe"
)

func makeTerminalRaw(file *os.File) (func() error, bool, error) {
	fd := file.Fd()
	var original syscall.Termios
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fd, uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&original)), 0, 0, 0)
	if errno == syscall.ENOTTY {
		return nil, false, nil
	}
	if errno != 0 {
		return nil, false, errno
	}
	raw := original
	raw.Iflag &^= syscall.ICRNL | syscall.IXON
	raw.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0
	_, _, errno = syscall.Syscall6(syscall.SYS_IOCTL, fd, uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(&raw)), 0, 0, 0)
	if errno != 0 {
		return nil, false, errno
	}
	restore := func() error {
		_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fd, uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(&original)), 0, 0, 0)
		if errno != 0 {
			return errno
		}
		return nil
	}
	return restore, true, nil
}
