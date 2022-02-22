package w3s

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"

	"github.com/alanshaw/go-carbites"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-merkledag"
	"github.com/ipld/go-car"
	"github.com/web3-storage/go-w3s-client/adder"
)

const targetChunkSize = 1024 * 1024 * 10

type putConfig struct {
	fsys    fs.FS
	dirname string
}

// Put uploads files to Web3.Storage. The file argument can be a single file or
// a directory. If a directory is passed and the directory does NOT implement
// fs.ReadDirFile then the WithDirname option should be passed (or the current
// process working directory will be used).
func (c *client) Put(ctx context.Context, file fs.File, options ...PutOption) (cid.Cid, error) {
	var cfg putConfig
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return cid.Undef, err
		}
	}

	info, err := file.Stat()
	if err != nil {
		return cid.Undef, err
	}

	dag := merkledag.NewDAGService(c.bsvc)
	dagFmtr, err := adder.NewAdder(ctx, dag)
	if err != nil {
		return cid.Undef, err
	}

	root, err := dagFmtr.Add(file, cfg.dirname, cfg.fsys)
	if err != nil {
		return cid.Undef, err
	}

	// If file is a dir, do not wrap in another.
	if info.IsDir() {
		mr, err := dagFmtr.MfsRoot()
		if err != nil {
			return cid.Undef, err
		}
		rdir := mr.GetDirectory()
		cdir, err := rdir.Child(info.Name())
		if err != nil {
			return cid.Undef, err
		}
		cnode, err := cdir.GetNode()
		if err != nil {
			return cid.Undef, err
		}
		root = cnode.Cid()
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
	spltr, err := carbites.Split(car, targetChunkSize, carbites.Treewalk)
	if err != nil {
		return cid.Undef, err
	}

	var root cid.Cid
	for {
		r, err := spltr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return cid.Undef, err
		}

		// TODO: concurrency
		c, err := c.sendCar(ctx, r)
		if err != nil {
			return cid.Undef, err
		}
		root = c
	}

	return root, nil
}

// TODO: retry
func (c *client) sendCar(ctx context.Context, r io.Reader) (cid.Cid, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.cfg.endpoint+"/car", r)
	if err != nil {
		return cid.Undef, err
	}
	req.Header.Add("Content-Type", "application/car")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.cfg.token))
	req.Header.Add("X-Client", clientName)
	res, err := c.cfg.hc.Do(req)
	if err != nil {
		return cid.Undef, err
	}
	if res.StatusCode != 200 {
		return cid.Undef, fmt.Errorf("unexpected response status: %d", res.StatusCode)
	}
	d := json.NewDecoder(res.Body)
	var out struct {
		Cid string `json:"cid"`
	}
	err = d.Decode(&out)
	if err != nil {
		return cid.Undef, err
	}
	return cid.Parse(out.Cid)
}
