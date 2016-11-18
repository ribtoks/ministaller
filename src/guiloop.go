// +build !windows

package main

import (
  "log"
)

var (
  finished = make(chan bool)
)

func onPercentUpdate(percent int) {
  log.Printf("Completed %v%%...", percent);
}

func onSystemMessage(message string) {
  log.Println("System message: " + message)
}

func onFinished() {
  finished <- true
}

func guiloop() {
  <- finished
}
