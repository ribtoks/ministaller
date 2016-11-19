package main

import (
  "path"
  "os"
  "sync"
  "log"
  "sort"
  "io"
  "io/ioutil"
  "errors"
  "path/filepath"
)

const (
  CopyPrice = 100
  RenamePrice = 10

  RemoveFactor = RenamePrice
  UpdateFactor = RenamePrice + CopyPrice
  AddFactor = CopyPrice
)

type BackupPair struct {
  relpath string
  newpath string
}

type ProgressHandler interface {
  HandleSystemMessage(message string)
  HandlePercentChange(percent int)
  HandleFinish()
}

type LogProgressHandler struct {
}

type ProgressReporter struct {
  grandTotal uint64
  currentProgress uint64
  progressChan chan int64
  percent int //0..100
  reportingChan chan bool
  systemMessageChan chan string
  finished chan bool
  progressHandler ProgressHandler
}

type PackageInstaller struct {
  backups map[string]string
  backupsChan chan BackupPair
  progressReporter *ProgressReporter
  installDir string
  packageDir string
  failInTheEnd bool // for debugging purposes
}

func (pi *PackageInstaller) Install(filesProvider UpdateFilesProvider) error {
  pi.progressReporter.grandTotal = pi.calculateGrandTotals(filesProvider)
  go pi.progressReporter.reportingLoop()
  defer close(pi.progressReporter.progressChan)
  defer func() {
    go func () {
      pi.progressReporter.finished <- true
    }()
  }()

  err := pi.installPackage(filesProvider)

  if err == nil {
    pi.afterSuccess()
  } else {
    pi.afterFailure(filesProvider)
  }

  return err
}

func (pi *PackageInstaller) calculateGrandTotals(filesProvider UpdateFilesProvider) uint64 {
  var sum uint64

  for _, fi := range filesProvider.FilesToRemove() {
    sum += uint64(fi.FileSize * RemoveFactor) / 100
  }

  for _, fi := range filesProvider.FilesToUpdate() {
    sum += uint64(fi.FileSize * UpdateFactor) / 100
  }

  for _, fi := range filesProvider.FilesToAdd() {
    sum += uint64(fi.FileSize * AddFactor) / 100
  }

  return sum
}

func (pi *PackageInstaller) installPackage(filesProvider UpdateFilesProvider) (err error) {
  log.Println("Installing package...")

  var wg sync.WaitGroup
  wg.Add(1)
  go func() {
    for bp := range pi.backupsChan {
      pi.backups[bp.relpath] = bp.newpath
    }
    wg.Done()
  }()

  pi.progressReporter.systemMessageChan <- "Removing components"
  err = pi.removeFiles(filesProvider.FilesToRemove())
  if err != nil {
    return err
  }

  pi.progressReporter.systemMessageChan <- "Updating components"
  err = pi.updateFiles(filesProvider.FilesToUpdate())
  if err != nil {
    return err
  }

  pi.progressReporter.systemMessageChan <- "Adding components"
  err = pi.addFiles(filesProvider.FilesToAdd())
  if err != nil {
    return err
  }

  go func() {
    close(pi.backupsChan)
  }()

  wg.Wait()

  if pi.failInTheEnd {
    err = errors.New("Fail by demand")
  }

  return err
}

func (pi *PackageInstaller) afterSuccess() {
  log.Println("After success")
  pi.progressReporter.systemMessageChan <- "Finishing the installation..."
  pi.removeBackups();
  cleanupEmptyDirs(pi.installDir)
}

func (pi *PackageInstaller) afterFailure(filesProvider UpdateFilesProvider) {
  log.Println("After failure")
  pi.progressReporter.systemMessageChan <- "Cleaning up..."
  purgeFiles(pi.installDir, filesProvider.FilesToAdd())
  pi.restoreBackups()
  pi.removeBackups()
  cleanupEmptyDirs(pi.installDir)
}

func copyFile(src, dst string) (err error) {
  in, err := os.Open(src)
  if err != nil {
    log.Printf("Failed to open source: %v", err)
    return
  }

  defer in.Close()

  out, err := os.Create(dst)
  if err != nil {
    log.Printf("Failed to create destination: %v", err)
    return
  }

  defer func() {
    cerr := out.Close()
    if err == nil {
      err = cerr
    }
  }()

  if _, err = io.Copy(out, in); err != nil {
    return
  }

  err = out.Sync()
  return
}

