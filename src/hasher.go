package main

import (
  "crypto/sha1"
  "os"
  "io"
  "encoding/hex"
  "path/filepath"
  "sync"
)

type HashResult struct {
  path string
  hash string
  err error
}

func CalculateHashes(root string) (map[string]string, err) {
  c, errc := calculateSha1Hashes(root)

  m := make(map[string]string)
  for r := range c {
    if r.err != nil {
      return nil, r.err
    }

    key := filepath.Rel(root, r.path)
    m[key] = r.hash
  }

  if err := <- errc; err != nil {
    return nil, err
  }

  return m, nil
}

func calculateSha1Hashes(root string) (<-chan HashResult, <-chan error) {
  c := make(chan HashResult)
  errc := make(chan error, 1)

  go func() {
    var wg sync.WaitGroup
    err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
      if err != nil {
        return err
      }

      if !info.Mode().IsRegular() {
        return nil
      }

      wg.Add(1)

      go func() {
        hash, err := calculateFileHash(path)
        c <- HashResult{path, hash, err}

        wg.Done()
      }()
    })

    go func() {
      wg.Wait()
      close(c)
    }()

    errc <- err
  }()

  return c, errc
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
