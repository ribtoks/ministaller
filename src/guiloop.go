// +build !windows

package main

import (
  "log"
)

var (
  finished = make(chan bool)
)

func NewUIProgressHandler() ProgressHandler {
  return &UIProgressHandler{}
}

type UIProgressHandler struct {
  LogProgressHandler
}

func (ph *UIProgressHandler) HandleFinish() {
  log.Printf("Finished")
  finished <- true
}

func guiinit() {
  // do nothing
}

func guiloop() {
  <- finished
}
