package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
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

func TestRoutingDiagnosticsEndpoint(t *testing.T) {
	t.Parallel()

	ts := newNativeTestServer(t)
	defer ts.Close()

	resp := doRequest(t, ts.Client(), http.MethodGet, ts.URL+"/api/routing/diagnostics", "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/routing/diagnostics expected 200, got %d", resp.StatusCode)
	}

	var env envelope
	decodeJSON(t, resp.Body, &env)
	if env.Code != 0 {
		t.Fatalf("GET /api/routing/diagnostics expected code=0, got %d", env.Code)
	}

	if _, ok := env.Data["mode"]; !ok {
		t.Fatalf("missing mode in diagnostics response")
	}
	if _, ok := env.Data["tunMode"]; !ok {
		t.Fatalf("missing tunMode in diagnostics response")
	}
	if _, ok := env.Data["rules"]; !ok {
		t.Fatalf("missing rules in diagnostics response")
	}
}

func TestRoutingHitsEndpoint(t *testing.T) {
	t.Parallel()

	ts := newNativeTestServer(t)
	defer ts.Close()

	resp := doRequest(t, ts.Client(), http.MethodGet, ts.URL+"/api/routing/hits", "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/routing/hits expected 200, got %d", resp.StatusCode)
	}

	var env envelope
	decodeJSON(t, resp.Body, &env)
	if env.Code != 0 {
		t.Fatalf("GET /api/routing/hits expected code=0, got %d", env.Code)
	}

	if _, ok := env.Data["items"]; !ok {
		t.Fatalf("missing items in routing hits response")
	}
}

func TestRoutingTestEndpoint(t *testing.T) {
	t.Parallel()

	ts := newNativeTestServer(t)
	defer ts.Close()

	routingBody := `{"mode":"custom","domainStrategy":"AsIs","rules":[{"id":"proxy-example","type":"domain","values":["domain:example.com"],"outbound":"proxy"},{"id":"block-ads","type":"domain","values":["keyword:ads"],"outbound":"block"}]}`
	putResp := doRequest(t, ts.Client(), http.MethodPut, ts.URL+"/api/routing", routingBody)
	defer putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("PUT /api/routing expected 200, got %d", putResp.StatusCode)
	}

	resp := doRequest(t, ts.Client(), http.MethodPost, ts.URL+"/api/routing/test", `{"target":"api.example.com","protocol":"tcp","port":443}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/routing/test expected 200, got %d", resp.StatusCode)
	}

	var env envelope
	decodeJSON(t, resp.Body, &env)
	if env.Code != 0 {
		t.Fatalf("POST /api/routing/test expected code=0, got %d", env.Code)
	}
	if got := readStringField(t, env.Data, "outbound"); got != "proxy" {
		t.Fatalf("expected outbound=proxy, got %q", got)
	}
	if got := readStringField(t, env.Data, "matchedValue"); got != "domain:example.com" {
		t.Fatalf("expected matchedValue=domain:example.com, got %q", got)
	}
}

func TestRoutingTestEndpointLocalhostBypass(t *testing.T) {
	t.Parallel()

	ts := newNativeTestServer(t)
	defer ts.Close()

	routingBody := `{"mode":"custom","domainStrategy":"AsIs","rules":[{"id":"force-proxy","type":"domain","values":["domain:localhost"],"outbound":"proxy"}]}`
	putResp := doRequest(t, ts.Client(), http.MethodPut, ts.URL+"/api/routing", routingBody)
	defer putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("PUT /api/routing expected 200, got %d", putResp.StatusCode)
	}

	resp := doRequest(t, ts.Client(), http.MethodPost, ts.URL+"/api/routing/test", `{"target":"localhost","protocol":"tcp","port":18000}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/routing/test expected 200, got %d", resp.StatusCode)
	}

	var env envelope
	decodeJSON(t, resp.Body, &env)
	if env.Code != 0 {
		t.Fatalf("POST /api/routing/test expected code=0, got %d", env.Code)
	}
	if got := readStringField(t, env.Data, "outbound"); got != "direct" {
		t.Fatalf("expected outbound=direct for localhost control-plane traffic, got %q", got)
	}
	if got := readStringField(t, env.Data, "matchedValue"); got != "full:localhost" {
		t.Fatalf("expected matchedValue=full:localhost, got %q", got)
	}
}