func (pi *PackageInstaller) backupFile(relpath string) error {
  log.Printf("Backing up %v", relpath)

  oldpath := path.Join(pi.installDir, relpath)
  backupPath := relpath + ".bak"

  newpath := path.Join(pi.installDir, backupPath)

  err := os.Rename(oldpath, newpath)

  if err == nil {
    pi.backupsChan <- BackupPair{relpath: relpath, newpath: newpath}
  } else {
    log.Printf("Backup failed: %v", err)
  }

  return err
}

func (pi *PackageInstaller) restoreBackups() {
  log.Printf("Restoring %v backups", len(pi.backups))

  var wg sync.WaitGroup

  for relpath, backuppath := range pi.backups {
    wg.Add(1)

    relativePath := relpath
    pathToRestore := backuppath

    go func() {
      defer wg.Done()

      oldpath := path.Join(pi.installDir, relativePath)
      err := os.Rename(pathToRestore, oldpath)

      if err != nil {
        log.Println(err)
      }
    }()
  }

  wg.Wait()
}

func (pi *PackageInstaller) removeBackups() {
  log.Printf("Removing %v backups", len(pi.backups))

  var wg sync.WaitGroup

  for _, backuppath := range pi.backups {
    wg.Add(1)

    pathToRemove := backuppath

    go func() {
      defer wg.Done()

      err := os.Remove(pathToRemove)
      if err != nil {
        log.Println(err)
      }
    }()
  }

  wg.Wait()
}

func (pi *PackageInstaller) removeFiles(files []*UpdateFileInfo) error {
  log.Printf("Removing %v files", len(files))

  var wg sync.WaitGroup
  errc := make(chan error)
  done := make(chan bool)

  for _, fi := range files {
    wg.Add(1)
    pathToRemove, filesize := fi.Filepath, fi.FileSize

    go func() {
      defer wg.Done()

      select {
      case <-done: return
      default:
      }

      fullpath := filepath.Join(pi.installDir, pathToRemove)
      log.Printf("Removing file %v", fullpath)

      err := pi.backupFile(pathToRemove)

      if err != nil {
        log.Printf("Removing file %v failed", pathToRemove)
        log.Println(err)
        errc <- err
        close(done)
      } else {
        go pi.progressReporter.accountRemove(filesize)
      }
    }()
  }

  go func() {
    errc <- nil
  }()

  wg.Wait()

  if err := <-errc; err != nil {
    return err
  }

  return nil
}

func (pi *PackageInstaller) updateFiles(files []*UpdateFileInfo) error {
  log.Printf("Updating %v files", len(files))

  var wg sync.WaitGroup
  errc := make(chan error)
  done := make(chan bool)

  for _, fi := range files {
    wg.Add(1)

    pathToUpdate, filesize := fi.Filepath, fi.FileSize

    go func() {
      defer wg.Done()

      select {
      case <-done: return
      default:
      }

      oldpath := path.Join(pi.installDir, pathToUpdate)
      log.Printf("Updating file %v", oldpath)

      err := pi.backupFile(pathToUpdate)

      if err == nil {
        newpath := path.Join(pi.packageDir, pathToUpdate)
        err = os.Rename(newpath, oldpath)
      }

      if err != nil {
        log.Printf("Updating file %v failed", pathToUpdate)
        log.Println(err)
        errc <- err
        close(done)
      } else {
        go pi.progressReporter.accountUpdate(filesize)
      }
    }()
  }

  go func() {
    errc <- nil
  }()

  wg.Wait()

  if err := <-errc; err != nil {
    return err
  }

  return nil
}

func (pi *PackageInstaller) addFiles(files []*UpdateFileInfo) error {
  log.Printf("Adding %v files", len(files))

  var wg sync.WaitGroup
  errc := make(chan error)
  done := make(chan bool)

  for _, fi := range files {
    wg.Add(1)

    pathToAdd, filesize := fi.Filepath, fi.FileSize

    go func() {
      defer wg.Done()

      select {
      case <-done: return
      default:
      }

      oldpath := path.Join(pi.installDir, pathToAdd)
      ensureDirExists(oldpath)

      newpath := path.Join(pi.packageDir, pathToAdd)
      err := os.Rename(newpath, oldpath)

      if err != nil {
        log.Printf("Adding file %v failed", pathToAdd)
        log.Println(err)
        errc <- err
        close(done)
      } else {
        go pi.progressReporter.accountAdd(filesize)
      }
    }()
  }

  go func() {
    errc <- nil
  }()

  wg.Wait()

  if err := <-errc; err != nil {
    return err
  }

  return nil
}

