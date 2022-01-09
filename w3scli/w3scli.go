package w3scli

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


	c, err := w3s.NewClient(
		w3s.WithEndpoint(os.Getenv("ENDPOINT")),
		w3s.WithToken(os.Getenv("TOKEN")),
	)
	if err != nil {
		panic(err)
	}

	arg := os.Args[1]
	file, err := os.Open(arg)
	if err != nil {
		panic(err)
	}
	
	putFile(c, file)

}


func putFile(c w3s.Client, f fs.File, opts ...w3s.PutOption) cid.Cid {
	cid, err := c.Put(context.Background(), f, opts...)
	if err != nil {
		panic(err)
	}
	fmt.Printf("https://%v.ipfs.dweb.link\n", cid)
	return cid
}
