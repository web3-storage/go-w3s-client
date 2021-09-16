package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"

	"github.com/ipfs/go-cid"
	"github.com/web3-storage/go-w3s-client"
	w3fs "github.com/web3-storage/go-w3s-client/fs"
)

// Usage:
// TOKEN="API_TOKEN" go run ./main.go
func main() {
	c, err := w3s.NewClient(
		w3s.WithEndpoint(os.Getenv("ENDPOINT")),
		w3s.WithToken(os.Getenv("TOKEN")),
	)
	if err != nil {
		panic(err)
	}

	// cid := putSingleFile(c)
	// getStatusForCid(c, cid)

	getFiles(c)
}

func putSingleFile(c w3s.Client) cid.Cid {
	file, err := os.Open("images/donotresist.jpg")
	if err != nil {
		panic(err)
	}
	return putFile(c, file)
}

func putMultipleFiles(c w3s.Client) cid.Cid {
	f0, err := os.Open("images/donotresist.jpg")
	if err != nil {
		panic(err)
	}
	f1, err := os.Open("images/pinpie.jpg")
	if err != nil {
		panic(err)
	}
	dir := w3fs.NewDir("comic", []fs.File{f0, f1})
	return putFile(c, dir)
}

func putMultipleFilesAndDirectories(c w3s.Client) cid.Cid {
	f0, err := os.Open("images/donotresist.jpg")
	if err != nil {
		panic(err)
	}
	f1, err := os.Open("images/pinpie.jpg")
	if err != nil {
		panic(err)
	}
	d0 := w3fs.NewDir("one", []fs.File{f0})
	d1 := w3fs.NewDir("two", []fs.File{f1})
	rootdir := w3fs.NewDir("comic", []fs.File{d0, d1})
	return putFile(c, rootdir)
}

func putDirectory(c w3s.Client) cid.Cid {
	dir, err := os.Open("images")
	if err != nil {
		panic(err)
	}
	return putFile(c, dir)
}

func putFile(c w3s.Client, f fs.File, opts ...w3s.PutOption) cid.Cid {
	cid, err := c.Put(context.Background(), f, opts...)
	if err != nil {
		panic(err)
	}
	fmt.Printf("https://%v.ipfs.dweb.link\n", cid)
	return cid
}

func getStatusForCid(c w3s.Client, cid cid.Cid) {
	s, err := c.Status(context.Background(), cid)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Status: %+v", s)
}

func getStatusForKnownCid(c w3s.Client) {
	cid, _ := cid.Parse("bafybeig7qnlzyregxe2m63b4kkpx3ujqm5bwmn5wtvtftp7j27tmdtznji")
	getStatusForCid(c, cid)
}

func getFiles(c w3s.Client) {
	cid, _ := cid.Parse("bafybeide43vps6vt2oo7nbqfwn5zz6l2alyi64mym3sb7reqhmypjnmej4")

	res, err := c.Get(context.Background(), cid)
	if err != nil {
		panic(err)
	}

	f, fsys, err := res.Files()
	if err != nil {
		panic(err)
	}

	info, err := f.Stat()
	if err != nil {
		panic(err)
	}

	if info.IsDir() {
		err = fs.WalkDir(fsys, "/", func(path string, d fs.DirEntry, err error) error {
			info, _ := d.Info()
			fmt.Printf("%s (%d bytes)\n", path, info.Size())
			return err
		})
		if err != nil {
			panic(err)
		}
	} else {
		fmt.Printf("%s (%d bytes)\n", cid.String(), info.Size())
	}
}
