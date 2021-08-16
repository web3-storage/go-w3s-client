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

	cid, err := c.Put(context.Background(), []fs.File{file0, file1})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("https://%v.ipfs.dweb.link\n", cid)

	s, err := c.Status(context.Background(), cid)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("%+v", s)
}
