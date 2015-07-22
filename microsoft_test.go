package microsoft

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func makeServer(code int, body string) (*httptest.Server, *http.Client) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, body)
	}))
	defer server.Close()

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	httpClient := &http.Client{Transport: transport}
	return server, httpClient
}

func TestGetToken(t *testing.T) {

	_, mockClient := makeServer(200, `{"token_id": "token_mock_id", "expires_in": "600"}`)

	authReq := AuthRequest{ClientId: "translate1", ClientSecret: "translates3cr3t", HTTPClient: mockClient}
	resp := authReq.GetAccessToken()
	t.Logf("%#v\n", resp)
}
