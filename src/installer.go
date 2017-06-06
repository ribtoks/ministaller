package main

import (
  "path"
  "os"
  "sync"
  "log"
  "sort"
  "io"
  "io/ioutil"
  "path/filepath"
  "os/exec"
)

const (
  CopyPrice = 100
  RenamePrice = 10

  RemoveBackupPrice = 1000
  RemoveFactor = RenamePrice
  UpdateFactor = RenamePrice + CopyPrice
  AddFactor = CopyPrice
)

const (
  BackupExt = ".bak"
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
  progressWG sync.WaitGroup
  percent int //0..100
  systemMessageChan chan string
  finished chan bool
  progressHandler ProgressHandler
}

type PackageInstaller struct {
  backups map[string]string
  backupsChan chan BackupPair
  backupsWG sync.WaitGroup
  progressReporter *ProgressReporter
  installDir string
  packageDir string
  removeSelfPath string // if updating the installer
  failInTheEnd bool // for debugging purposes
}

func (pi *PackageInstaller) Install(filesProvider UpdateFilesProvider) error {
  pi.progressReporter.grandTotal = pi.calculateGrandTotals(filesProvider)
  
  go pi.progressReporter.reportingLoop()

  pi.beforeInstall()

  err := pi.installPackage(filesProvider)

  if (err == nil) && (!pi.failInTheEnd) {
    pi.afterSuccess()
  } else {
    pi.afterFailure(filesProvider)
  }
  
  pi.progressReporter.waitProgressReported()
  pi.progressReporter.shutdown()

  return err
}

func (pi *PackageInstaller) calculateGrandTotals(filesProvider UpdateFilesProvider) uint64 {
  var sum uint64

  for _, fi := range filesProvider.FilesToRemove() {
    sum += uint64(fi.FileSize * RemoveFactor) / 100
    sum += uint64(RemoveBackupPrice)
  }

  for _, fi := range filesProvider.FilesToUpdate() {
    sum += uint64(fi.FileSize * UpdateFactor) / 100
    sum += uint64(RemoveBackupPrice)
  }

  for _, fi := range filesProvider.FilesToAdd() {
    sum += uint64(fi.FileSize * AddFactor) / 100
  }

  return sum
}

func (pi *PackageInstaller) beforeInstall() {
  pi.removeOldBackups()
}

