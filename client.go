package w3s

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/alanshaw/go-carbites"
	"github.com/ipfs/go-cid"
)

// Option is an option configuring a web3.storage client.
type Option func(cfg *clientConfig) error

// Client is a HTTP API client to the web3.storage service.
type Client interface {
	Get(cid.Cid) (GetResponse, error)
	Put(files []os.File) (cid.Cid, error)
	Status(cid.Cid) (Status, error)
}

// GetResponse is a response to a call to the Get method.
type GetResponse interface {
	Files() []os.File
}

// Status is pin and deal status for a given CID.
type Status interface{}

type clientConfig struct {
	token    string
	endpoint string
}

type client struct {
	cfg *clientConfig
}

// WithEndpoint sets the URL of the root API when making requests (default
// https://api.web3.storage).
func WithEndpoint(token string) Option {
	return func(cfg *clientConfig) error {
		cfg.token = token
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
	// JWT token is currently required, but maybe not forever.
	if cfg.token == "" {
		return nil, fmt.Errorf("missing auth token")
	}
	return &client{&cfg}, nil
}

func (c *client) Get(cid cid.Cid) (GetResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *client) Put(files []os.File) (cid.Cid, error) {

	// TODO: import files

	carChunks := make(chan io.Reader)
	var wg sync.WaitGroup
	wg.Add(1)

	root := cid.Undef

	go func() {
		defer wg.Done()
		for {
			select {
			case r := <-carChunks:
				// TODO: send request, read response and assign to root
			}
		}
	}()

	targetSize := 1024 * 1024 * 10
	strategy := carbites.Treewalk
	err := carbites.Split(context.Background(), reader, targetSize, strategy, carChunks)
	if err != nil {
		return cid.Undef, err
	}
	wg.Wait()

	return root, nil
}

func (c *client) Status(cid cid.Cid) (Status, error) {
	return nil, fmt.Errorf("not implemented")
}
