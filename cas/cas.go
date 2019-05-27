package cas

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
)

type Writer interface {
	// save some data and return a hex encoded sha256
	Store(data []byte) (string, error)
}
type Reader interface {
	Copy(w io.Writer, h string) error
}

type ReadWriter interface {
	Reader
	Writer
}

type Dir struct {
	Dir string
}

func (c *Dir) Store(data []byte) (string, error) {
	h := sha256.Sum256(data)
	s := hex.EncodeToString(h[:])
	p, exists := casFile(c.Dir, s)
	if !exists {
		if err := ioutil.WriteFile(p, data, 0666); err != nil {
			return "", err
		}
	}
	return s, nil
}

func (c *Dir) Copy(w io.Writer, h string) error {
	p, exists := casFile(c.Dir, h)
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

type HTTPReader struct{ Addr string }

func (c *HTTPReader) Copy(w io.Writer, h string) error {
	resp, err := http.Get(fmt.Sprintf("%s/%s", c.Addr, h))
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

// a ChainedReader reads from CAS readers until one doesn't return error
type ChainedReader struct {
	Readers []Reader
	Hits    []int
}

func (c *ChainedReader) Copy(w io.Writer, h string) error {
	if l := len(c.Readers); len(c.Hits) != l {
		c.Hits = make([]int, l)
	}
	var err error
	for i, r := range c.Readers {
		if err = r.Copy(w, h); err == nil {
			c.Hits[i]++
			break
		}
	}
	return err
}

func (c *ChainedReader) PrettyHits() string {
	total := 0
	for _, h := range c.Hits {
		total += h
	}

	var buf strings.Builder
	for _, h := range c.Hits {
		fmt.Fprintf(&buf, "%.2f%% ", float64(h)/float64(total))
	}
	return buf.String()
}

// a caching Reader is a CAS that stores what it successfully reads.
type CachingReader struct {
	R Reader
	W Writer
}

func (c CachingReader) Copy(w io.Writer, h string) error {
	var buf bytes.Buffer
	if err := c.R.Copy(&buf, h); err != nil {
		return err
	}
	if _, err := c.W.Store(buf.Bytes()); err != nil {
		return err
	}
	_, err := io.Copy(w, &buf)
	return err
}
