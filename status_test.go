package w3s

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/ipfs/go-cid"
)

var statusHelloCarHandler = func(w http.ResponseWriter, r *http.Request) {
	carbytes, err := hex.DecodeString(helloCarHex)
	if err != nil {
		fmt.Printf("DecodeString: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	status := map[string]interface{}{
		"cid":     helloRoot,
		"dagSize": len(carbytes),
		"created": "2022-02-20T15:04:05.999Z",
		"pins":    []interface{}{},
		"deals":   []interface{}{},
	}

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

func TestStatusHappyPath(t *testing.T) {
	routes := routeMap{
		"/status/" + helloRoot: {
			http.MethodGet: statusHelloCarHandler,
		},
	}

	hc, cleanup := startTestServer(t, routes)
	defer cleanup()

	client, err := NewClient(WithHTTPClient(hc), WithToken("validtoken"))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	c, _ := cid.Parse(helloRoot)

	st, err := client.Status(context.Background(), c)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}

	if st.Cid.String() != helloRoot {
		t.Fatalf("got cid %s, wanted %s", st.Cid.String(), helloRoot)
	}

	if st.DagSize != 208 {
		t.Fatalf("got dagsize %d, wanted %d", st.DagSize, 208)
	}
}

var statusIncompleteDealHandler = func(w http.ResponseWriter, r *http.Request) {
	carbytes, err := hex.DecodeString(helloCarHex)
	if err != nil {
		fmt.Printf("DecodeString: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	status := map[string]interface{}{
		"cid":     helloRoot,
		"dagSize": len(carbytes),
		"created": "2022-02-20T15:04:05.999Z",
		"pins":    []interface{}{},
		"deals": []map[string]interface{}{
			{
				"dealId":            0,
				"storageProvider":   "",
				"status":            "Queued",
				"pieceCid":          "baga6ea4seaql2leesalhfwmb5xbvhrotu4x76mxn25wcpile6qpbn777omdmqfa",
				"dataCid":           "bafybeieesr6dknelbduxqmmtt7q2nernm4mmka4k66t7f4d6sgsq3ixc4u",
				"dataModelSelector": "Links/144/Hash/Links/2/Hash/Links/0/Hash",
				"activation":        "",
				"expiration":        "",
				"created":           "",
				"updated":           "",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

func TestStatusIncompleteDeal(t *testing.T) {
	routes := routeMap{
		"/status/" + helloRoot: {
			http.MethodGet: statusIncompleteDealHandler,
		},
	}

	hc, cleanup := startTestServer(t, routes)
	defer cleanup()

	client, err := NewClient(WithHTTPClient(hc), WithToken("validtoken"))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	c, _ := cid.Parse(helloRoot)

	st, err := client.Status(context.Background(), c)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}

	if st.Cid.String() != helloRoot {
		t.Fatalf("got cid %s, wanted %s", st.Cid.String(), helloRoot)
	}

	if st.DagSize != 208 {
		t.Fatalf("got dagsize %d, wanted %d", st.DagSize, 208)
	}
}
