package w3s

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/ipld/go-car"
)

func hasValidToken(w http.ResponseWriter, r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	if auth != "Bearer "+validToken {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}

	return true
}

var putCarHandler = func(w http.ResponseWriter, r *http.Request) {
	if !hasValidToken(w, r) {
		return
	}

	cr, err := car.NewCarReader(r.Body)
	if err != nil {
		fmt.Printf("NewCarReader: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
	}
	rootCid := cr.Header.Roots[0]

	w.WriteHeader(http.StatusOK)
	var out struct {
		Cid string `json:"cid"`
	}
	out.Cid = rootCid.String()

	json.NewEncoder(w).Encode(out)
}

func TestPutCarHappyPath(t *testing.T) {
	routes := routeMap{
		"/car": {
			http.MethodPost: putCarHandler,
		},
	}

	hc, cleanup := startTestServer(t, routes)
	defer cleanup()

	client, err := NewClient(WithHTTPClient(hc), WithToken("validtoken"))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	carbytes, err := hex.DecodeString(helloCarHex)
	if err != nil {
		t.Fatalf("failed to decode car hex: %v", err)
		return
	}

	c, err := client.PutCar(context.Background(), bytes.NewReader(carbytes))
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}

	if c.String() != helloRoot {
		t.Fatalf("got cid %s, wanted %s", c.String(), helloRoot)
	}
}
