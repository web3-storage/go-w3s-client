package main

import (
	"context"
	"fmt"
	"os"

	"github.com/web3-storage/go-w3s-client"
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

	file, err := os.Open("images")
	if err != nil {
		panic(err)
	}
	cid, err := c.Put(context.Background(), file)
	if err != nil {
		panic(err)
	}
	fmt.Printf("https://%v.ipfs.dweb.link\n", cid)

	s, err := c.Status(context.Background(), cid)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Status: %+v", s)

	// Get status of an existing CID:
	// cid, _ := cid.Parse("bafybeig7qnlzyregxe2m63b4kkpx3ujqm5bwmn5wtvtftp7j27tmdtznji")
	// s, err := c.Status(context.Background(), cid)
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Printf("Status: %+v", s)
}