func purgeFiles(root string, files []*UpdateFileInfo) {
  log.Printf("Purging %v files", len(files))

  var wg sync.WaitGroup

  for _, fi := range files {
    wg.Add(1)

    fileToPurge := fi.Filepath

    go func() {
      defer wg.Done()

      fullpath := path.Join(root, fileToPurge)
      err := os.Remove(fullpath)
      if err != nil {
        log.Println(err)
      }
    }()
  }

  wg.Wait()
}

func ensureDirExists(fullpath string) (err error) {
  dirpath := path.Dir(fullpath)
  err = os.MkdirAll(dirpath, os.ModeDir)
  if err != nil {
    log.Printf("Failed to create directory %v", dirpath)
  }

  return err
}

type ByLength []string

func (s ByLength) Len() int {
    return len(s)
}
func (s ByLength) Swap(i, j int) {
    s[i], s[j] = s[j], s[i]
}
func (s ByLength) Less(i, j int) bool {
    return len(s[i]) > len(s[j])
}

func cleanupEmptyDirs(root string) {
  c := make(chan string)

  go func() {
    var wg sync.WaitGroup
    err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
      if err != nil {
        return err
      }

      if info.Mode().IsDir() {
        wg.Add(1)
        go func() {
          c <- path
          wg.Done()
        }()
      }

      return nil
    })

    if err != nil {
      log.Println(err)
    }

    go func() {
      wg.Wait()
      close(c)
    }()
  }()

  dirs := make([]string, 0)
  for path := range c {
    dirs = append(dirs, path)
  }

  removeEmptyDirs(dirs)
}

func removeEmptyDirs(dirs []string) {
  sort.Sort(ByLength(dirs))

  for _, dirpath := range dirs {
    entries, err := ioutil.ReadDir(dirpath)
    if err != nil { continue }

    if len(entries) == 0 {
      log.Printf("Removing empty dir %v", dirpath)

      err = os.Remove(dirpath)
      if err != nil {
        log.Println(err)
      }
    }
  }
}

func (pr *ProgressReporter) accountRemove(progress int64) {
  pr.progressChan <- (progress*RemoveFactor)/100
}

func (pr *ProgressReporter) accountUpdate(progress int64) {
  pr.progressChan <- (progress*UpdateFactor)/100
}

func (pr *ProgressReporter) accountAdd(progress int64) {
  pr.progressChan <- (progress*AddFactor)/100
}

func (pr *ProgressReporter) reportingLoop() {
  for chunk := range pr.progressChan {
    pr.currentProgress += uint64(chunk)

    percent := (pr.currentProgress*100) / pr.grandTotal
    pr.percent = int(percent)

    go func() {
      pr.reportingChan <- true
    }()
  }

  close(pr.reportingChan)
}

func (pr *ProgressReporter) receiveUpdates() {
  for _ = range pr.reportingChan {
    pr.progressHandler.HandlePercentChange(pr.percent)
  }
}

func (pr *ProgressReporter) receiveSystemMessages() {
  for msg := range pr.systemMessageChan {
    pr.progressHandler.HandleSystemMessage(msg)
  }
}

func (pr *ProgressReporter) receiveFinish() {
  <- pr.finished
  pr.progressHandler.HandleFinish()
}

func (pr *ProgressReporter) handleProgress() {
  go pr.receiveSystemMessages()
  go pr.receiveUpdates()
  go pr.receiveFinish()
}

func (ph *LogProgressHandler) HandlePercentChange(percent int) {
  log.Printf("Completed %v%%", percent)
}

func (ph *LogProgressHandler) HandleSystemMessage(msg string) {
  log.Printf("System message: %v", msg)
}

func (ph *LogProgressHandler) HandleFinish() {
  log.Printf("Finished")
}
