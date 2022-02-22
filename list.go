package w3s

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/tomnomnom/linkheader"
)

const maxPageSize = 100

type UploadIterator struct {
	paginator *pageIterator
	max       int
	count     int
	page      []*Status
}

// Next retrieves status information for the next upload in the list.
func (li *UploadIterator) Next() (*Status, error) {
	li.count++
	if li.max > 0 && li.count > li.max {
		return nil, io.EOF
	}
	if len(li.page) > 0 {
		item := li.page[0]
		li.page = li.page[1:]
		return item, nil
	}
	res, err := li.paginator.Next()
	if err != nil {
		return nil, err
	}
	var page []*Status
	d := json.NewDecoder(res.Body)
	err = d.Decode(&page)
	if err != nil {
		return nil, err
	}
	li.page = page
	if len(li.page) > 0 {
		item := li.page[0]
		li.page = li.page[1:]
		return item, nil
	}
	return nil, io.EOF
}

type listConfig struct {
	before     time.Time
	maxResults int
}

// List retrieves the list of uploads to Web3.Storage.
func (c *client) List(ctx context.Context, options ...ListOption) (*UploadIterator, error) {
	var cfg listConfig
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	fetchNextPage := func(url string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s%s", c.cfg.endpoint, url), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.cfg.token))
		req.Header.Add("Access-Control-Request-Headers", "Link")
		req.Header.Add("X-Client", clientName)
		res, err := c.cfg.hc.Do(req)
		if err != nil {
			return nil, err
		}
		if res.StatusCode != 200 {
			return nil, fmt.Errorf("unexpected response status: %d", res.StatusCode)
		}
		return res, nil
	}

	var before string
	if cfg.before.IsZero() {
		before = time.Now().Format(iso8601)
	} else {
		before = cfg.before.Format(iso8601)
	}

	size := cfg.maxResults
	if size > maxPageSize {
		size = maxPageSize
	}

	var urlPath string
	if size <= 0 {
		urlPath = fmt.Sprintf("/user/uploads?before=%s", url.QueryEscape(before))
	} else {
		urlPath = fmt.Sprintf("/user/uploads?before=%s&size=%d", url.QueryEscape(before), size)
	}

	return &UploadIterator{
		paginator: newPageIterator(urlPath, fetchNextPage),
		max:       cfg.maxResults,
	}, nil
}

type pageIterator struct {
	nextURL       string
	fetchNextPage func(string) (*http.Response, error)
}

func newPageIterator(url string, fetchNextPage func(string) (*http.Response, error)) *pageIterator {
	return &pageIterator{
		nextURL:       url,
		fetchNextPage: fetchNextPage,
	}
}

func (pi *pageIterator) Next() (*http.Response, error) {
	res, err := pi.fetchNextPage(pi.nextURL)
	if err != nil {
		return nil, err
	}
	linkHdrs := res.Header["Link"]
	if len(linkHdrs) == 0 {
		return nil, io.EOF
	}
	links := linkheader.Parse(linkHdrs[0])
	for _, l := range links {
		if l.Rel == "next" {
			pi.nextURL = links[0].URL
			return res, nil
		}
	}
	return nil, io.EOF
}
