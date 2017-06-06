package main

import (
  "crypto/sha1"
  "os"
  "io"
  "encoding/hex"
  "path/filepath"
  "sync"
  "log"
)

type HashResult struct {
  path string
  hash string
  err error
}

func CalculateHashes(root string) (map[string]string, error) {
  var wg sync.WaitGroup
  c := make(chan HashResult)

  go calculateSha1Hashes(root, &wg, c)

  m := make(map[string]string)

  for r := range c {
    wg.Done()

    if r.err != nil {
      log.Printf("Error while calculating hash: %v", r.err)
      continue
    }

    key, err := filepath.Rel(root, r.path)
    if err != nil {
      log.Printf("Error while calculating relative path: %v", err)
    } else {
      key = filepath.ToSlash(key)
      m[key] = r.hash
    }
  }

  log.Printf("Hashes accounting finished")

  return m, nil
}

func calculateSha1Hashes(root string, wg *sync.WaitGroup, c chan HashResult) {
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
    }()

    return nil
  })

  if err != nil { log.Printf("Error while hashing: %v", err) }

  wg.Wait()
  close(c)

  log.Println("Hashing generation finished")
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
