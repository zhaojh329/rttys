package pwauth

import (
	"os/user"
	"syscall"
	"unsafe"
)

const (
	LOGON32_LOGON_INTERACTIVE = 2
	LOGON32_PROVIDER_DEFAULT  = 0
)

var errERROR_ACCOUNT_RESTRICTION error = syscall.Errno(1327)

var (
	advapi32       = syscall.NewLazyDLL("advapi32.dll")
	procLogonUserW = advapi32.NewProc("LogonUserW")
)

func LogonUserW(username, domain, password *uint16, logonType, logonProvider uint32) (token syscall.Handle, err error) {
	r1, _, e1 := procLogonUserW.Call(
		uintptr(unsafe.Pointer(username)),
		uintptr(unsafe.Pointer(domain)),
		uintptr(unsafe.Pointer(password)),
		uintptr(logonType),
		uintptr(logonProvider),
		uintptr(unsafe.Pointer(&token)))
	if int(r1) == 0 {
		return syscall.InvalidHandle, e1
	}
	return token, nil
}

func auth(username, password string) error {
	if _, err := user.Lookup(username); err != nil {
		return err
	}

	pUsername, _ := syscall.UTF16PtrFromString(username)
	pDomain, _ := syscall.UTF16PtrFromString(".")
	pPassword, _ := syscall.UTF16PtrFromString(password)

	_, err := LogonUserW(pUsername, pDomain, pPassword, LOGON32_LOGON_INTERACTIVE, LOGON32_PROVIDER_DEFAULT)

	if err == errERROR_ACCOUNT_RESTRICTION {
		return nil
	}

	return err
}
