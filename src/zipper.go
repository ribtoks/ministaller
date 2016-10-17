package main

import (
  "archive/zip"
  "path/filepath"
  "os"
  "io"
  "log"
)

func Unzip(src, dest string) error {
  log.Printf("Extracting %v into %v", src, dest)
  
  r, err := zip.OpenReader(src)
  if err != nil {
    return err
  }

  defer func() {
    if err := r.Close(); err != nil {
      panic(err)
    }
  }()

  extractAndWriteFile := func(f *zip.File) error {
    rc, err := f.Open()
    if err != nil {
      return err
    }
    
    defer func() {
      if err := rc.Close(); err != nil {
        panic(err)
      }
    }()

    path := filepath.Join(dest, f.Name)

    if f.FileInfo().IsDir() {
      os.MkdirAll(path, f.Mode())
    } else {
      os.MkdirAll(filepath.Dir(path), f.Mode())
      f, err := os.OpenFile(path, os.O_WRONLY | os.O_CREATE | os.O_TRUNC, f.Mode())
      if err != nil {
        return err
      }
      
      defer func() {
        if err := f.Close(); err != nil {
          panic(err)
        }
      }()

      _, err = io.Copy(f, rc)
      if err != nil {
        return err
      }
    }
    
    return nil
  }

  for _, f := range r.File {
    err := extractAndWriteFile(f)
    if err != nil {
      return err
    }
  }

  return nil
}
