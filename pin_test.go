package w3s

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/ipfs/go-cid"
)

var pinsHandler = func(w http.ResponseWriter, r *http.Request) {
	if !hasValidToken(w, r) {
		return
	}

	body := map[string]interface{}{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		fmt.Printf("json decode: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return

	}

	fmt.Printf("body: %+v\n", body)

	resp := map[string]interface{}{
		"requestId": fmt.Sprintf("pin-%s", body["cid"]),
		"status":    "pinned",
		"created":   "2022-02-20T15:04:05.999Z",
		"pin": map[string]interface{}{
			"cid": body["cid"],
		},
		"delegates": []interface{}{},
	}

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func TestPinsHappyPath(t *testing.T) {
	routes := routeMap{
		"/pins": {
			http.MethodPost: pinsHandler,
		},
	}

	hc, cleanup := startTestServer(t, routes)
	defer cleanup()

	client, err := NewClient(WithHTTPClient(hc), WithToken("validtoken"))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	c, _ := cid.Parse(helloRoot)

	pr, err := client.Pin(context.Background(), c)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}

	if pr.Pin.Cid.String() != helloRoot {
		t.Fatalf("got cid %s, wanted %s", pr.Pin.Cid.String(), helloRoot)
	}
}
