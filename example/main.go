package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"

	"github.com/ipfs/go-cid"
	"github.com/web3-storage/go-w3s-client"
)

// Usage:
// TOKEN="API_TOKEN" go run ./main.go
func main() {
	c, _ := w3s.NewClient(
		w3s.WithEndpoint(os.Getenv("ENDPOINT")),
		w3s.WithToken(os.Getenv("TOKEN")),
	)

	cid := put(c)
	status(c, cid)

	// cid, _ := cid.Parse("bafybeig7qnlzyregxe2m63b4kkpx3ujqm5bwmn5wtvtftp7j27tmdtznji")
	// status(c, cid)
}

func put(c w3s.Client) cid.Cid {
	file0, _ := os.Open("pinpie.jpg")
	file1, _ := os.Open("donotresist.jpg")

	cid, err := c.Put(context.Background(), []fs.File{file0, file1})
	if err != nil {
		panic(err)
	}

	fmt.Printf("https://%v.ipfs.dweb.link\n", cid)
	return cid
}

func status(c w3s.Client, cid cid.Cid) w3s.Status {
	s, err := c.Status(context.Background(), cid)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Status: %+v", s)
	return s
}
