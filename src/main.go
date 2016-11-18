package main

import (
  "fmt"
  "log"
  "flag"
  "os"
  "io"
  "errors"
  "path"
  "path/filepath"
  "io/ioutil"
  "os/exec"
  "net/http"
)

// flags
var (
  installPathFlag = flag.String("install-path", "", "Path to the existing installation")
  packagePathFlag = flag.String("package-path", "", "Path to package with updates")
  forceUpdateFlag = flag.Bool("force-update", false, "Overwrite same files")
  keepMissingFlag = flag.Bool("keep-missing", false, "Keep files not found in the update package")
  logPathFlag = flag.String("l", "ministaller.log", "absolute path to log file")
  launchExeFlag = flag.String("launch-exe", "", "relative path to exe to launch after install")
  failFlag = flag.Bool("fail", false, "Fail after install to test rollback")
  stdoutFlag = flag.Bool("stdout", false, "Log to stdout and to logfile")
  urlFlag = flag.String("url", "", "Url to the package")
  hashFlag = flag.String("hash", "", "Hash of the downloaded file to check")
  showUIFlag = flag.Bool("gui", false, "Show simple progress GUI")
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

  pathToArchive := *packagePathFlag

  if len(*urlFlag) > 0 {
    localPath, err := downloadFile(*urlFlag)
    if err != nil {
      log.Fatal(err.Error())
    }

    defer os.Remove(localPath)

    hash, err := calculateFileHash(localPath)
    if err != nil {
      log.Println(err.Error())
    } else {
      if hash != *hashFlag {
        log.Printf("Hash mismatch! %v expected but %v found", *hashFlag, hash)
      } else {
        log.Println("Download succeeded")
        pathToArchive = localPath
      }
    }
  }

  packageDirPath, err := ioutil.TempDir("", appName)
  if err != nil {
    log.Fatal(err)
  }

  defer os.RemoveAll(packageDirPath)

  err = Unzip(pathToArchive, packageDirPath)
  if err != nil {
    log.Fatal(err)
  }

  packageDirPath = findUsefulDir(packageDirPath)
  packageDirPath = filepath.ToSlash(packageDirPath)
  log.Printf("Using %v for package path", packageDirPath)

  installDirPath := filepath.ToSlash(*installPathFlag)
  log.Printf("Using %v for install path", installDirPath)

  df := &DiffGenerator{
    filesToAdd: make([]*UpdateFileInfo, 0),
    filesToRemove: make([]*UpdateFileInfo, 0),
    filesToUpdate: make([]*UpdateFileInfo, 0),
    filesToAddQueue: make(chan *UpdateFileInfo),
    filesToRemoveQueue: make(chan *UpdateFileInfo),
    filesToUpdateQueue: make(chan *UpdateFileInfo),
    errors: make(chan error, 1),
    installDirHashes: make(map[string]string),
    packageDirHashes: make(map[string]string),
    installDirPath: installDirPath,
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

  backupsDirPath = filepath.ToSlash(backupsDirPath)

  defer os.RemoveAll(backupsDirPath)

  progressReporter := &ProgressReporter{
    progressChan: make(chan int64),
    reportingChan: make(chan bool),
    systemMessageChan: make(chan string),
    finished: make(chan bool),
  }

  go progressReporter.receiveSystemMessages(onSystemMessage)
  go progressReporter.receiveUpdates(onPercentUpdate)
  go progressReporter.receiveFinish(onFinished)

  pi := &PackageInstaller{
    backups: make(map[string]string),
    backupsChan: make(chan BackupPair),
    progressReporter: progressReporter,
    installDir: installDirPath,
    packageDir: packageDirPath,
    backupsDir: backupsDirPath,
    failInTheEnd: *failFlag }

  if *showUIFlag {
    go doInstall(pi, df)
    guiloop()
  } else {
    doInstall(pi, df)
  }
}

func doInstall(pi *PackageInstaller, df *DiffGenerator) {
  err := pi.Install(df)

  if err == nil {
    log.Println("Install succeeded")
    if len(*launchExeFlag) > 0 {
      launchPostInstallExe()
    }
  } else {
    log.Printf("Install failed: %v", err)
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

  if len(*urlFlag) == 0 {
    packageFileInfo, err := os.Stat(*packagePathFlag)
    if os.IsNotExist(err) { return err }
    if packageFileInfo.IsDir() { return errors.New("package-path should point to a file") }
  }

  return nil
}

func setupLogging() (f *os.File, err error) {
  f, err = os.OpenFile(*logPathFlag, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
  if err != nil {
    fmt.Println("error opening file: %v", *logPathFlag)
    return nil, err
  }

  if *stdoutFlag {
    mw := io.MultiWriter(os.Stdout, f)
    log.SetOutput(mw)
  } else {
    log.SetOutput(f)
  }

  log.Println("------------------------------")
  log.Println("Ministaller log started")

  return f, err
}

func downloadFile(remoteAddr string) (filepath string, err error) {
  log.Printf("Downloading %v", remoteAddr)

  tempfile, err := ioutil.TempFile("", appName)
  if err != nil {
    return "", err
  }
  defer tempfile.Close()

  resp, err := http.Get(remoteAddr)
  defer resp.Body.Close()

  n, err := io.Copy(tempfile, resp.Body)
  if err != nil {
    return "", err
  }

  log.Printf("Downloaded %v bytes", n)

  return tempfile.Name(), nil
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
