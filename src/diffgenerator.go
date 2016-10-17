package main

import (
  "log"
  "io/ioutil"
  "path"
  "os"
  "path/filepath"
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
  generatorQuit chan bool
  fileHasher FileHasher
  installDirPath string
  packageDirPath string
  keepMissing bool
  forceUpdate bool
}

func (df *DiffGenerator) GenerateDiffs() error {
  go df.generateDirectoryDiff(df.installDirPath, df.packageDirPath)

GeneratorLoop:
  for {
    select {
    case fi := <- df.filesToAddQueue: df.filesToAdd = append(df.filesToAdd, fi)
    case fi := <- df.filesToRemoveQueue: df.filesToRemove = append(df.filesToRemove, fi)
    case fi := <- df.filesToUpdateQueue: df.filesToUpdate = append(df.filesToUpdate, fi)
    case <- df.generatorQuit: { break GeneratorLoop }
    }
  }
}

func (df *DiffGenerator) generateDirectoryDiff(installDir, packageDir string) {
  log.Printf("Intall dir: %v, packageDir: %v\n", installDir, packageDir);

  go df.findFilesToRemoveOrUpdate(installDir, packageDir)
  go df.findFilesToAdd(installDir, packageDir)
}

func (df *DiffGenerator) findFilesToRemoveOrUpdate(installDir, packageDir string) error {
  entries, err := ioutil.ReadDir(installDir)
  if err != nil {
    return
  }

DirLoop:
  for _, entry := range entries {
    installedPath := filepath.Join(installDir, entry.Name())
    packagePath := filepath.Join(packageDir, entry.Name())
    log.Printf("Checking path %v", fullpath)

    if !entry.IsDir() {

      resultChan, errChan := df.fileHasher.RetrieveHash(installedPath)

      var hash string
      select {
      case h := <- resultChan: hash = h
      case err = <- errChan: { continue DirLoop }
      }

      relativePath := filepath.Rel(df.installDirPath, installedPath)
      ufi := &UpdateFileInfo{
        filepath: relativePath,
        sha1: hash }

      if _, err := os.Stat(packagePath); os.IsNotExist(err) {
        if !df.keepMissing {
          df.filesToRemove <- ufi
        }
      } else {
        df.filesToUpdate <- ufi
      }
    } else {

    }
  }
}

func (df *DiffGenerator) findFilesToAdd(installDir, packageDir string) error {
  return nil
}
