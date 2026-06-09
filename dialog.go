//go:build windows

package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

func showFatalDialog(title, message string) {
	user32 := windows.NewLazySystemDLL("user32.dll")
	messageBox := user32.NewProc("MessageBoxW")
	messageBox.Call(
		0,
		uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(message))),
		uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(title))),
		0x10, // MB_ICONERROR
	)
}
