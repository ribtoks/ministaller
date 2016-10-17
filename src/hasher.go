package main

import (
  "crypto/sha1"
  "os"
  "io"
  "encoding/hex"
)

type FileHasher map[string]string

func (fh *FileHasher) RetrieveHash(filepath string) (string, error) {
  hash, ok := (*fh)[filepath];
  if !ok {
    filehash, err := calculateFileHash(filepath)
    if err != nil {
      return "", err
    }

    (*fh)[filepath] = filehash
    hash = filehash
  }

  return hash, nil
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


