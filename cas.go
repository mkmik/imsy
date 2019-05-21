package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"path"
)

type CAS interface {
	// save some data and return a hex encoded sha256
	Store(data []byte) (string, error)
	Copy(w io.Writer, h string) error
}

type dirCAS struct {
	dir string
}

func (c *dirCAS) Store(data []byte) (string, error) {
	h := sha256.Sum256(data)
	s := hex.EncodeToString(h[:])
	p, exists := casFile(c.dir, s)
	if !exists {
		if err := ioutil.WriteFile(p, data, 0666); err != nil {
			return "", err
		}
	}
	return s, nil
}

func (c *dirCAS) Copy(w io.Writer, h string) error {
	p, exists := casFile(c.dir, h)
	if !exists {
		return os.ErrNotExist
	}
	f, err := os.Open(p)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(w, f)
	return err
}

func casFile(casDir string, key string) (filename string, exists bool) {
	p := path.Join(casDir, key)
	if _, err := os.Stat(p); err == nil {
		return p, true
	}
	return p, false
}