package main

import (
  "log"
  "os"
  "path/filepath"
  "sync"
)

type UpdateFileInfo struct {
  filepath string
  sha1 string
}

type UpdateFilesProvider interface {
  FilesToAdd() []*UpdateFileInfo
  FilesToRemove() []*UpdateFileInfo
  FilesToUpdate() []*UpdateFileInfo
}

type DiffGenerator struct {
  filesToAdd []*UpdateFileInfo
  filesToRemove []*UpdateFileInfo
  filesToUpdate []*UpdateFileInfo
  filesToAddQueue chan *UpdateFileInfo
  filesToRemoveQueue chan *UpdateFileInfo
  filesToUpdateQueue chan *UpdateFileInfo
  errors chan error
  installDirHashes map[string]string
  packageDirHashes map[string]string
  installDirPath string
  packageDirPath string
  keepMissing bool
  forceUpdate bool
}

func (df *DiffGenerator) GenerateDiffs() error {
  err := df.calculateHashes()
  if err != nil {
    return err
  }

  var wg sync.WaitGroup

  wg.Add(1)
  go func() {
    for fi := range df.filesToAddQueue {
      df.filesToAdd = append(df.filesToAdd, fi)
    }

    log.Println("Finished processing files to add")
    wg.Done()
  }()

  wg.Add(1)
  go func() {    
    for fi := range df.filesToRemoveQueue {
      df.filesToRemove = append(df.filesToRemove, fi)
    }

    log.Println("Finished processing files to remove")
    wg.Done()
  }()

  wg.Add(1)
  go func() {
    for fi := range df.filesToUpdateQueue {
      df.filesToUpdate = append(df.filesToUpdate, fi)
    }

    log.Println("Finished processing files to update")
    wg.Done()
  }()
  
  go df.generateDirectoryDiff(df.installDirPath, df.packageDirPath)
  go func() {
    wg.Wait()
    df.errors <- nil
  }()

  if err := <- df.errors; err != nil {
    return err
  }

  return nil
}

func (df *DiffGenerator) calculateHashes() error {
  log.Println("Calculating hashes...")
  
  ierrc := make(chan error, 1)
  perrc := make(chan error, 1)

  go func() {
    var err error
    df.installDirHashes, err = CalculateHashes(df.installDirPath)
    ierrc <- err
  }()

  go func() {
    var err error
    df.packageDirHashes, err = CalculateHashes(df.packageDirPath)
    perrc <- err
  }()

  log.Println("Hashes calculated")

  if err := <- ierrc; err != nil { return err }
  if err := <- perrc; err != nil { return err }

  return nil
}

func (df *DiffGenerator) generateDirectoryDiff(installDir, packageDir string) {
  log.Printf("Install dir: %v, packageDir: %v\n", installDir, packageDir);

  go df.findFilesToRemoveOrUpdate(installDir, packageDir)
  go df.findFilesToAdd(installDir, packageDir)
}

func (df *DiffGenerator) findFilesToRemoveOrUpdate(installDir, packageDir string) {
  var wg sync.WaitGroup
  err := filepath.Walk(installDir, func(path string, info os.FileInfo, err error) error {
    if err != nil {
      return err
    }

    if !info.Mode().IsRegular() {
      return nil
    }

    wg.Add(1)

    go func() {
      relativePath, err := filepath.Rel(df.installDirPath, path)
      if err != nil { log.Fatal(err) }
      packagePath := filepath.Join(df.packageDirPath, relativePath)
      installFileHash := df.installDirHashes[relativePath]

      ufi := &UpdateFileInfo{
        filepath: relativePath,
        sha1: installFileHash }

      if _, err := os.Stat(packagePath); os.IsNotExist(err) {
        if !df.keepMissing {
          df.filesToRemoveQueue <- ufi
        }
      } else {
        packageFileHash := df.packageDirHashes[relativePath]

        if (packageFileHash != installFileHash) || (df.forceUpdate) {
          df.filesToUpdateQueue <- ufi
        }
      }

      wg.Done()
    }()

    return nil
  })

  go func() {
    wg.Wait()
    close(df.filesToRemoveQueue)
    close(df.filesToUpdateQueue)
  }()

  if err != nil {
    df.errors <- err
  }
}

func (df *DiffGenerator) findFilesToAdd(installDir, packageDir string) {
  var wg sync.WaitGroup
  err := filepath.Walk(packageDir, func(path string, info os.FileInfo, err error) error {
    if err != nil {
      return err
    }

    if !info.Mode().IsRegular() {
      return nil
    }

    wg.Add(1)

    go func() {
      relativePath, err := filepath.Rel(df.installDirPath, path)
      if err != nil { log.Fatal(err) }
      installPath := filepath.Join(df.installDirPath, relativePath)

      if _, err := os.Stat(installPath); os.IsNotExist(err) {
        packageFileHash := df.packageDirHashes[relativePath]
        
        df.filesToAddQueue <- &UpdateFileInfo{
          filepath: relativePath,
          sha1: packageFileHash }
      }

      wg.Done()
    }()

    return nil
  })

  go func() {
    wg.Wait()
    close(df.filesToAddQueue)
  }()

  if err != nil {
    df.errors <- err
  }
}
