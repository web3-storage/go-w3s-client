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
    "github.com/web3-storage/go-w3s-client"
)

func main() {
    c := w3s.NewClient(w3s.WithToken("<AUTH_TOKEN>"))
    // WIP
}
```

## API

[pkg.go.dev Reference](https://pkg.go.dev/github.com/web3-storage/go-w3s-client)

## Contribute

Feel free to dive in! [Open an issue](https://github.com/web3-storage/go-w3s-client/issues/new) or submit PRs.

## License

Dual-licensed under [MIT + Apache 2.0](https://github.com/web3-storage/go-w3s-client/blob/main/LICENSE.md)
