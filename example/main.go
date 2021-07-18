package main

import (
	"context"
	"fmt"
	"os"

	files "github.com/ipfs/go-ipfs-files"
	"github.com/web3-storage/go-w3s-client"
)

// Usage:
// TOKEN="API_TOKEN" go run ./main.go
func main() {
	c, _ := w3s.NewClient(
		w3s.WithEndpoint(os.Getenv("ENDPOINT")),
		w3s.WithToken(os.Getenv("TOKEN")),
	)

	file0, _ := os.Open("pinpie.jpg")
	file1, _ := os.Open("donotresist.jpg")

	dir := files.NewMapDirectory(map[string]files.Node{
		"pinpie.jpg":      files.NewReaderFile(file0),
		"donotresist.jpg": files.NewReaderFile(file1),
	})

	cid, _ := c.Put(context.Background(), dir)

	fmt.Printf("https://%s.ipfs.dweb.link\n", cid)
}
