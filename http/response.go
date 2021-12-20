package http

import (
	"context"
	"io"
	"io/fs"
	"net/http"

	"github.com/ipfs/go-blockservice"
	"github.com/ipld/go-car"
	"github.com/web3-storage/go-w3s-client/fs/adapter"
)

// Web3Response is a response to a call to the Get method.
type Web3Response struct {
	*http.Response
	bsvc blockservice.BlockService
}

func NewWeb3Response(r *http.Response, bsvc blockservice.BlockService) *Web3Response {
	return &Web3Response{r, bsvc}
}

// Files consumes the HTTP response and returns the root file (which may be a
// directory). You can use the returned FileSystem implementation to read
// nested files and directories if the returned file is a directory.
func (r *Web3Response) Files() (fs.File, fs.FS, error) {
	cr, err := car.NewCarReader(r.Body)
	if err != nil {
		return nil, nil, err
	}

	for {
		b, err := cr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, nil, err
		}
		err = r.bsvc.AddBlock(context.Background(), b)
		if err != nil {
			return nil, nil, err
		}
	}

	ctx := r.Request.Context()
	rootCid := cr.Header.Roots[0]

	fs, err := adapter.NewFsWithContext(ctx, rootCid, r.bsvc)
	if err != nil {
		return nil, nil, err
	}

	f, err := fs.Open("/")
	if err != nil {
		return nil, nil, err
	}

	return f, fs, nil
}
