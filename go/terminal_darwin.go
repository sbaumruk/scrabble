package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

var origTermios syscall.Termios

func initTerminal() {
	syscall.Syscall(syscall.SYS_IOCTL, 0, syscall.TIOCGETA,
		uintptr(unsafe.Pointer(&origTermios)))
}

func enableRaw() {
	raw := origTermios
	raw.Lflag &^= syscall.ECHO | syscall.ICANON
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0
	syscall.Syscall(syscall.SYS_IOCTL, 0, syscall.TIOCSETA,
		uintptr(unsafe.Pointer(&raw)))
	fmt.Print("\x1b[?25l") // hide cursor
}

func disableRaw() {
	syscall.Syscall(syscall.SYS_IOCTL, 0, syscall.TIOCSETA,
		uintptr(unsafe.Pointer(&origTermios)))
	fmt.Print("\x1b[?25h") // show cursor
}
