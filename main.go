package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

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

// prepare takes a binary file and saves a list of chunk hashes, one per line,
// while saving chunks in CAS.
// The resulting hash list is then also stored in the CAS and its key (its hash)
// is printed.
func prepare(w io.Writer, rd io.Reader, cas CAS) error {
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

	cas := &dirCAS{dir: *casDir}

	var err error
	switch cmd := flag.Arg(0); cmd {
	case "prepare":
		err = prepare(out, in, cas)
	default:
		err = fmt.Errorf("unknown command %q", cmd)
	}
	if err != nil {
		log.Fatal(err)
	}
}
