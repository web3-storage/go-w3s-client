package w3s

import (
	"io/fs"

	ds "github.com/ipfs/go-datastore"
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
