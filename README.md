# go-w3s-client

⚠️ WIP. A client to the web3.storage API.

## Install

```sh
go get github.com/web3-storage/go-w3s-client
```

## Usage

```go
package main

import (
    "os"
    "github.com/web3-storage/go-w3s-client"
)

func main() {
    c, _ := w3s.NewClient(w3s.WithToken("<AUTH_TOKEN>"))
    f, _ := os.Open("images/pinpie.jpg")
    cid, _ := c.Put(context.Background(), f)
	fmt.Printf("https://%v.ipfs.dweb.link\n", cid)
}
```

See [example](./example) for more.

## API

[pkg.go.dev Reference](https://pkg.go.dev/github.com/web3-storage/go-w3s-client)

## Contribute

Feel free to dive in! [Open an issue](https://github.com/web3-storage/go-w3s-client/issues/new) or submit PRs.

## License

Dual-licensed under [MIT + Apache 2.0](https://github.com/web3-storage/go-w3s-client/blob/main/LICENSE.md)
