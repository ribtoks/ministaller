package main

import (
  "github.com/Ribtoks/gform"
  "github.com/Ribtoks/w32"
)

var (
  guifinished chan bool
  mw *gform.Form
  pb *gform.ProgressBar
  lb *gform.Label
)

func NewUIProgressHandler() ProgressHandler {
  return &WinUIProgressHandler{}
}

type WinUIProgressHandler struct {
}

func (ph *WinUIProgressHandler) HandlePercentChange(percent int) {
  pb.SetValue(uint32(percent))
}

func (ph *WinUIProgressHandler) HandleSystemMessage(msg string) {
  lb.SetCaption(msg)
}

func (ph *WinUIProgressHandler) HandleFinish() {
  guifinished <- true
  gform.Exit()
}

func guiinit() {
  gform.Init()

  mw = gform.NewForm(nil)
  mw.SetSize(360, 125)
  mw.SetCaption("ministaller")
  mw.EnableMaxButton(false)
  mw.EnableSizable(false)
  mw.OnClose().Bind(func (arg *gform.EventArg) {
    gform.MsgBox(arg.Sender().Parent(), "Info", "Please wait for the installation to finish", w32.MB_OK | w32.MB_ICONWARNING)
  });

  lb = gform.NewLabel(mw)
  lb.SetPos(21, 10)
  lb.SetSize(300, 25)
  lb.SetCaption("Preparing the install...")

  pb = gform.NewProgressBar(mw)
  pb.SetPos(20, 35)
  pb.SetSize(300, 25)

  mw.Show()
  mw.Center()
}

func guiloop() {
  go gform.RunMainLoop()
  <- guifinished
}

func init() {
  guifinished = make(chan bool)
}
