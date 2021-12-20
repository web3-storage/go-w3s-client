package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ipfs/go-cid"
	"github.com/web3-storage/go-w3s-client"
)

func main() {
	err := mainRet()
	if err != nil {
		fmt.Fprint(os.Stderr, "Error: "+err.Error()+"\n")
		os.Exit(1)
	}
	os.Exit(0)
}

type options struct {
	help    bool
	car     bool
	noStdin bool
	quiet   bool
}

func mainRet() error {
	var ctx context.Context
	{
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(context.Background())
		defer cancel()
		// Dealing with canceling on SIGTERM
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			select {
			case <-c:
				cancel()
			case <-ctx.Done():
			}
		}()
	}

	var opts options
	flag.BoolVar(&opts.help, "help", false, "Shows the help message.")
	flag.BoolVar(&opts.car, "car", false, "Upload a CAR archive as is rather than as a file.")
	flag.BoolVar(&opts.noStdin, "no-stdin", false, "Disable understanding the \"-\" path as /dev/stdin and understand it as a file named -.")
	flag.BoolVar(&opts.quiet, "quiet", false, "Only outputs the resulting hash (script friendly option).")
	flag.Parse()

	if opts.help {
		fmt.Fprintln(os.Stderr, os.Args[0]+`: Upload some files or a directories to web3.storage.
It deals with spliting CAR archives into many 100Mib CARs.

Environment variables:
WEB3_STORAGE_KEY = <web3.storage api key>

Positionals:
<file or directory to add>

Flags:`)
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, `
Example:
WEB3_STORAGE_KEY="eyJhbGc...TSyYE" `+os.Args[0]+` someFiles/`)
		return nil
	}

	// flag sanity part:
	key := os.Getenv("WEB3_STORAGE_KEY")
	if len(key) == 0 {
		return errors.New("empty WEB3_STORAGE_KEY environment variable")
	}

	var path string
	{
		paths := flag.Args()
		if len(paths) != 1 {
			if opts.noStdin && len(paths) == 0 {
				fmt.Fprintln(os.Stderr, "No path specified reading from stdin.")
				path = "-"
			} else {
				return errors.New("expected one file path")
			}
		} else {
			path = paths[0]
		}
	}

	// Starting to add:
	c, err := w3s.NewClient(w3s.WithToken(key))
	if err != nil {
		return fmt.Errorf("failing to create client: %e", err)
	}

	// Override the `-` name to stdin
	var f *os.File
	if !opts.noStdin && path == "-" {
		f = os.Stdin
	} else {
		f, err = os.Open(filepath.Clean(path))
		if err != nil {
			return fmt.Errorf("failing to open path: %e", err)
		}
	}

	var result cid.Cid
	if opts.car {
		// Adding a CAR
		result, err = c.PutCar(ctx, f)
		if err != nil {
			return fmt.Errorf("failing to upload CAR: %e", err)
		}
	} else {
		// Adding a file / directory
		result, err = c.Put(ctx, f)
		if err != nil {
			return fmt.Errorf("failing upload file: %e", err)
		}
	}

	r := result.String()
	if !opts.quiet {
		r = "Sucessfully added: " + r
	}
	fmt.Fprintln(os.Stdout, r)
	return nil
}
