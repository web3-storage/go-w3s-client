package w3s

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"

	bserv "github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	w3http "github.com/web3-storage/go-w3s-client/http"
)

const clientName = "web3.storage/go"

// Client is a HTTP API client to the web3.storage service.
type Client interface {
	Get(context.Context, cid.Cid) (*w3http.Web3Response, error)
	Put(context.Context, fs.File, ...PutOption) (cid.Cid, error)
	PutCar(context.Context, io.Reader) (cid.Cid, error)
	Status(context.Context, cid.Cid) (*Status, error)
}

type clientConfig struct {
	token    string
	endpoint string
	ds       ds.Batching
}

type client struct {
	cfg  *clientConfig
	bsvc bserv.BlockService
	hc   *http.Client
}

// NewClient creates a new web3.storage API client.
func NewClient(options ...Option) (Client, error) {
	cfg := clientConfig{
		endpoint: "https://api.web3.storage",
	}
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}
	if cfg.token == "" {
		return nil, fmt.Errorf("missing auth token")
	}
	c := client{cfg: &cfg, hc: &http.Client{}}
	if cfg.ds != nil {
		c.bsvc = bserv.New(blockstore.NewBlockstore(cfg.ds), nil)
	} else {
		ds := dssync.MutexWrap(ds.NewMapDatastore())
		c.bsvc = bserv.New(blockstore.NewBlockstore(ds), nil)
	}
	return &c, nil
}
