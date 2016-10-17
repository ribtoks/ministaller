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
  "fmt"
  "log"
  "flag"
  "os"
  "errors"
  "io/ioutil"
)

// flags
var (
  installPath = flag.String("install-path", "", "Path to the existing installation")
  packagePath = flag.String("package-path", "", "Path to package with updates")
  forceUpdate = flag.Bool("force-update", false, "Overwrite same files")
  keepMissing = flag.Bool("keep-missing", false, "Keep files not found in the update package")
  logPath = flag.String("l", "ministaller.log", "absolute path to log file")
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

  err = Unzip(*packagePath, packageDirPath)
  if err != nil {
    log.Fatal(err)
  }
}

func parseFlags() error {
  flag.Parse()

  installFileInfo, err := os.Stat(*installPath)
  if os.IsNotExist(err) { return err }
  if !installFileInfo.IsDir() { return errors.New("install-path does not point to a directory") }

  packageFileInfo, err := os.Stat(*packagePath)
  if os.IsNotExist(err) { return err }
  if packageFileInfo.IsDir() { return errors.New("package-path should point to a file") }

  return nil
}

func setupLogging() (f *os.File, err error) {
  f, err = os.OpenFile(*logPath, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
  if err != nil {
    fmt.Println("error opening file: %v", *logPath)
    return nil, err
  }

  log.SetOutput(f)
  log.Println("------------------------------")
  log.Println("Ministaller log started")
  
  return f, err
}
