package w3s

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/ipfs-cluster/api"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

const iso8601 = "2006-01-02T15:04:05.999Z07:00"

type PinStatus int

const (
	PinStatusPinned    = PinStatus(api.TrackerStatusPinned)
	PinStatusPinning   = PinStatus(api.TrackerStatusPinning)
	PinStatusPinQueued = PinStatus(api.TrackerStatusPinQueued)
	PinStatusRemote    = PinStatus(api.TrackerStatusRemote)
	PinStatusUnpinned  = PinStatus(api.TrackerStatusUnpinned)
	PinStatusUnknown   = PinStatus(-1)
)

func (s PinStatus) String() string {
	if s == PinStatusPinned {
		return "Pinned"
	}
	if s == PinStatusPinning {
		return "Pinning"
	}
	if s == PinStatusPinQueued {
		return "PinQueued"
	}
	if s == PinStatusRemote {
		return "Remote"
	}
	if s == PinStatusUnpinned {
		return "Unpinned"
	}
	return "Unknown"
}

type Pin struct {
	PeerID   peer.ID
	PeerName string
	Region   string
	Status   PinStatus
	Updated  time.Time
}

type pinJson struct {
	PeerID   string `json:"peerId"`
	PeerName string `json:"peerName"`
	Region   string `json:"region"`
	Status   string `json:"status"`
	Updated  string `json:"updated"`
}

func (p *Pin) UnmarshalJSON(b []byte) error {
	var raw pinJson
	err := json.Unmarshal(b, &raw)
	if err != nil {
		return err
	}
	p.PeerID, err = peer.Decode(raw.PeerID)
	if err != nil {
		return err
	}
	p.PeerName = raw.PeerName
	p.Region = raw.Region
	if raw.Status == "Pinned" {
		p.Status = PinStatusPinned
	} else if raw.Status == "Pinning" {
		p.Status = PinStatusPinning
	} else if raw.Status == "PinQueued" {
		p.Status = PinStatusPinQueued
	} else if raw.Status == "Remote" {
		p.Status = PinStatusRemote
	} else if raw.Status == "Unpinned" {
		p.Status = PinStatusUnpinned
	} else {
		p.Status = PinStatusUnknown
	}
	return nil
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
	DealID            uint64
	StorageProvider   address.Address
	Status            DealStatus
	PieceCid          cid.Cid
	DataCid           cid.Cid
	DataModelSelector string
	Activation        time.Time
	Created           time.Time
	Updated           time.Time
}

type dealJson struct {
	DealID            uint64 `json:"dealId,omitempty"`
	StorageProvider   string `json:"storageProvider,omitempty"`
	Status            string `json:"status"`
	PieceCid          string `json:"pieceCid,omitempty"`
	DataCid           string `json:"dataCid,omitempty"`
	DataModelSelector string `json:"dataModelSelector,omitempty"`
	Activation        string `json:"activation,omitempty"`
	Created           string `json:"created"`
	Updated           string `json:"updated"`
}

func (d *Deal) UnmarshalJSON(b []byte) error {
	var raw dealJson
	err := json.Unmarshal(b, &raw)
	if err != nil {
		return err
	}
	d.DealID = raw.DealID
	d.StorageProvider, err = address.NewFromString(raw.StorageProvider)
	if err != nil {
		return err
	}
	if raw.Status == "Queued" {
		d.Status = DealStatusQueued
	} else if raw.Status == "Published" {
		d.Status = DealStatusPublished
	} else if raw.Status == "Active" {
		d.Status = DealStatusActive
	} else {
		return fmt.Errorf("unknown deal status: %s", raw.Status)
	}
	if raw.PieceCid != "" {
		d.PieceCid, err = cid.Parse(raw.PieceCid)
		if err != nil {
			return err
		}
	} else {
		d.PieceCid = cid.Undef
	}
	if raw.DataCid != "" {
		d.DataCid, err = cid.Parse(raw.DataCid)
		if err != nil {
			return err
		}
	} else {
		d.DataCid = cid.Undef
	}
	d.DataModelSelector = raw.DataModelSelector
	if raw.Activation != "" {
		d.Activation, err = time.Parse(iso8601, raw.Activation)
		if err != nil {
			return err
		}
	}
	d.Created, err = time.Parse(iso8601, raw.Created)
	if err != nil {
		return err
	}
	d.Updated, err = time.Parse(iso8601, raw.Updated)
	if err != nil {
		return err
	}
	return nil
}

// Status is IPFS pin and Filecoin deal status for a given CID.
type Status struct {
	Cid     cid.Cid
	DagSize uint64
	Created time.Time
	Pins    []Pin
	Deals   []Deal
}

type statusJson struct {
	Cid     string `json:"cid"`
	DagSize uint64 `json:"dagSize"`
	Created string `json:"created"`
	Pins    []Pin  `json:"pins"`
	Deals   []Deal `json:"deals"`
}

func (s *Status) UnmarshalJSON(b []byte) error {
	var raw statusJson
	err := json.Unmarshal(b, &raw)
	if err != nil {
		return err
	}
	s.Cid, err = cid.Parse(raw.Cid)
	if err != nil {
		return err
	}
	s.DagSize = raw.DagSize
	s.Created, err = time.Parse(iso8601, raw.Created)
	if err != nil {
		return err
	}
	s.Pins = raw.Pins
	s.Deals = raw.Deals
	return nil
}

func (c *client) Status(ctx context.Context, cid cid.Cid) (*Status, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/status/%s", c.cfg.endpoint, cid), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.cfg.token))
	req.Header.Add("X-Client", clientName)
	res, err := c.cfg.hc.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected response status: %d", res.StatusCode)
	}
	var s Status
	d := json.NewDecoder(res.Body)
	err = d.Decode(&s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
