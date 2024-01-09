> ⚠️ **DEPRECATED** This repo will be archived on January 9, 2024 as this web3.storage API will no longer take new uploads. Please use the [new client and API](https://github.com/web3-storage/go-w3up) for future usage of web3.storage. Documentation for the new client can be found [here](https://web3.storage/docs). You can learn more about these changes [here](https://blog.web3.storage/posts/the-data-layer-is-here-with-the-new-web3-storage).

# go-w3s-client

A client to the Web3.Storage API.

Demo: https://youtu.be/FLsQZ_ogeOg

## Install

```sh
go get github.com/web3-storage/go-w3s-client
```

## Usage

```go
package main

import (
    "io/fs"
    "os"
    "github.com/web3-storage/go-w3s-client"
)

func main() {
    c, _ := w3s.NewClient(w3s.WithToken("<AUTH_TOKEN>"))
    f, _ := os.Open("images/pinpie.jpg")

    // OR add a whole directory:
    //
    //   f, _ := os.Open("images")
    //
    // OR create your own directory:
    //
    //   img0, _ := os.Open("aliens.jpg")
    //   img1, _ := os.Open("donotresist.jpg")
    //   f := w3fs.NewDir("images", []fs.File{img0, img1})

    // Write a file/directory
    cid, _ := c.Put(context.Background(), f)
    fmt.Printf("https://%v.ipfs.dweb.link\n", cid)

    // Retrieve a file/directory
    res, _ := c.Get(context.Background(), cid)
    
    // res is a http.Response with an extra method for reading IPFS UnixFS files!
    f, fsys, _ := res.Files()

    // List directory entries
    if d, ok := f.(fs.ReadDirFile); ok {
        ents, _ := d.ReadDir(0)
        for _, ent := range ents {
            fmt.Println(ent.Name())
        }
    }

    // Walk whole directory contents (including nested directories)
    fs.WalkDir(fsys, "/", func(path string, d fs.DirEntry, err error) error {
        info, _ := d.Info()
        fmt.Printf("%s (%d bytes)\n", path, info.Size())
        return err
    })

    // Open a file in a directory
    img, _ := fsys.Open("pinpie.jpg")
    // img.Stat()
    // img.Read(...)
    // img.Close()
}
```

See [example](./example) for more.

## API

[pkg.go.dev Reference](https://pkg.go.dev/github.com/web3-storage/go-w3s-client)

## Contribute

Feel free to dive in! [Open an issue](https://github.com/web3-storage/go-w3s-client/issues/new) or submit PRs.

## License

Dual-licensed under [MIT + Apache 2.0](https://github.com/web3-storage/go-w3s-client/blob/main/LICENSE.md)
