package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type apiClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

type apiEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	Details json.RawMessage `json:"details"`
}

type apiError struct {
	StatusCode int
	Code       int
	Message    string
}

func (e *apiError) Error() string {
	if e == nil {
		return ""
	}
	if e.Code != 0 {
		return fmt.Sprintf("api error %d: %s", e.Code, e.Message)
	}
	if e.StatusCode != 0 {
		return fmt.Sprintf("http %d: %s", e.StatusCode, e.Message)
	}
	return e.Message
}

func newAPIClient(baseURL, token string) *apiClient {
	return &apiClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   strings.TrimSpace(token),
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (c *apiClient) request(ctx context.Context, method, path string, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var env apiEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return err
		}
		return &apiError{StatusCode: resp.StatusCode, Message: resp.Status}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &apiError{StatusCode: resp.StatusCode, Code: env.Code, Message: env.Message}
	}
	if out != nil && len(env.Data) > 0 && string(env.Data) != "null" {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return err
		}
	}
	return nil
}

func (c *apiClient) streamSSE(ctx context.Context, path string, onOpen func(), handler func(event string, data []byte) error) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return &apiError{StatusCode: resp.StatusCode, Message: resp.Status}
	}
	if onOpen != nil {
		onOpen()
	}

	scanner := bufio.NewScanner(resp.Body)
	buffer := make([]byte, 0, 64*1024)
	scanner.Buffer(buffer, 1024*1024)

	eventName := "message"
	dataLines := make([]string, 0, 4)
	dispatch := func() error {
		if len(dataLines) == 0 {
			eventName = "message"
			return nil
		}
		payload := strings.Join(dataLines, "\n")
		dataLines = dataLines[:0]
		err := handler(eventName, []byte(payload))
		eventName = "message"
		return err
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if err := dispatch(); err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		if strings.HasPrefix(line, "event:") {
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			return nil
		}
		return err
	}
	return dispatch()
}

func (c *apiClient) GetCoreStatus(ctx context.Context) (CoreStatus, error) {
	var out CoreStatus
	err := c.request(ctx, http.MethodGet, "/api/core/status", nil, &out)
	return out, err
}

func (c *apiClient) StartCore(ctx context.Context) (CoreStatus, error) {
	var out CoreStatus
	err := c.request(ctx, http.MethodPost, "/api/core/start", map[string]any{}, &out)
	return out, err
}

func (c *apiClient) StopCore(ctx context.Context) (CoreStatus, error) {
	var out CoreStatus
	err := c.request(ctx, http.MethodPost, "/api/core/stop", map[string]any{}, &out)
	return out, err
}

func (c *apiClient) RestartCore(ctx context.Context) (CoreStatus, error) {
	var out CoreStatus
	err := c.request(ctx, http.MethodPost, "/api/core/restart", map[string]any{}, &out)
	return out, err
}

func (c *apiClient) ClearCoreError(ctx context.Context) (CoreStatus, error) {
	var out CoreStatus
	err := c.request(ctx, http.MethodPost, "/api/core/error/clear", map[string]any{}, &out)
	return out, err
}

func (c *apiClient) GetProfiles(ctx context.Context) ([]ProfileItem, error) {
	var out []ProfileItem
	err := c.request(ctx, http.MethodGet, "/api/profiles", nil, &out)
	return out, err
}

func (c *apiClient) SelectProfile(ctx context.Context, id string) error {
	return c.request(ctx, http.MethodPost, "/api/profiles/"+url.PathEscape(id)+"/select", map[string]any{}, nil)
}

func (c *apiClient) TestProfileDelay(ctx context.Context, id string) (DelayTestResult, error) {
	var out DelayTestResult
	err := c.request(ctx, http.MethodGet, "/api/profiles/"+url.PathEscape(id)+"/delay", nil, &out)
	return out, err
}

func (c *apiClient) BatchTestProfileDelay(ctx context.Context, ids []string, timeoutMs, limit int) (BatchDelayTestResult, error) {
	var out BatchDelayTestResult
	err := c.request(ctx, http.MethodPost, "/api/profiles/delay/batch", BatchDelayTestRequest{
		ProfileIDs: ids,
		TimeoutMs:  timeoutMs,
		Limit:      limit,
	}, &out)
	return out, err
}

func (c *apiClient) ImportProfile(ctx context.Context, uri string) (ProfileItem, error) {
	var out ProfileItem
	err := c.request(ctx, http.MethodPost, "/api/profiles/import", map[string]string{"uri": uri}, &out)
	return out, err
}

