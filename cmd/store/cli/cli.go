package main

// Simple client for using the Store interface on the command line.

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"upspin.googlesource.com/upspin.git/access"
	"upspin.googlesource.com/upspin.git/store/gcp"
	"upspin.googlesource.com/upspin.git/upspin"
)

var (
	Store upspin.Store = newStore()

	inFile  = flag.String("in", "", "input file")
	outFile = flag.String("out", "", "output file")
)

func main() {
	flag.Usage = Usage
	flag.Parse()

	if len(flag.Args()) < 1 {
		Usage()
	}

	switch strings.ToLower(flag.Arg(0)) {
	case "get":
		if len(flag.Args()) < 2 {
			Usage()
		}
		get(flag.Arg(1))
	case "put":
		if len(flag.Args()) > 1 {
			Usage()
		}
		put()
	default:
		log.Println("Can't understand command. Use GET or PUT")
		Usage()
	}
}

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\tcli [flags] <GET|PUT> [<ref>]\n")
	fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func newStore() upspin.Store {
	context := store.Context{
		ServerURL: "http://localhost:8080",
		Client:    &http.Client{},
	}
	e := upspin.Endpoint{
		Transport: upspin.GCP,
	}
	s, err := access.BindStore(context, e)
	if err != nil {
		log.Fatalf("Can't bind: %v", err)
	}
	return s
}

func get(key string) {
	buf, _, err := innerGet(key, 0)

	if err != nil {
		log.Fatal(err)
	}
	// Write to outfile or to stdout if none set
	var output *os.File
	if *outFile == "" {
		output = os.Stdout
	} else {
		output, err = os.Create(*outFile)
		if err != nil {
			log.Fatal(err)
		}
		defer output.Close()
	}
	_, err = io.Copy(output, bytes.NewReader(buf))
	if err != nil {
		log.Fatalf("Copying to output failed: %v", err)
	}
}

func innerGet(key string, count int) ([]byte, []upspin.Location, error) {
	if count > 3 {
		return nil, nil, errors.New("Too many redirections")
	}
	buf, locs, err := Store.Get(key)
	if err != nil {
		log.Fatalf("Error getting from server: %v", err)
	}
	if locs != nil {
		log.Printf("We got redirected. Following new location: %v", locs[0])
		buf, locs, err = innerGet(locs[0].Reference.Key, count+1)
	}
	return buf, locs, err
}

func put() {
	var input *os.File
	var err error
	if *inFile == "" {
		input = os.Stdin
	} else {
		input, err = os.Open(*inFile)
		if err != nil {
			log.Fatal(err)
		}
		defer input.Close()
	}

	data, err := ioutil.ReadAll(input)
	if err != nil {
		log.Fatal(err)
	}

	key, err := Store.Put(data)
	if err != nil {
		log.Fatalf("Error putting to server: %v", err)
	}
	log.Printf("Put file to storage. Key: %v", key)
}