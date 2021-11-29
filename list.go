package w3s

import (
	"context"
	"time"
)

type ListIterator struct {
	ctx        context.Context
	before     time.Time
	maxResults int
}

type listConfig struct {
	before     time.Time
	maxResults int
}

func (li *ListIterator) Next() (*Status, error) {

}

func (c *client) List(ctx context.Context, options ...ListOption) (*ListIterator, error) {
	var cfg listConfig
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	return &ListIterator{
		ctx:        ctx,
		before:     cfg.before,
		maxResults: cfg.maxResults,
	}, nil
}
