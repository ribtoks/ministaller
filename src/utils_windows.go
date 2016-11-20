package main

import (
	"syscall"
	"unicode/utf16"
	"unsafe"
  "os"
  "os/exec"
  "path/filepath"
)

var (
	kernel = syscall.MustLoadDLL("kernel32.dll")
	getModuleFileNameProc = kernel.MustFindProc("GetModuleFileNameW")
)

func getModuleFileName() (string, error) {
	var n uint32
	b := make([]uint16, syscall.MAX_PATH)
	size := uint32(len(b))

	ret, _, err := getModuleFileNameProc.Call(0, uintptr(unsafe.Pointer(&b[0])), uintptr(size))
	n = uint32(ret)
	if n == 0 {
		return "", err
	}

	return string(utf16.Decode(b[0:n])), nil
}

func executablePath() string {
  exepath, err := getModuleFileName()
  if err != nil {
    exepath, _ = exec.LookPath(os.Args[0])
  }

  return filepath.ToSlash(execpath)
}