func (c *apiClient) GetSubscriptions(ctx context.Context) ([]SubscriptionItem, error) {
	var out []SubscriptionItem
	err := c.request(ctx, http.MethodGet, "/api/subscriptions", nil, &out)
	return out, err
}

func (c *apiClient) UpdateAllSubscriptions(ctx context.Context) error {
	return c.request(ctx, http.MethodPost, "/api/subscriptions/update", map[string]any{}, nil)
}

func (c *apiClient) UpdateSubscription(ctx context.Context, id string) error {
	return c.request(ctx, http.MethodPost, "/api/subscriptions/"+url.PathEscape(id)+"/update", map[string]any{}, nil)
}

func (c *apiClient) GetAvailability(ctx context.Context) (AvailabilityResult, error) {
	var out AvailabilityResult
	err := c.request(ctx, http.MethodGet, "/api/network/availability", nil, &out)
	return out, err
}

func (c *apiClient) ApplySystemProxy(ctx context.Context, mode, exceptions string) (map[string]any, error) {
	var out map[string]any
	err := c.request(ctx, http.MethodPost, "/api/system-proxy/apply", SystemProxyApplyRequest{Mode: mode, Exceptions: exceptions}, &out)
	return out, err
}

func (c *apiClient) ExitCleanup(ctx context.Context, shutdown bool) (map[string]any, error) {
	var out map[string]any
	err := c.request(ctx, http.MethodPost, "/api/app/exit-cleanup", map[string]bool{"shutdownBackend": shutdown}, &out)
	return out, err
}

func (c *apiClient) GetConfig(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	err := c.request(ctx, http.MethodGet, "/api/config", nil, &out)
	return out, err
}

func (c *apiClient) UpdateConfig(ctx context.Context, cfg map[string]any) (map[string]any, error) {
	var out map[string]any
	err := c.request(ctx, http.MethodPut, "/api/config", cfg, &out)
	return out, err
}

func (c *apiClient) GetRouting(ctx context.Context) (RoutingConfig, error) {
	var out RoutingConfig
	err := c.request(ctx, http.MethodGet, "/api/routing", nil, &out)
	return out, err
}

func (c *apiClient) GetRoutingDiagnostics(ctx context.Context) (RoutingDiagnostics, error) {
	var out RoutingDiagnostics
	err := c.request(ctx, http.MethodGet, "/api/routing/diagnostics", nil, &out)
	return out, err
}

func (c *apiClient) GetRoutingHits(ctx context.Context) (RoutingHitStats, error) {
	var out RoutingHitStats
	err := c.request(ctx, http.MethodGet, "/api/routing/hits", nil, &out)
	return out, err
}

func (c *apiClient) TestRouting(ctx context.Context, req RoutingTestRequest) (RoutingTestResult, error) {
	var out RoutingTestResult
	err := c.request(ctx, http.MethodPost, "/api/routing/test", req, &out)
	return out, err
}

func (c *apiClient) UpdateGeoData(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	err := c.request(ctx, http.MethodPost, "/api/routing/geodata/update", map[string]any{}, &out)
	return out, err
}

func (c *apiClient) RepairTun(ctx context.Context) (TunRepairResult, error) {
	var out TunRepairResult
	err := c.request(ctx, http.MethodPost, "/api/routing/tun/repair", map[string]any{}, &out)
	return out, err
}

func (c *apiClient) GetStats(ctx context.Context) (StatsResult, error) {
	var out StatsResult
	err := c.request(ctx, http.MethodGet, "/api/stats", nil, &out)
	return out, err
}

func (c *apiClient) StreamLogs(ctx context.Context, onOpen func(), handler func(LogLine) error) error {
	return c.streamSSE(ctx, "/api/logs/stream", onOpen, func(event string, data []byte) error {
		if event != "log" {
			return nil
		}
		var line LogLine
		if err := json.Unmarshal(data, &line); err != nil {
			return err
		}
		return handler(line)
	})
}

func (c *apiClient) StreamEvents(ctx context.Context, onOpen func(), handler func(EventMessage) error) error {
	return c.streamSSE(ctx, "/api/events/stream", onOpen, func(event string, data []byte) error {
		if event != "message" {
			return nil
		}
		var msg EventMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return err
		}
		return handler(msg)
	})
}
