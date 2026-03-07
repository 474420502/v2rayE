package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"v2raye/backend-go/internal/service/native"
	"v2raye/backend-go/internal/storage"
)

type envelope struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

func TestClashRoutesRemoved(t *testing.T) {
	ts := newNativeTestServer(t)
	defer ts.Close()

	tests := []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/api/clash/proxies"},
		{method: http.MethodGet, path: "/api/clash/connections"},
		{method: http.MethodPost, path: "/api/clash/proxies/select", body: `{"selected":"any"}`},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			resp := doRequest(t, ts.Client(), tc.method, ts.URL+tc.path, tc.body)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("expected 404 for %s %s, got %d", tc.method, tc.path, resp.StatusCode)
			}
		})
	}
}

func TestConfigUpdatePersistsAcrossServiceInstances(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	store, err := storage.New(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	serviceA := native.New(dataDir, "xray", store)
	serverA := New("", "", serviceA)
	tsA := httptest.NewServer(serverA.httpServer.Handler)
	defer tsA.Close()

	updateBody := `{"systemProxyExceptions":"localhost,10.0.0.0/8","httpPort":19090}`
	putResp := doRequest(t, tsA.Client(), http.MethodPut, tsA.URL+"/api/config", updateBody)
	defer putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("PUT /api/config expected 200, got %d", putResp.StatusCode)
	}

	var putEnv envelope
	decodeJSON(t, putResp.Body, &putEnv)
	if putEnv.Code != 0 {
		t.Fatalf("PUT /api/config expected code=0, got %d", putEnv.Code)
	}

	serviceB := native.New(dataDir, "xray", store)
	serverB := New("", "", serviceB)
	tsB := httptest.NewServer(serverB.httpServer.Handler)
	defer tsB.Close()

	getResp := doRequest(t, tsB.Client(), http.MethodGet, tsB.URL+"/api/config", "")
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/config expected 200, got %d", getResp.StatusCode)
	}

	var getEnv envelope
	decodeJSON(t, getResp.Body, &getEnv)
	if getEnv.Code != 0 {
		t.Fatalf("GET /api/config expected code=0, got %d", getEnv.Code)
	}

	if got := getEnv.Data["systemProxyExceptions"]; got != "localhost,10.0.0.0/8" {
		t.Fatalf("systemProxyExceptions not persisted, got %#v", got)
	}

	httpPort, ok := getEnv.Data["httpPort"].(float64)
	if !ok || int(httpPort) != 19090 {
		t.Fatalf("httpPort not persisted, got %#v", getEnv.Data["httpPort"])
	}
}

func TestConfigUpdateNormalizesLegacyEngineAndTunFields(t *testing.T) {
	t.Parallel()

	ts := newNativeTestServer(t)
	defer ts.Close()

	legacyBody := `{"coreEngine":"auto","tunMode":"","enableTun":true,"tunStack":"gvisor"}`
	putResp := doRequest(t, ts.Client(), http.MethodPut, ts.URL+"/api/config", legacyBody)
	defer putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("PUT /api/config expected 200, got %d", putResp.StatusCode)
	}

	var putEnv envelope
	decodeJSON(t, putResp.Body, &putEnv)
	if putEnv.Code != 0 {
		t.Fatalf("PUT /api/config expected code=0, got %d", putEnv.Code)
	}

	if got := readStringField(t, putEnv.Data, "coreEngine"); got != "xray-core" {
		t.Fatalf("coreEngine not normalized, got %q", got)
	}
	if got := readStringField(t, putEnv.Data, "tunMode"); got != "gvisor" {
		t.Fatalf("tunMode not normalized from legacy fields, got %q", got)
	}
	if got := readStringField(t, putEnv.Data, "tunStack"); got != "gvisor" {
		t.Fatalf("tunStack not aligned with tunMode, got %q", got)
	}
	if got := readBoolField(t, putEnv.Data, "enableTun"); !got {
		t.Fatalf("enableTun expected true after normalization")
	}

	if got := readStringField(t, putEnv.Data, "dnsMode"); got == "" {
		t.Fatalf("dnsMode default should be present after update")
	}
	if _, ok := putEnv.Data["dnsList"]; !ok {
		t.Fatalf("dnsList default should be present after update")
	}
}

func newNativeTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	dataDir := t.TempDir()
	store, err := storage.New(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	svc := native.New(dataDir, "xray", store)
	server := New("", "", svc)
	return httptest.NewServer(server.httpServer.Handler)
}

func doRequest(t *testing.T, client *http.Client, method, url, body string) *http.Response {
	t.Helper()

	reqBody := bytes.NewBufferString(body)
	if body == "" {
		reqBody = bytes.NewBuffer(nil)
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func decodeJSON(t *testing.T, r io.Reader, dst interface{}) {
	t.Helper()

	dec := json.NewDecoder(r)
	if err := dec.Decode(dst); err != nil {
		t.Fatalf("decode json: %v", err)
	}
}

func readStringField(t *testing.T, data map[string]interface{}, key string) string {
	t.Helper()

	value, ok := data[key]
	if !ok {
		t.Fatalf("missing key %q in response data", key)
	}
	s, ok := value.(string)
	if !ok {
		t.Fatalf("key %q is not a string: %#v", key, value)
	}
	return s
}

func readBoolField(t *testing.T, data map[string]interface{}, key string) bool {
	t.Helper()

	value, ok := data[key]
	if !ok {
		t.Fatalf("missing key %q in response data", key)
	}
	b, ok := value.(bool)
	if !ok {
		t.Fatalf("key %q is not a bool: %#v", key, value)
	}
	return b
}
