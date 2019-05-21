package main

import (
	"bytes"
	"crypto/sha256"
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

func chunkFile(casDir string, h string) (filename string, exists bool) {
	p := path.Join(casDir, h)
	if _, err := os.Stat(p); err == nil {
		return p, true
	}
	return p, false
}

func saveChunk(casDir string, h []byte, data []byte) error {
	n := fmt.Sprintf("%x", h)
	p, exists := chunkFile(casDir, n)
	if exists {
		return nil
	}

	ioutil.WriteFile(p, data, 0666)
	return nil
}

// prepare takes a binary file and saves a list of chunk hashes, one per line,
// while saving chunks in CAS.
// The resulting hash list is then also stored in the CAS and its key (its hash)
// is printed.
func prepare(w io.Writer, rd io.Reader, casDir string) error {
	var hlistBuf bytes.Buffer
	hlistHasher := sha256.New()
	hlistW := io.MultiWriter(&hlistBuf, hlistHasher)

	cer := chunker.NewWithBoundaries(rd, pol, minChunk, maxChunk)
	for {
		ch, err := cer.Next(nil)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		h := sha256.Sum256(ch.Data)
		fmt.Fprintf(hlistW, "%x\n", h)

		if err := saveChunk(casDir, h[:], ch.Data); err != nil {
			return err
		}
	}

	// save list of hashes in CAS too
	hlistHash := hlistHasher.Sum(nil)
	if err := saveChunk(casDir, hlistHash, hlistBuf.Bytes()); err != nil {
		return err
	}
	fmt.Fprintf(w, "%x\n", hlistHash)
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
