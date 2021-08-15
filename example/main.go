package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"

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

	cid, _ := c.Put(context.Background(), []fs.File{file0, file1})

	fmt.Printf("https://%s.ipfs.dweb.link\n", cid)
}
