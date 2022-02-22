package w3s

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"testing"

	"github.com/ipfs/go-cid"
)

const (
	validToken = "validtoken"

	// a car containing a single file called helloword.txt
	helloRoot   = "bafybeicymili4gmgoa4xpx5jfghi7leffvai4fd47f6nxgrhq4ug6ekiga"
	helloCarHex = "3aa265726f6f747381d82a582500017012205862168e1986703977dfa9298e8fac852d408e147cf97cdb9a2787286f1148306776657273696f6e0162017012205862168e1986703977dfa9298e8fac852d408e147cf97cdb9a2787286f11483012380a2401551220315f5bdb76d078c43b8ac0064e4a0164612b1fce77c869345bfc94c75894edd3120e68656c6c6f776f726c642e747874180d0a0208013101551220315f5bdb76d078c43b8ac0064e4a0164612b1fce77c869345bfc94c75894edd348656c6c6f2c20776f726c6421"

	// a car containing a single file called thanks.txt
	thanksRoot   = "bafybeid7orcaehmy2lzlkr4wnfgexmm2xoonmamaimdsjycex7wu4pjip4"
	thanksCarHex = "3aa265726f6f747381d82a582500017012207f7444021d98d2f2b54796694c4bb19abb9cd60180430724e044bfed4e3d287f6776657273696f6e015e017012207f7444021d98d2f2b54796694c4bb19abb9cd60180430724e044bfed4e3d287f12340a24015512200386a02a5f79b12d40569f36f0e3623d71f6655d00c5c0fc3826b4a945670685120a7468616e6b732e74787418170a0208013b015512200386a02a5f79b12d40569f36f0e3623d71f6655d00c5c0fc3826b4a9456706855468616e6b7320666f7220616c6c207468652066697368"
)

type routeMap map[string]map[string]http.HandlerFunc

func startTestServer(t *testing.T, routes routeMap) (*http.Client, func()) {
	mux := http.NewServeMux()
	for path, methodHandlers := range routes {
		for method, handler := range methodHandlers {
			mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != method {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				handler(w, r)
			})
		}
	}

	ts := httptest.NewServer(mux)

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("failed to parse httptest.Server URL: %v", err)
	}

	hc := &http.Client{
		Transport: urlRewriteTransport{URL: u},
	}

	return hc, func() {
		ts.Close()
	}
}

type urlRewriteTransport struct {
	Transport http.RoundTripper
	URL       *url.URL
}

func (t urlRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = t.URL.Scheme
	req.URL.Host = t.URL.Host
	req.URL.Path = path.Join(t.URL.Path, req.URL.Path)
	rt := t.Transport
	if rt == nil {
		rt = http.DefaultTransport
	}
	return rt.RoundTrip(req)
}

var getHelloCarHandler = func(w http.ResponseWriter, r *http.Request) {
	carbytes, err := hex.DecodeString(helloCarHex)
	if err != nil {
		fmt.Printf("DecodeString: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/car")
	w.Header().Set("Content-Disposition", `attachment; filename="`+helloRoot+`.car"`)

	w.WriteHeader(http.StatusOK)
	w.Write(carbytes)
}

func TestGetHappyPath(t *testing.T) {
	routes := routeMap{
		"/car/" + helloRoot: {
			http.MethodGet: getHelloCarHandler,
		},
	}

	hc, cleanup := startTestServer(t, routes)
	defer cleanup()

	client, err := NewClient(WithHTTPClient(hc), WithToken("validtoken"))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	c, _ := cid.Parse(helloRoot)
	resp, err := client.Get(context.Background(), c)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("got status %d, wanted %d", resp.StatusCode, 200)
	}

	f, fsys, err := resp.Files()
	if err != nil {
		t.Fatalf("failed to read files: %v", err)
	}

	info, err := f.Stat()
	if err != nil {
		t.Fatalf("failed to send stat car: %v", err)
	}

	if !info.IsDir() {
		t.Fatalf("expected a car containing a directory of files")
	}
	err = fs.WalkDir(fsys, "/", func(path string, d fs.DirEntry, werr error) error {
		_, err := d.Info()
		return err
	})
	if err != nil {
		t.Fatalf("failed to send walk car: %v", err)
	}
}
