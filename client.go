package w3s

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"sync"
	"time"

	"github.com/alanshaw/go-carbites"
	bserv "github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	ipfsfiles "github.com/ipfs/go-ipfs-files"
	ipld "github.com/ipfs/go-ipld-format"
	dag "github.com/ipfs/go-merkledag"
	"github.com/ipfs/ipfs-cluster/adder"
	"github.com/ipfs/ipfs-cluster/api"
	car "github.com/ipld/go-car"
)

const targetChunkSize = 1024 * 1024 * 10

// Client is a HTTP API client to the web3.storage service.
type Client interface {
	Get(context.Context, cid.Cid) (GetResponse, error)
	Put(context.Context, []fs.File, ...PutOption) (cid.Cid, error)
	PutCar(context.Context, io.Reader) (cid.Cid, error)
	Status(context.Context, cid.Cid) (*Status, error)
}

// GetResponse is a response to a call to the Get method.
type GetResponse interface {
	Files() []fs.File
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
	PeerID   string    `json:"peerId"`
	PeerName string    `json:"peerName"`
	Region   string    `json:"region"`
	Status   PinStatus `json:"status"`
	Updated  time.Time `json:"updated"`
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
	DealID            uint64     `json:"dealId"`
	StorageProvider   string     `json:"storageProvider"`
	Status            DealStatus `json:"status"`
	PieceCid          cid.Cid    `json:"pieceCid"`
	DataCid           cid.Cid    `json:"dataCid"`
	DataModelSelector string     `json:"dataModelSelector"`
	Activation        time.Time  `json:"activation"`
	Created           time.Time  `json:"created"`
	Updated           time.Time  `json:"updated"`
}

// Status is IPFS pin and Filecoin deal status for a given CID.
type Status struct {
	Cid     cid.Cid `json:"cid"`
	DagSize uint64  `json:"dagSize"`
	Created string  `json:"created"`
	Pins    []Pin   `json:"pins"`
	Deals   []Deal  `json:"deals"`
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
func (c *client) sendCar(ctx context.Context, r io.Reader) (cid.Cid, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.cfg.endpoint+"/car", r)
	if err != nil {
		return cid.Undef, err
	}
	req.Header.Add("Content-Type", "application/car")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.cfg.token))
	res, err := c.hc.Do(req)
	if err != nil {
		return cid.Undef, err
	}
	if res.StatusCode != 200 {
		return cid.Undef, fmt.Errorf("unexpected response status: %d", res.StatusCode)
	}
	d := json.NewDecoder(res.Body)
	var out struct {
		Cid cid.Cid `json:"cid"`
	}
	err = d.Decode(&out)
	if err != nil {
		return cid.Undef, err
	}
	return out.Cid, nil
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

func convertToIpfsDirectory(files []fs.File, fsys fs.FS) (ipfsfiles.Directory, error) {
	var entries []ipfsfiles.DirEntry
	for _, f := range files {
		info, err := f.Stat()
		if err != nil {
			return nil, err
		}
		n, err := NewFsNode("", f, info, fsys)
		if err != nil {
			return nil, err
		}
		entries = append(entries, ipfsfiles.FileEntry(info.Name(), n))
	}
	return ipfsfiles.NewSliceDirectory(entries), nil
}

type putConfig struct {
	fsys fs.FS
}

// Put uploads files to Web3.Storage.
func (c *client) Put(ctx context.Context, files []fs.File, options ...PutOption) (cid.Cid, error) {
	var cfg putConfig
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return cid.Undef, err
		}
	}

	dag := c.dag
	if dag == nil {
		dag = c.newMemDag()
	}

	params := api.DefaultAddParams()
	params.Chunker = "size-1048576"
	// TODO: Maxlinks: 1024
	params.CidVersion = 1
	params.RawLeaves = true
	params.Wrap = true

	// If only 1 file, and that file is a dir, do not wrap in another.
	if len(files) == 1 {
		info, err := files[0].Stat()
		if err != nil {
			return cid.Undef, err
		}
		if info.IsDir() {
			params.Wrap = false
		}
	}

	a := adder.New(&clusterDagService{dag}, params, nil)

	dir, err := convertToIpfsDirectory(files, cfg.fsys)
	if err != nil {
		return cid.Undef, err
	}

	root, err := a.FromFiles(ctx, dir)
	if err != nil {
		return cid.Undef, err
	}

	carReader, carWriter := io.Pipe()

	go func() {
		err = car.WriteCar(ctx, dag, []cid.Cid{root}, carWriter)
		if err != nil {
			carWriter.CloseWithError(err)
			return
		}
		carWriter.Close()
	}()

	return c.PutCar(ctx, carReader)
}

// PutCar uploads a CAR (Content Addressable Archive) to Web3.Storage.
func (c *client) PutCar(ctx context.Context, car io.Reader) (cid.Cid, error) {
	carChunks := make(chan io.Reader)

	var root cid.Cid
	var wg sync.WaitGroup
	wg.Add(1)

	var sendErr error
	go func() {
		defer wg.Done()
		for r := range carChunks {
			// TODO: concurrency
			c, err := c.sendCar(ctx, r)
			if err != nil {
				sendErr = err
				break
			}
			root = c
		}
	}()

	err := carbites.Split(ctx, car, targetChunkSize, carbites.Treewalk, carChunks)
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
	d := json.NewDecoder(res.Body)
	var s Status
	err = d.Decode(&s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
