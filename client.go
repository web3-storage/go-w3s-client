package w3s

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/alanshaw/go-carbites"
	bserv "github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	files "github.com/ipfs/go-ipfs-files"
	ipld "github.com/ipfs/go-ipld-format"
	dag "github.com/ipfs/go-merkledag"
	"github.com/ipfs/ipfs-cluster/adder"
	"github.com/ipfs/ipfs-cluster/api"
	car "github.com/ipld/go-car"
)

const targetChunkSize = 1024 * 1024 * 10

// Option is an option configuring a web3.storage client.
type Option func(cfg *clientConfig) error

// Client is a HTTP API client to the web3.storage service.
type Client interface {
	Get(context.Context, cid.Cid) (GetResponse, error)
	Put(context.Context, files.Directory) (cid.Cid, error)
	Status(context.Context, cid.Cid) (*Status, error)
}

// GetResponse is a response to a call to the Get method.
type GetResponse interface {
	Files() []os.File
}

type PinStatus int

const (
	PinStatusPinned    = PinStatus(api.TrackerStatusPinned)
	PinStatusPinning   = PinStatus(api.TrackerStatusPinning)
	PinStatusPinQueued = PinStatus(api.TrackerStatusPinQueued)
)

func (s PinStatus) String() string {
	return api.TrackerStatus(s).String()
}

type Pin struct {
	peerID   string
	peerName string
	region   string
	status   PinStatus
	updated  time.Time
}

type DealStatus int

const (
	DealStatusQueued DealStatus = iota
	DealStatusPublished
	DealStatusActive
)

func (s DealStatus) String() string {
	return []string{"Queued", "Published", "Active"}[s]
}

type Deal struct {
	dealID            uint64
	miner             string
	status            DealStatus
	pieceCid          cid.Cid
	dataCid           cid.Cid
	dataModelSelector string
	activation        time.Time
	created           time.Time
	updated           time.Time
}

// Status is IPFS pin and Filecoin deal status for a given CID.
type Status struct {
	cid     string
	dagSize uint64
	created string
	pins    []Pin
	deals   []Deal
}

type clientConfig struct {
	token    string
	endpoint string
	ds       ds.Batching
}

type client struct {
	cfg *clientConfig
	dag ipld.DAGService
	hc  *http.Client
}

// WithEndpoint sets the URL of the root API when making requests (default
// https://api.web3.storage).
func WithEndpoint(endpoint string) Option {
	return func(cfg *clientConfig) error {
		if endpoint != "" {
			cfg.endpoint = endpoint
		}
		return nil
	}
}

// WithToken sets the auth token to use in the Authorization header when making
// requests to the API.
func WithToken(token string) Option {
	return func(cfg *clientConfig) error {
		cfg.token = token
		return nil
	}
}

// WithDatastore sets the underlying datastore to use when reading or writing
// DAG block data. The default is to use a new in-memory store per Get/Put
// request.
func WithDatastore(ds ds.Batching) Option {
	return func(cfg *clientConfig) error {
		if ds != nil {
			cfg.ds = ds
		}
		return nil
	}
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
		bs := bserv.New(blockstore.NewBlockstore(cfg.ds), nil)
		c.dag = dag.NewDAGService(bs)
	}
	return &c, nil
}

func (c *client) newMemDag() ipld.DAGService {
	ds := dssync.MutexWrap(ds.NewMapDatastore())
	bs := bserv.New(blockstore.NewBlockstore(ds), nil)
	return dag.NewDAGService(bs)
}

// TODO: retry
func (c *client) sendCar(r io.Reader) error {
	req, err := http.NewRequest("POST", c.cfg.endpoint+"/car", r)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/car")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.cfg.token))
	res, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("unexpected response status: %d", res.StatusCode)
	}
	return nil
}

func (c *client) Get(ctx context.Context, cid cid.Cid) (GetResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

type clusterDagService struct {
	ipld.DAGService
}

func (clusterDagService) Finalize(ctx context.Context, cid cid.Cid) (cid.Cid, error) {
	return cid, nil
}

func (c *client) Put(ctx context.Context, dir files.Directory) (cid.Cid, error) {
	dag := c.dag
	if dag == nil {
		dag = c.newMemDag()
	}

	params := api.DefaultAddParams()
	params.CidVersion = 1
	params.RawLeaves = true
	params.Wrap = true

	a := adder.New(&clusterDagService{dag}, params, nil)
	root, err := a.FromFiles(ctx, dir)
	if err != nil {
		return cid.Undef, err
	}

	carReader, carWriter := io.Pipe()
	carChunks := make(chan io.Reader)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		err = car.WriteCar(ctx, dag, []cid.Cid{root}, carWriter)
		if err != nil {
			carWriter.CloseWithError(err)
			return
		}
		carWriter.Close()
	}()

	var sendErr error
	go func() {
		defer wg.Done()
		for r := range carChunks {
			// TODO: concurrency
			err := c.sendCar(r)
			if err != nil {
				sendErr = err
				break
			}
		}
	}()

	err = carbites.Split(ctx, carReader, targetChunkSize, carbites.Treewalk, carChunks)
	if err != nil {
		return cid.Undef, err
	}
	wg.Wait()

	return root, sendErr
}

func (c *client) Status(ctx context.Context, cid cid.Cid) (*Status, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/status/%s", c.cfg.endpoint, cid), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.cfg.token))
	res, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected response status: %d", res.StatusCode)
	}
}
