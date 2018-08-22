package main

import (
	"unsafe"
	"syscall"
)

/*
#cgo CFLAGS: -D_GNU_SOURCE=1
#cgo LDFLAGS: -lcrypt

#include <stdlib.h>
#include <shadow.h>
#include <string.h>
#include <unistd.h>
#include <crypt.h>
#include <stdbool.h>

static bool login(const char *username, const char *password)
{
	struct spwd spw;
	struct spwd *result;
	struct crypt_data cdata;
	char buf[1024], *sp;
	int s;

	if (!username || *username == 0)
		return false;

	s = getspnam_r(username, &spw, buf, sizeof(buf), &result);
	if (s || !result)
		return false;

	cdata.initialized = 0;
	sp = crypt_r(password, spw.sp_pwdp, &cdata);
	if (!sp)
		return false;

	return !strcmp(sp, spw.sp_pwdp);
}
*/
import "C"

func checkUser() bool {
    return syscall.Getuid() == 0
}

func login(username, password string) bool {
	c_username := C.CString(username)
	c_password := C.CString(password)

	ok := C.login(c_username, c_password);

	C.free(unsafe.Pointer(c_username))
	C.free(unsafe.Pointer(c_password))

	return bool(ok)
}
