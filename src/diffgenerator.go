package main

import (
  "log"
  "io/ioutil"
)

type UpdateFileInfo struct {
  Filepath string
  Sha1 string
}

type UpdateFilesProvider interface {
  FilesToAdd() []UpdateFileInfo
  FilesToRemove() []UpdateFileInfo
  FilesToUpdate() []UpdateFileInfo
}

type DiffGenerator struct {
  filesToAdd []UpdateFileInfo
  filesToRemove []UpdateFileInfo
  filesToUpdate []UpdateFileInfo
  fileHasher FileHasher
  installDirPath string
  packageDirPath string
  keepMissing bool
  forceUpdate bool  
}

func (df *DiffGenerator) GenerateDiffs() error {
  return df.generateDirectoryDiff(df.installDirPath, df.packageDirPath)
}

func (df *DiffGenerator) generateDirectoryDiff(installDir, packageDir string) error {
  log.Printf("Intall dir: %v, packageDir: %v\n", installDir, packageDir);

  err := df.findFilesToRemoveOrUpdate(installDir, packageDir)
  if err != nil {
    return err
  }

  err = df.findFilesToAdd(installDir, packageDir)
  return err
}

func (df *DiffGenerator) findFilesToRemoveOrUpdate(installDir, packageDir string) error {
  entries, err := ioutil.ReadDir(installDir)
  if err != nil {
    return err
  }

  for _, entry := range entries {
    log.Printf("Checking path %v", entry.Name())
  }

  return nil
}

func (df *DiffGenerator) findFilesToAdd(installDir, packageDir string) error {
  return nil
}


