/*
 * This file is a part of Ministaller - lightweight updater for
 * portable applications
 * Copyright (C) 2016 Taras Kushnir <kushnirTV@gmail.com>
 *
 * Ministaller is distributed under the GNU General Public License, version 3.0
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package ministaller

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
