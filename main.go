package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
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
	casDir  = flag.String("cas-dir", "cas", "directory to store chunks")
	listen  = flag.String("listen", ":8080", "listen address")
	casAddr = flag.String("cas-addr", "http://localhost:8080", "url to a cas (e.g. imsy serve) instance")
	output  = flag.String("o", "", "output file")
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

		h, err := cas.Store(ch.Data)
		if err != nil {
			return err
		}
		fmt.Fprintln(&hlistBuf, h)
	}

	lh, err := cas.Store(hlistBuf.Bytes())
	if err != nil {
		return err
	}
	fmt.Fprintln(w, lh)
	return nil
}

func serve(listen string, cas CAS) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		h := r.URL.Path[1:]
		if err := cas.Copy(w, h); err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "not found", http.StatusNotFound)
			}
		}
		log.Printf("fetched %q", h)
	})
	log.Fatal(http.ListenAndServe(listen, nil))
	return nil
}

// pull fetches h from the CAS and interprets it as a list of hashes.
// It then fetches each object from the CAS keyed by those hashes and appends
// their content to outfile.
func pull(h string, outfile string, cas CASReader) error {
	var buf bytes.Buffer
	if err := cas.Copy(&buf, h); err != nil {
		return err
	}

	f, err := os.Create(outfile)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		h := scanner.Text()
		if err := cas.Copy(f, h); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func main() {
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	out := os.Stdout
	in := os.Stdin

	os.MkdirAll(*casDir, 0777)
	cas := &dirCAS{dir: *casDir}

	var err error
	switch cmd := flag.Arg(0); cmd {
	case "prepare":
		err = prepare(out, in, cas)
	case "serve":
		err = serve(*listen, cas)
	case "pull":
		if flag.NArg() < 2 || *output == "" {
			flag.Usage()
			os.Exit(1)
		}
		err = pull(flag.Arg(1), *output, chainedCASReader{cas, &httpCAS{addr: *casAddr}})
	default:
		err = fmt.Errorf("unknown command %q", cmd)
	}
	if err != nil {
		log.Fatal(err)
	}
}