func TestRoutingTestEndpointLocalBypassCanBeDisabled(t *testing.T) {
	t.Parallel()

	ts := newNativeTestServer(t)
	defer ts.Close()

	routingBody := `{"mode":"custom","domainStrategy":"AsIs","localBypassEnabled":false,"rules":[{"id":"force-proxy-localhost","type":"domain","values":["full:localhost"],"outbound":"proxy"}]}`
	putResp := doRequest(t, ts.Client(), http.MethodPut, ts.URL+"/api/routing", routingBody)
	defer putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("PUT /api/routing expected 200, got %d", putResp.StatusCode)
	}

	resp := doRequest(t, ts.Client(), http.MethodPost, ts.URL+"/api/routing/test", `{"target":"localhost","protocol":"tcp","port":18000}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/routing/test expected 200, got %d", resp.StatusCode)
	}

	var env envelope
	decodeJSON(t, resp.Body, &env)
	if env.Code != 0 {
		t.Fatalf("POST /api/routing/test expected code=0, got %d", env.Code)
	}
	if got := readStringField(t, env.Data, "outbound"); got != "proxy" {
		t.Fatalf("expected outbound=proxy when local bypass disabled, got %q", got)
	}
	if got := readStringField(t, env.Data, "matchedValue"); got != "full:localhost" {
		t.Fatalf("expected matchedValue=full:localhost, got %q", got)
	}
}

func TestRoutingTestEndpointDirectModeDefaultAndCustom(t *testing.T) {
	t.Parallel()

	ts := newNativeTestServer(t)
	defer ts.Close()

	routingBody := `{"mode":"direct","domainStrategy":"AsIs","rules":[{"id":"example-proxy","type":"domain","values":["full:example.com"],"outbound":"proxy"}]}`
	putResp := doRequest(t, ts.Client(), http.MethodPut, ts.URL+"/api/routing", routingBody)
	defer putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("PUT /api/routing expected 200, got %d", putResp.StatusCode)
	}

	t.Run("custom rule takes precedence", func(t *testing.T) {
		resp := doRequest(t, ts.Client(), http.MethodPost, ts.URL+"/api/routing/test", `{"target":"example.com","protocol":"tcp","port":443}`)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("POST /api/routing/test expected 200, got %d", resp.StatusCode)
		}

		var env envelope
		decodeJSON(t, resp.Body, &env)
		if env.Code != 0 {
			t.Fatalf("POST /api/routing/test expected code=0, got %d", env.Code)
		}
		if got := readStringField(t, env.Data, "outbound"); got != "proxy" {
			t.Fatalf("expected outbound=proxy for custom direct-mode rule, got %q", got)
		}
		if got := readStringField(t, env.Data, "matchedValue"); got != "full:example.com" {
			t.Fatalf("expected matchedValue=full:example.com, got %q", got)
		}
	})

	t.Run("unmatched target falls back to direct", func(t *testing.T) {
		resp := doRequest(t, ts.Client(), http.MethodPost, ts.URL+"/api/routing/test", `{"target":"unmatched.example.net","protocol":"tcp","port":443}`)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("POST /api/routing/test expected 200, got %d", resp.StatusCode)
		}

		var env envelope
		decodeJSON(t, resp.Body, &env)
		if env.Code != 0 {
			t.Fatalf("POST /api/routing/test expected code=0, got %d", env.Code)
		}
		if got := readStringField(t, env.Data, "outbound"); got != "direct" {
			t.Fatalf("expected outbound=direct fallback in direct mode, got %q", got)
		}
		if got := readStringField(t, env.Data, "note"); got != "default direct mode" {
			t.Fatalf("expected note=default direct mode, got %q", got)
		}
	})
}

func TestRoutingTestEndpointBypassCNPrivateIPDirect(t *testing.T) {
	t.Parallel()

	ts := newNativeTestServer(t)
	defer ts.Close()

	routingBody := `{"mode":"bypass_cn","domainStrategy":"IPIfNonMatch","rules":[{"id":"force-proxy-private","type":"ip","values":["10.0.0.0/8"],"outbound":"proxy"}]}`
	putResp := doRequest(t, ts.Client(), http.MethodPut, ts.URL+"/api/routing", routingBody)
	defer putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("PUT /api/routing expected 200, got %d", putResp.StatusCode)
	}

	resp := doRequest(t, ts.Client(), http.MethodPost, ts.URL+"/api/routing/test", `{"target":"10.2.3.4","protocol":"tcp","port":443}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/routing/test expected 200, got %d", resp.StatusCode)
	}

	var env envelope
	decodeJSON(t, resp.Body, &env)
	if env.Code != 0 {
		t.Fatalf("POST /api/routing/test expected code=0, got %d", env.Code)
	}
	if got := readStringField(t, env.Data, "outbound"); got != "direct" {
		t.Fatalf("expected outbound=direct for private IP in bypass_cn mode, got %q", got)
	}
}

