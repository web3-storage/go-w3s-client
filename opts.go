package w3s

import (
	"fmt"
	"io/fs"
	"net/http"
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/multiformats/go-multiaddr"
)

// Option is an option configuring a web3.storage client.
type Option func(cfg *clientConfig) error

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

// WithHTTPClient sets the HTTP client to use when making requests which allows
// timeouts and redirect behaviour to be configured. The default is to use the
// DefaultClient from the Go standard library.
func WithHTTPClient(hc *http.Client) Option {
	return func(cfg *clientConfig) error {
		if hc != nil {
			cfg.hc = hc
		}
		return nil
	}
}

// PutOption is an option configuring a call to Put.
type PutOption func(cfg *putConfig) error

// WithFs sets the file system interface for use with file operations.
func WithFs(fsys fs.FS) PutOption {
	return func(cfg *putConfig) error {
		if fsys != nil {
			cfg.fsys = fsys
		}
		return nil
	}
}

// WithDirname sets the root directory path, for use when the provided file is a
// directory and does NOT implement fs.ReadDirFile. The default is "", which
// will resolve to the current working directory if the file system interface is
// the default (the OS).
func WithDirname(dirname string) PutOption {
	return func(cfg *putConfig) error {
		cfg.dirname = dirname
		return nil
	}
}

// ListOption is an option configuring a call to List.
type ListOption func(cfg *listConfig) error

// WithBefore sets the time that items in the list were uploaded before.
func WithBefore(before time.Time) ListOption {
	return func(cfg *listConfig) error {
		cfg.before = before
		return nil
	}
}

// WithMaxResults sets the maximum number of results that will be available from
// the iterator.
func WithMaxResults(maxResults int) ListOption {
	return func(cfg *listConfig) error {
		cfg.maxResults = maxResults
		return nil
	}
}

// PinOption is an option configuring a call to Pin.
type PinOption func(cfg *pinConfig) error

// WithPinName sets the name to use for the pinned data.
func WithPinName(name string) PinOption {
	return func(cfg *pinConfig) error {
		cfg.name = name
		return nil
	}
}

// WithPinOrigin adds a multiaddr known to provide the data.
func WithPinOrigin(ma string) PinOption {
	return func(cfg *pinConfig) error {
		_, err := multiaddr.NewMultiaddr(ma)
		if err != nil {
			return fmt.Errorf("origin: %w", err)
		}
		cfg.origins = append(cfg.origins, ma)
		return nil
	}
}

// WithPinMeta adds metadata about pinned data.
func WithPinMeta(key, value string) PinOption {
	return func(cfg *pinConfig) error {
		if cfg.meta == nil {
			cfg.meta = map[string]string{}
		}
		cfg.meta[key] = value
		return nil
	}
}
