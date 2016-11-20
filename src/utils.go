package main

import (
  "os"
  "os/exec"
)

func executablePath() string {
  fullpath, _ := exec.LookPath(os.Args[0])
  return fullpath
}
