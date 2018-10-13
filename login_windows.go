package main

import (
	"syscall"
	"unsafe"
)

func checkUser() bool {
	return true
}

func login(username, password string) bool {
	mod := syscall.NewLazyDLL("Advapi32.dll")
	LOGON32_LOGON_INTERACTIVE := 2
	LOGON32_PROVIDER_DEFAULT := 0
	var token syscall.Handle

	proc := mod.NewProc("LogonUserW")
	ret, _, err := proc.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(username))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("."))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(password))),
		uintptr(LOGON32_LOGON_INTERACTIVE),
		uintptr(LOGON32_PROVIDER_DEFAULT),
		uintptr(unsafe.Pointer(&token)))

	return ret == 1 || err == syscall.Errno(1327)
}
