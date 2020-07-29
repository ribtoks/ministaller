This repository is a fork of (AllenDang/gform) with fixes for x64 and some other improvements

# gform is an easy to use Windows GUI toolkit for Go
It provides two approaches to create UI.

## 1. Pure code.

    gform.Init()

    mainWindow := gform.NewForm(nil)
    mainWindow.SetPos(300, 100)
    mainWindow.SetSize(500, 300)
    mainWindow.SetCaption("Controls Demo")
    
    btn := gform.NewPushButton(mainWindow)
    btn.SetPos(10, 10)
    btn.OnLBUp().Bind(func(args *gform.EventArg) { 
        fmt.Println("Clicked!")
    })
    
    mainWindow.Show()
    
    gform.RunMainLoop()

## 2. Create dialog in resource file and attach to it.

    gform.Init()
    
    dialog := gform.NewDialogFromResId(nil, 101) //101 is the resource Id.
    dialog.Center()
    dialog.Show()
    
    edt = gform.AttachEdit(dialog, 1000)
    edt.SetCaption("Hello")
    
    btn := gform.AttachPushButton(dialog, 2)
    btn.OnLBDown().Attach(onclick)
    
    gform.RunMainLoop()
    
# View
##### Create Manifest `yourappname.manifest`

```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
    <assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
        <assemblyIdentity version="1.0.0.0" processorArchitecture="*" name="SomeFunkyNameHere" type="win32"/>
        <dependency>
            <dependentAssembly>
                <assemblyIdentity type="win32" name="Microsoft.Windows.Common-Controls" version="6.0.0.0" processorArchitecture="*" publicKeyToken="6595b64144ccf1df" language="*"/>
            </dependentAssembly>
        </dependency>
    </assembly>
```

Then either compile the manifest using the [rsrc tool](https://github.com/akavel/rsrc), like this:

	go get github.com/akavel/rsrc
	rsrc -manifest yourappname.manifest -o rsrc.syso

(`go build` is smart enough to pick it up)

or rename the `yourappname.manifest` file to `yourappname.exe.manifest` and distribute it with the application instead.

# Event handling
gform provides two approaches to handle event. For most commonly used events, convenient event handler is introduced. To handle windows message directly, "Bind" mechanism is introduce.

## 1. Convenient event handler.
These kind of event handler follows the same naming convention, "OnSomething".
    
    btn.OnLBUp().Bind(btn_onclick) //LB means Left Button.
    btn.OnMBUp //MB means middle button
    btn.OnKillFocus
    btn.OnDropFiles
    ...

If you bind two methods for one event, the first bind will be overwritten by later bind. E.g.
    
    btn.OnLBUp().Bind(btn_onclick1)
    btn.OnLBUp().Bind(btn_onclick2)

Only "btn_onclick2" will be triggered.

You can also bind "nil" to a event handler, that simply means nothing will be triggered.

## 2. Raw windows message handler.
It's a common case that we need to handler various windows messages in GUI, and to wrap them all is basically "mission impossible" (and I don't think a GUI framework should do that frankly), so gform leaves the freedom to user.
The "Bind" method could bind an event handler directly to a raw windows message. E.g. 

    btn.Bind(w32.WM_CLIPBOARDUPDATE, btn_onClipboardUpdate)

    func btn_onClipboardUpdate(arg *EventArg) {
        sender := arg.Sender()
        if data, ok := arg.Data().(*gform.RawMsg); ok {
            println(data.Hwnd, data.Msg, data.WParam, data.LParam)
        }
    }

The event handler uses the same method signature "func(arg *EventArg)", but a new struct named "RawMsg" will be filled to the "data" field of EventArg.
    
    type RawMsg struct {
        Hwnd           w32.HWND
        Msg            uint
        WParam, LParam uintptr
    } 

The same with convenient event handler, if you bind two methods for one event, the first bind will be overwritten by later bind.
And bind "nil" to a message is allowed.

# Setup

1. Make sure you have a working Go installation and build environment, see more for details from below page.
   http://golang.org/doc/install
   
2. go get github.com/ribtoks/gform

Have fun now!

# Recommand Tools

1. ResEdit - very good tool to edit resource file, strongly recommand!
http://www.resedit.net/

2. windres - tools to compile *.rc file to *.o which makes it is possible to embed resource file into *.exe.

# Contribute

Contributions in form of design, code, documentation, bug reporting or other
ways you see fit are very welcome.

Thank You!
