package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/restic/chunker"
)

const (
	pol chunker.Pol = 0x2652bce9495479

	kib = 1024
	mib = 1024 * kib

	minChunk = 64 * kib
	maxChunk = 1 * mib
)

var (
	casDir = flag.String("cas-dir", "cas", "directory to store chunks")
)

type CAS interface {
	// save some data and return a hex encoded sha256
	store(data []byte) (string, error)
}

type dirCAS struct {
	dir string
}

func (c *dirCAS) store(data []byte) (string, error) {
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

func casFile(casDir string, key string) (filename string, exists bool) {
	p := path.Join(casDir, key)
	if _, err := os.Stat(p); err == nil {
		return p, true
	}
	return p, false
}

// prepare takes a binary file and saves a list of chunk hashes, one per line,
// while saving chunks in CAS.
// The resulting hash list is then also stored in the CAS and its key (its hash)
// is printed.
func prepare(w io.Writer, rd io.Reader, casDir string) error {
	cas := &dirCAS{dir: casDir}

	var hlistBuf bytes.Buffer

	cer := chunker.NewWithBoundaries(rd, pol, minChunk, maxChunk)
	for {
		ch, err := cer.Next(nil)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		h, err := cas.store(ch.Data)
		if err != nil {
			return err
		}
		fmt.Fprintln(&hlistBuf, h)
	}

	lh, err := cas.store(hlistBuf.Bytes())
	if err != nil {
		return err
	}
	fmt.Fprintln(w, lh)
	return nil
}

func main() {
	flag.Parse()
	os.MkdirAll(*casDir, 0777)

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	out := os.Stdout
	in := os.Stdin

	var err error
	switch cmd := flag.Arg(0); cmd {
	case "prepare":
		err = prepare(out, in, *casDir)
	default:
		err = fmt.Errorf("unknown command %q", cmd)
	}
	if err != nil {
		log.Fatal(err)
	}
}
