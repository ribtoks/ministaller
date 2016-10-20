package main

import (
  "fmt"
  "log"
  "flag"
  "os"
  "errors"
  "path"
  "io/ioutil"
  "encoding/json"
  "os/exec"
)

// flags
var (
  installPathFlag = flag.String("install-path", "", "Path to the existing installation")
  packagePathFlag = flag.String("package-path", "", "Path to package with updates")
  forceUpdateFlag = flag.Bool("force-update", false, "Overwrite same files")
  keepMissingFlag = flag.Bool("keep-missing", false, "Keep files not found in the update package")
  logPathFlag = flag.String("l", "ministaller.log", "absolute path to log file")
  launchExeFlag = flag.String("launch-exe", "", "relative path to exe to launch after install")
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

  packageDirPath = findUsefulDir(packageDirPath)
  log.Printf("Using %v for package path", packageDirPath)

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

  backupsDirPath, err := ioutil.TempDir("", appName)
  if err != nil {
    log.Fatal(err)
  }

  defer os.RemoveAll(backupsDirPath)

  pi := PackageInstaller{
    backups: make(map[string]string),
    installDir: *installPathFlag,
    packageDir: packageDirPath,
    backupsDir: backupsDirPath }

  err := pi.Install(df)
  if err != nil {
    log.Printf("Install failed: %v", err)
    return
  }
  
  log.Println("Install succeeded")
  
  if len(*launchExeFlag) > 0 {
    launchPostInstallExe()
  }
}

func findUsefulDir(initialDir string) string {
  entries, err := ioutil.ReadDir(initialDir)
  if err != nil { return initialDir }

  currDir := initialDir

  for (len(entries) == 1) && (entries[0].IsDir()) {
    nextDir := path.Join(currDir, entries[0].Name())
    entries, err = ioutil.ReadDir(nextDir)
    if err != nil { return currDir }
    currDir = nextDir
  }

  return currDir
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

func launchPostInstallExe() {
  fullpath := path.Join(*installPathFlag, *launchExeFlag)
  log.Printf("Trying to launch %v", fullpath)
  
  cmd := exec.Command(fullpath, "")
  err := cmd.Start()
  if err != nil {
    log.Println(err)
  }
}
