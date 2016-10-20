package main

import (
  "path"
  "os"
  "fmt"
  "sync"
  "log"
  "sort"
  "io/ioutil"
)

type PackageInstaller struct {
  backups map[string]string
  installDir string
  packageDir string
  backupsDir string
}

func (pi *PackageInstaller) installPackage(filesProvider UpdateFilesProvider) error {
  _ = pi.removeFiles(filesProvider.FilesToRemove())
  return nil
}

func (pi *PackageInstaller) backupFile(relpath string) error {
  oldpath := path.Join(pi.installDir, relpath)
  backupPath := fmt.Sprintf("%v.bak", relpath)

  newpath := path.Join(pi.backupsDir, backupPath)
  ensureDirExists(newpath)

  err := os.Rename(oldpath, newpath)
  if err == nil {
    pi.backups[relpath] = newpath
  }

  return err
}

func (pi *PackageInstaller) restoreBackups() {
  log.Printf("Restoring %v backups", len(pi.backups))

  var wg sync.WaitGroup

  for relpath, backuppath := range pi.backups {
    wg.Add(1)

    go func() {
      defer wg.Done()

      oldpath := path.Join(pi.installDir, relpath)
      err := os.Rename(backuppath, oldpath)

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

    go func() {
      defer wg.Done()

      err := os.Remove(backuppath)
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

    go func() {
      defer wg.Done()

      select {
      case <-done: return
      default:
      }

      fullpath := path.Join(pi.installDir, fi.Filepath)

      err := pi.backupFile(fullpath)

      if err == nil {
        err = os.Remove(fullpath)
      }

      if err != nil {
        log.Printf("Removing file %v failed", fi)
        log.Println(err)
        errc <- err
        close(done)
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

    go func() {
      defer wg.Done()

      select {
      case <-done: return
      default:
      }

      oldpath := path.Join(pi.installDir, fi.Filepath)

      err := pi.backupFile(oldpath)

      if err == nil {
        newpath := path.Join(pi.packageDir, fi.Filepath)
        err = os.Rename(newpath, oldpath)
      }

      if err != nil {
        log.Printf("Updating file %v failed", fi)
        log.Println(err)
        errc <- err
        close(done)
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

    go func() {
      defer wg.Done()

      select {
      case <-done: return
      default:
      }

      oldpath := path.Join(pi.installDir, fi.Filepath)

      newpath := path.Join(pi.packageDir, fi.Filepath)
      err := os.Rename(newpath, oldpath)

      if err != nil {
        log.Printf("Adding file %v failed", fi)
        log.Println(err)
        errc <- err
        close(done)
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

func (pi *PackageInstaller) removeFilesToAdd(files []*UpdateFileInfo) {
  var wg sync.WaitGroup

  for _, fi := range files {
    wg.Add(1)

    go func() {
      defer wg.Done()

      fullpath := path.Join(pi.installDir, fi.Filepath)
      err := os.Remove(fullpath)
      if err != nil {
        log.Println(err)
      }
    }()
  }

  wg.Wait()
}

func ensureDirExists(fullpath string) (err error) {
  err = os.MkdirAll(path.Dir(fullpath), os.ModeDir)
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

func cleanupEmptyDirs(root string) error {
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

    go func() {
      wg.Wait()
      close(c)
    }()
  }()

  dirs := make([]string)
  for path := range c {
    dirs = append(dirs, c)
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
        log.Prinln(err)
      }
    }
  }
}
