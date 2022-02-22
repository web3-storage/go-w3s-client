package w3s

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ipfs/go-cid"
)

type PinResponse struct {
	RequestID string
	Status    string
	Created   time.Time
	Pin       PinResponseDetail
	Delegates []string
}

type pinResponseJson struct {
	RequestID string            `json:"requestId"`
	Status    string            `json:"status"`
	Created   string            `json:"created"`
	Pin       PinResponseDetail `json:"pin"`
	Delegates []string          `json:"delegates"`
}

func (p *PinResponse) UnmarshalJSON(b []byte) error {
	var raw pinResponseJson
	err := json.Unmarshal(b, &raw)
	if err != nil {
		return err
	}
	p.RequestID = raw.RequestID
	p.Status = raw.Status
	if raw.Created != "" {
		p.Created, err = time.Parse(iso8601, raw.Created)
		if err != nil {
			return err
		}
	}
	p.Pin = raw.Pin
	p.Delegates = raw.Delegates

	return nil
}

type PinResponseDetail struct {
	Cid        cid.Cid
	SourceCid  cid.Cid
	ContentCid cid.Cid
	Name       string
	Origins    []string
	Meta       map[string]string
	Deleted    time.Time
	Created    time.Time
	Updated    time.Time
	Pins       []Pin
}

type pinResponseDetailJson struct {
	Cid        string            `json:"cid"`
	SourceCid  string            `json:"sourceCid"`
	ContentCid string            `json:"contentCid"`
	Name       string            `json:"name"`
	Origins    []string          `json:"origins,omitempty"`
	Meta       map[string]string `json:"meta,omitempty"`
	Deleted    string            `json:"deleted,omitempty"`
	Created    string            `json:"created,omitempty"`
	Updated    string            `json:"updated,omitempty"`
	Pins       []Pin             `json:"pins,omitempty"`
}

func (p *PinResponseDetail) UnmarshalJSON(b []byte) error {
	var raw pinResponseDetailJson
	err := json.Unmarshal(b, &raw)
	if err != nil {
		return err
	}

	if raw.Cid != "" {
		p.Cid, err = cid.Parse(raw.Cid)
		if err != nil {
			return err
		}
	} else {
		p.Cid = cid.Undef
	}

	if raw.SourceCid != "" {
		p.SourceCid, err = cid.Parse(raw.SourceCid)
		if err != nil {
			return err
		}
	} else {
		p.SourceCid = cid.Undef
	}

	if raw.ContentCid != "" {
		p.ContentCid, err = cid.Parse(raw.ContentCid)
		if err != nil {
			return err
		}
	} else {
		p.ContentCid = cid.Undef
	}

	p.Name = raw.Name
	p.Origins = raw.Origins
	p.Meta = raw.Meta

	if raw.Deleted != "" {
		p.Deleted, err = time.Parse(iso8601, raw.Deleted)
		if err != nil {
			return err
		}
	}

	if raw.Created != "" {
		p.Created, err = time.Parse(iso8601, raw.Created)
		if err != nil {
			return err
		}
	}

	if raw.Updated != "" {
		p.Updated, err = time.Parse(iso8601, raw.Updated)
		if err != nil {
			return err
		}
	}

	p.Pins = raw.Pins

	return nil
}

type pinConfig struct {
	name    string
	origins []string
	meta    map[string]string
}

// Pin adds a new pin to Web3.Storage.
func (c *client) Pin(ctx context.Context, cid cid.Cid, options ...PinOption) (*PinResponse, error) {
	var cfg pinConfig
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	pinData := struct {
		Cid     string            `json:"cid"`
		Name    string            `json:"name,omitempty"`
		Origins []string          `json:"origins,omitempty"`
		Meta    map[string]string `json:"meta,omitempty"`
	}{
		Cid:     cid.String(),
		Name:    cfg.name,
		Origins: cfg.origins,
		Meta:    cfg.meta,
	}

	encoded := new(bytes.Buffer)
	err := json.NewEncoder(encoded).Encode(&pinData)
	if err != nil {
		return nil, fmt.Errorf("encode pin request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/pins", c.cfg.endpoint), encoded)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.cfg.token))
	req.Header.Add("X-Client", clientName)
	res, err := c.cfg.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send pin request: %w", err)
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected response status: %d", res.StatusCode)
	}
	defer res.Body.Close()

	d := json.NewDecoder(res.Body)

	var pr PinResponse
	err = d.Decode(&pr)
	if err != nil {
		return nil, fmt.Errorf("decode pin response: %w", err)
	}
	return &pr, nil
}
