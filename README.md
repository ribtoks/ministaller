# ministaller
Lightweight installer/updater for desktop application written in Go

![alt tag](https://raw.githubusercontent.com/Ribtoks/ministaller/master/ministaller.png)

[![Build status](https://ci.appveyor.com/api/projects/status/n32q1fas77p0r90j/branch/master?svg=true)](https://ci.appveyor.com/project/Ribtoks/ministaller/branch/master)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/8636089d05bf4d9dbaa49cd8499f4326)](https://www.codacy.com/app/ribtoks/ministaller)

## Description

This updater is meant for simple safe update of distribution of some main application from update in _zip_ archive. It is capable of partial and full updates (controlled by cmd line parameters) as well as downloading an update with SHA1 hashsum check afterwards. The GUI with simple progress bar is implemented only for Windows OS using direct Win API calls.

It compiles to a fully standalone executable which can be distributed along with the main application. It can be treated as a lightweight and simplified version of a _MaintananceTool_ from Qt world.

## Build

### General instructions

    go get github.com/ribtoks/gform
    git clone https://github.com/ribtoks/ministaller.git
    cd ministaller
    go build -o ministaller.exe -ldflags="-H windowsgui"
    
Check out the `appveyor.yml` file for detailed build instructions.

### x86 instructions only

For native look and feel in Windows it's needed to build an application manifest and embed it as a resource. This is already done for x64 platforms and you don't need to do anything except of `go build`. For x86 you will need to install `rsrc` tool via `go get` and build enclosed manifest for it:

    go get github.com/akavel/rsrc
    rsrc -manifest ministaller.manifest -arch 386 -o rsrc.syso
    
and only then build _ministaller_.

## Usage

Command line switches:

    Usage of ./ministaller:
      -fail
        Fail after install to test rollback
      -force-update
        Overwrite same files
      -gui
        Show simple progress GUI
      -hash string
        Hash of the downloaded file to check
      -install-path string
        Path to the existing installation
      -keep-missing
        Keep files not found in the update package
      -l string
        absolute path to log file (default "ministaller.log")
      -launch-args string
        arguments for launch-exe
      -launch-exe string
        relative path to exe to launch after install
      -package-path string
        Path to package with updates
      -stdout
        Log to stdout and to logfile
      -url string
        Url to the package

Sample usage from Qt application is:

    const QString appDirPath = QCoreApplication::applicationDirPath();
    QDir appDir(appDirPath);
    
    QDir documentsDir(QStandardPaths::writableLocation(QStandardPaths::DocumentsLocation));
    const QString logFilePath = documentsDir.filePath("ministaller.log");

    QStringList arguments;
    arguments << "-force-update" << "-gui" <<
                 "-install-path" << installPath <<
                 "-l" << logFilePath <<
                 "-launch-exe" << "your-main-app.exe" <<
                 "-package-path" << packagePath <<
                 "-stdout";

    QProcess::startDetached(appDir.filePath("ministaller.exe"), arguments);
    
This code worked for me with paths with non-latin and Unicode symbols in Windows 10.
    
## Disclaimer

Theoretically such an application is useless for full update on other platforms but Windows, because OS X has _dmg_ packages which can simply override previous contents (and Sparkle framework otherwise) and updates in Linux and many other \*nix systems are propagated through repositories (or ports).