func TestRoutingTunRepairEndpoint(t *testing.T) {
	t.Parallel()

	ts := newNativeTestServer(t)
	defer ts.Close()

	resp := doRequest(t, ts.Client(), http.MethodPost, ts.URL+"/api/routing/tun/repair", "{}")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/routing/tun/repair expected 200, got %d", resp.StatusCode)
	}

	var env envelope
	decodeJSON(t, resp.Body, &env)
	if env.Code != 0 {
		t.Fatalf("POST /api/routing/tun/repair expected code=0, got %d", env.Code)
	}

	if _, ok := env.Data["triggeredAt"]; !ok {
		t.Fatalf("missing triggeredAt in tun repair response")
	}
}

func TestSystemProxyUsersEndpoint(t *testing.T) {
	t.Parallel()

	ts := newNativeTestServer(t)
	defer ts.Close()

	resp := doRequest(t, ts.Client(), http.MethodGet, ts.URL+"/api/system-proxy/users", "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/system-proxy/users expected 200, got %d", resp.StatusCode)
	}

	var env struct {
		Code int                      `json:"code"`
		Data []map[string]interface{} `json:"data"`
	}
	decodeJSON(t, resp.Body, &env)
	if env.Code != 0 {
		t.Fatalf("GET /api/system-proxy/users expected code=0, got %d", env.Code)
	}
	if env.Data == nil {
		t.Fatalf("expected users list, got nil")
	}
}

func TestProfilesBatchDelayEndpoint(t *testing.T) {
	t.Parallel()

	ts := newNativeTestServer(t)
	defer ts.Close()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			_ = conn.Close()
		}
	}()

	type createProfileResponse struct {
		Code int `json:"code"`
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	goodProfileBody := `{"name":"good","protocol":"vless","address":"127.0.0.1","port":` + readPort(listener.Addr().String()) + `,"vless":{"uuid":"11111111-1111-1111-1111-111111111111","encryption":"none"}}`
	goodResp := doRequest(t, ts.Client(), http.MethodPost, ts.URL+"/api/profiles", goodProfileBody)
	defer goodResp.Body.Close()
	var goodEnv createProfileResponse
	decodeJSON(t, goodResp.Body, &goodEnv)

	badProfileBody := `{"name":"bad","protocol":"vless","address":"127.0.0.1","port":1,"vless":{"uuid":"22222222-2222-2222-2222-222222222222","encryption":"none"}}`
	badResp := doRequest(t, ts.Client(), http.MethodPost, ts.URL+"/api/profiles", badProfileBody)
	defer badResp.Body.Close()
	var badEnv createProfileResponse
	decodeJSON(t, badResp.Body, &badEnv)

	body := `{"profileIds":["` + goodEnv.Data.ID + `","` + badEnv.Data.ID + `"],"timeoutMs":500,"limit":2}`
	resp := doRequest(t, ts.Client(), http.MethodPost, ts.URL+"/api/profiles/delay/batch", body)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/profiles/delay/batch expected 200, got %d", resp.StatusCode)
	}

	var env envelope
	decodeJSON(t, resp.Body, &env)
	if env.Code != 0 {
		t.Fatalf("POST /api/profiles/delay/batch expected code=0, got %d", env.Code)
	}
	if got := int(env.Data["total"].(float64)); got != 2 {
		t.Fatalf("expected total=2, got %d", got)
	}
	if _, ok := env.Data["results"]; !ok {
		t.Fatalf("missing results in batch delay response")
	}
}

func TestBackendRejectsPublicClientByDefault(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	store, err := storage.New(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	svc := native.New(dataDir, "xray", store)
	server := New("", "", svc)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	rec := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for public client, got %d", rec.Code)
	}

	var env envelope
	decodeJSON(t, rec.Body, &env)
	if env.Code != 40301 {
		t.Fatalf("expected code=40301, got %d", env.Code)
	}
}

func TestBackendAllowsPublicClientWhenConfigured(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	store, err := storage.New(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	svc := native.New(dataDir, "xray", store)
	server := New("", "", svc, WithPublicAccessAllowed())

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	rec := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 when public access enabled, got %d", rec.Code)
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

func readPort(addr string) string {
	_, port, _ := net.SplitHostPort(addr)
	return port
}
