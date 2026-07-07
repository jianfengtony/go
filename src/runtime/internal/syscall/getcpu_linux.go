// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux && (amd64 || arm64)

package syscall

import "unsafe"

// Getcpu returns the OS-assigned CPU number of the calling thread via
// the getcpu(2) system call. On error, cpu is 0 and errno is nonzero.
func Getcpu() (cpu uint32, errno uintptr) {
	var c uint32
	_, _, e := Syscall6(SYS_GETCPU, uintptr(unsafe.Pointer(&c)), 0, 0, 0, 0, 0)
	return c, e
}
