package main

import (
  "crypto/sha1"
  "os"
  "io"
  "encoding/hex"
)

func CalculateHash(filepath string) (<-chan string, <-chan error) {
  resultChan := make(chan string)
  errChan := make(chan error)

  go func() {
    filehash, err := calculateFileHash(filepath)
    if err != nil {
      errChan <- err
    } else {
      resultChan <- filehash
    }
  }()

  return resultChan, errChan
}

func calculateFileHash(filepath string) (string, error) {
  f, err := os.Open(filepath)
  if err != nil {
    return "", err
  }

  defer f.Close()

  hasher := sha1.New()

  if _, err := io.Copy(hasher, f); err != nil {
    return "", err
  }

  hashBytes := hasher.Sum(nil)
  hexStr := hex.EncodeToString(hashBytes)
  return hexStr, nil
}
