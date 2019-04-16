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

	usernamePtr, _ := syscall.UTF16PtrFromString(username)
	domainPtr, _ := syscall.UTF16PtrFromString(".")
	passwordPtr, _ := syscall.UTF16PtrFromString(password)

	proc := mod.NewProc("LogonUserW")
	ret, _, err := proc.Call(
		uintptr(unsafe.Pointer(usernamePtr)),
		uintptr(unsafe.Pointer(domainPtr)),
		uintptr(unsafe.Pointer(passwordPtr)),
		uintptr(LOGON32_LOGON_INTERACTIVE),
		uintptr(LOGON32_PROVIDER_DEFAULT),
		uintptr(unsafe.Pointer(&token)))

	return ret == 1 || err == syscall.Errno(1327)
}
