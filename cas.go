package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
)

type CASWriter interface {
	// save some data and return a hex encoded sha256
	Store(data []byte) (string, error)
}
type CASReader interface {
	Copy(w io.Writer, h string) error
}

type CAS interface {
	CASReader
	CASWriter
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

type httpCAS struct{ addr string }

func (c *httpCAS) Copy(w io.Writer, h string) error {
	resp, err := http.Get(fmt.Sprintf("%s/%s", c.addr, h))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return os.ErrNotExist
		}
		return fmt.Errorf("status ok expected, got %q", resp.Status)
	}

	_, err = io.Copy(w, resp.Body)
	return err
}

type chainedCASReader []CASReader

func (c chainedCASReader) Copy(w io.Writer, h string) error {
	var err error
	for _, r := range c {
		if err = r.Copy(w, h); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}
		break
	}
	return err
}
