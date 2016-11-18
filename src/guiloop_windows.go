package main

import (
  "github.com/Ribtoks/gform"
)

var (
  pb  *gform.ProgressBar
  lb *gform.Label
)

func onPercentUpdate(percent int) {
  pb.SetValue(uint32(percent))

  if percent == 100 {
    gform.Exit()
  }
}

func onSystemMessage(message string) {
  lb.SetCaption(message)
}

func onFinished() {
  gform.Exit()
}

func guiloop() {
  gform.Init()

  mw := gform.NewForm(nil)
  mw.SetSize(360, 100)
  mw.SetCaption("Progress bar")
  mw.EnableMaxButton(false)
  mw.EnableSizable(false)
  mw.OnClose().Bind(func (arg *gform.EventArg) {
    
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

  gform.RunMainLoop()
}
