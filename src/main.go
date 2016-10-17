package main

import (
  "fmt"
  "log"
  "flag"
  "os"
  "errors"
  "io/ioutil"
)

// flags
var (
  installPath = flag.String("install-path", "", "Path to the existing installation")
  packagePath = flag.String("package-path", "", "Path to package with updates")
  forceUpdate = flag.Bool("force-update", false, "Overwrite same files")
  keepMissing = flag.Bool("keep-missing", false, "Keep files not found in the update package")
  logPath = flag.String("l", "ministaller.log", "absolute path to log file")
)

const (
  appName = "ministaller"
)

func main() {
  err := parseFlags()
  if err != nil {
    flag.PrintDefaults()
    log.Fatal(err.Error())
  }

  logfile, err := setupLogging()
  if err != nil {
    defer logfile.Close()
  }

  packageDirPath, err := ioutil.TempDir("", appName)
  if err != nil {
    log.Fatal(err)
  }

  defer os.RemoveAll(packageDirPath)

  err = Unzip(*packagePath, packageDirPath)
  if err != nil {
    log.Fatal(err)
  }
}

func parseFlags() error {
  flag.Parse()

  installFileInfo, err := os.Stat(*installPath)
  if os.IsNotExist(err) { return err }
  if !installFileInfo.IsDir() { return errors.New("install-path does not point to a directory") }

  packageFileInfo, err := os.Stat(*packagePath)
  if os.IsNotExist(err) { return err }
  if packageFileInfo.IsDir() { return errors.New("package-path should point to a file") }

  return nil
}

func setupLogging() (f *os.File, err error) {
  f, err = os.OpenFile(*logPath, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
  if err != nil {
    fmt.Println("error opening file: %v", *logPath)
    return nil, err
  }

  log.SetOutput(f)
  log.Println("------------------------------")
  log.Println("Ministaller log started")
  
  return f, err
}
