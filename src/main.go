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
  installPathFlag = flag.String("install-path", "", "Path to the existing installation")
  packagePathFlag = flag.String("package-path", "", "Path to package with updates")
  forceUpdateFlag = flag.Bool("force-update", false, "Overwrite same files")
  keepMissingFlag = flag.Bool("keep-missing", false, "Keep files not found in the update package")
  logPathFlag = flag.String("l", "ministaller.log", "absolute path to log file")
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

  err = Unzip(*packagePathFlag, packageDirPath)
  if err != nil {
    log.Fatal(err)
  }

  df := DiffGenerator{
    filesToAdd: make([]*UpdateFileInfo, 0),
    filesToRemove: make([]*UpdateFileInfo, 0),
    filesToUpdate: make([]*UpdateFileInfo, 0),
    filesToAddQueue: make(chan *UpdateFileInfo),
    filesToRemoveQueue: make(chan *UpdateFileInfo),
    filesToUpdateQueue: make(chan *UpdateFileInfo),
    errors: make(chan error, 1),
    installDirHashes: make(map[string]string),
    packageDirHashes: make(map[string]string),
    installDirPath: *installPathFlag,
    packageDirPath: packageDirPath,
    keepMissing: *keepMissingFlag,
    forceUpdate: *forceUpdateFlag }

  err = df.GenerateDiffs()
  if err != nil {
    log.Fatal(err)
  }
}

func parseFlags() error {
  flag.Parse()

  installFileInfo, err := os.Stat(*installPathFlag)
  if os.IsNotExist(err) { return err }
  if !installFileInfo.IsDir() { return errors.New("install-path does not point to a directory") }

  packageFileInfo, err := os.Stat(*packagePathFlag)
  if os.IsNotExist(err) { return err }
  if packageFileInfo.IsDir() { return errors.New("package-path should point to a file") }

  return nil
}

func setupLogging() (f *os.File, err error) {
  f, err = os.OpenFile(*logPathFlag, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
  if err != nil {
    fmt.Println("error opening file: %v", *logPathFlag)
    return nil, err
  }

  log.SetOutput(f)
  log.Println("------------------------------")
  log.Println("Ministaller log started")
  
  return f, err
}