func (pi *PackageInstaller) installPackage(filesProvider UpdateFilesProvider) (err error) {
  log.Println("Installing package...")

  go func() {
    for bp := range pi.backupsChan {
      pi.backups[bp.relpath] = bp.newpath
      pi.backupsWG.Done()
    }
    
    log.Printf("Backups accounting finished. %v backups available", len(pi.backups))
  }()
  
  defer func() {
    log.Println("Stopping backups accounting routine...")
    close(pi.backupsChan)
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

  log.Println("Waiting for backups to finish accounting...")
  pi.backupsWG.Wait()

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
  log.Printf("About to copy file %v to %v", src, dst)

  fi, err := os.Stat(src)
  if err != nil { return err }
  sourceMode := fi.Mode()

  in, err := os.Open(src)
  if err != nil {
    log.Printf("Failed to open source: %v", err)
    return err
  }

  defer in.Close()

  out, err := os.OpenFile(dst, os.O_RDWR | os.O_TRUNC | os.O_CREATE, sourceMode)
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
  backupPath := relpath + BackupExt

  newpath := path.Join(pi.installDir, backupPath)
  // remove previous backup if any
  os.Remove(newpath)

  err := os.Rename(oldpath, newpath)

  if err == nil {
    pi.backupsWG.Add(1)
    go func() {
      pi.backupsChan <- BackupPair{relpath: relpath, newpath: newpath}
    }()
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

    go func(relativePath, pathToRestore string) {
      defer wg.Done()

      oldpath := path.Join(pi.installDir, relativePath)
      log.Printf("Restoring %v to %v", pathToRestore, oldpath)
      err := os.Rename(pathToRestore, oldpath)

      if err != nil {
        log.Printf("Error while restoring %v: %v", pathToRestore, err)
      }
    }(relpath, backuppath)
  }

  wg.Wait()
}

func (pi *PackageInstaller) removeOldBackups() {
  backeduppath := currentExeFullPath + BackupExt
  err := os.Remove(backeduppath)
  if err == nil {
    log.Println("Old installer backup removed", backeduppath)
  } else if os.IsNotExist(err) {
    log.Println("Old installer backup was not found")
  } else {
    log.Printf("Error while removing old backup: %v", err)
  }
}

func (pi *PackageInstaller) removeBackups() {
  log.Printf("Removing %v backups", len(pi.backups))

  selfpath, err := filepath.Rel(pi.installDir, currentExeFullPath)
  if err == nil {
    if backuppath, ok := pi.backups[selfpath]; ok {
      pi.removeSelfPath = backuppath
      delete(pi.backups, selfpath)
      log.Printf("Removed exe path %v from backups", selfpath)
    }
  }

  var wg sync.WaitGroup

  for _, backuppath := range pi.backups {
    wg.Add(1)

    go func(pathToRemove string) {
      defer wg.Done()

      err := os.Remove(pathToRemove)
      if err != nil {
        log.Printf("Error while removing %v: %v", pathToRemove, err)
      }

      pi.progressReporter.accountBackupRemove()
    }(backuppath)
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
        log.Printf("Removing file %v failed: %v", pathToRemove, err)
        errc <- err
        close(done)
      } else {
        pi.progressReporter.accountRemove(filesize)
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
        pi.progressReporter.accountUpdate(filesize)
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
      
      log.Printf("Adding file %v", pathToAdd)

      if err != nil {
        log.Printf("Adding file %v failed: %v", pathToAdd, err)
        errc <- err
        close(done)
      } else {
        pi.progressReporter.accountAdd(filesize)
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

func (pi *PackageInstaller) removeSelfIfNeeded() {
  if len(pi.removeSelfPath) == 0 {
    log.Println("No need to remove itself")
    return
  }

  // TODO: move this to windows-only define
  pathToRemove := filepath.FromSlash(pi.removeSelfPath)
  log.Println("Removing exe backup", pathToRemove)
  cmd := exec.Command("cmd", "/C", "ping localhost -n 2 -w 5000 > nul & del", pathToRemove)
  err := cmd.Start()
  if err != nil {
    log.Println(err)
  }
}

func purgeFiles(root string, files []*UpdateFileInfo) {
  log.Printf("Purging %v files", len(files))

  var wg sync.WaitGroup

  for _, fi := range files {
    wg.Add(1)

    go func(fileToPurge string) {
      defer wg.Done()

      fullpath := path.Join(root, fileToPurge)
      err := os.Remove(fullpath)
      if err != nil {
        log.Printf("Error while purging %v: %v", fullpath, err)
      }
    }(fi.Filepath)
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
  dirs := make([]string, 0, 10)

  err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
    if err != nil {
      return err
    }

    if info.Mode().IsDir() {  
      dirs = append(dirs, path)
    }
    
    return nil
  })

  if err != nil {
    log.Printf("Error while cleaning up empty dirs: %v", err)
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
        log.Printf("Error while removing dir %v: %v", dirpath, err)
      }
    }
  }
}

func (pr *ProgressReporter) accountRemove(progress int64) {
  pr.progressWG.Add(1)
  go func() {
    pr.progressChan <- (progress*RemoveFactor)/100
  }()
}

func (pr *ProgressReporter) accountUpdate(progress int64) {
  pr.progressWG.Add(1)
  go func() {
    pr.progressChan <- (progress*UpdateFactor)/100
  }()
}

func (pr *ProgressReporter) accountAdd(progress int64) {
  pr.progressWG.Add(1)
  go func() {
    pr.progressChan <- (progress*AddFactor)/100
  }()
}

func (pr *ProgressReporter) accountBackupRemove() {
  // exact size of files is not known when removeBackups()
  // so using some arbitrary value (fair dice roll)
  pr.progressWG.Add(1)
  go func() {
    pr.progressChan <- RemoveBackupPrice
  }()
}

func (pr *ProgressReporter) reportingLoop() {
  for chunk := range pr.progressChan {
    pr.currentProgress += uint64(chunk)

    percent := (pr.currentProgress*100) / pr.grandTotal
    pr.percent = int(percent)
      
    pr.progressHandler.HandlePercentChange(pr.percent)
    pr.progressWG.Done()
  }
  
  log.Println("Reporting loop finished")
}

func (pr *ProgressReporter) waitProgressReported() {
  log.Println("Waiting for progress reporting to finish")
  pr.progressWG.Wait()
}

func (pr *ProgressReporter) shutdown() {
  log.Println("Shutting down progress reporter...")
  close(pr.progressChan)
  go func() {
    pr.finished <- true
  }()
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
