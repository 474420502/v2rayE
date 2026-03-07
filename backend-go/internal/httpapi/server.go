package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"v2raye/backend-go/internal/domain"
	"v2raye/backend-go/internal/service"
)

// Server is the HTTP API server.
type Server struct {
	httpServer   *http.Server
	requireToken bool
	token        string
	svc          service.BackendService

	mu          sync.Mutex
	subscribers map[int]chan []byte
	nextSubID   int
}

// New creates an HTTP API server bound to addr with optional Bearer token auth.
func New(addr, token string, svc service.BackendService) *Server {
	requireToken := strings.TrimSpace(token) != ""
	s := &Server{
		requireToken: requireToken,
		token:        token,
		svc:          svc,
		subscribers:  make(map[int]chan []byte),
	}

	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("/api/health", s.handleHealth)

	// Core
	mux.HandleFunc("/api/core/status", s.auth(s.handleCoreStatus))
	mux.HandleFunc("/api/core/start", s.auth(s.handleCoreStart))
	mux.HandleFunc("/api/core/stop", s.auth(s.handleCoreStop))
	mux.HandleFunc("/api/core/restart", s.auth(s.handleCoreRestart))
	mux.HandleFunc("/api/core/error/clear", s.auth(s.handleCoreErrorClear))

	// Profiles (CRUD + import)
	mux.HandleFunc("/api/profiles/delete", s.auth(s.handleProfilesDelete))
	mux.HandleFunc("/api/profiles/import", s.auth(s.handleProfileImport))
	mux.HandleFunc("/api/profiles", s.auth(s.handleProfiles))
	mux.HandleFunc("/api/profiles/", s.auth(s.handleProfileOps))

	// Subscriptions
	mux.HandleFunc("/api/subscriptions/update", s.auth(s.handleSubscriptionsUpdate))
	mux.HandleFunc("/api/subscriptions", s.auth(s.handleSubscriptions))
	mux.HandleFunc("/api/subscriptions/", s.auth(s.handleSubscriptionOps))

	// Network & proxy
	mux.HandleFunc("/api/network/availability", s.auth(s.handleNetworkAvailability))
	mux.HandleFunc("/api/system-proxy/apply", s.auth(s.handleSystemProxyApply))

	// Config & routing
	mux.HandleFunc("/api/config", s.auth(s.handleConfig))
	mux.HandleFunc("/api/routing", s.auth(s.handleRouting))

	// Stats & logs
	mux.HandleFunc("/api/stats", s.auth(s.handleStats))
	mux.HandleFunc("/api/logs/stream", s.auth(s.handleLogsStream))

	// Events SSE (metadata events)
	mux.HandleFunc("/api/events/stream", s.auth(s.handleEventsStream))

	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return s
}

// Run starts the server and blocks until ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.httpServer.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.httpServer.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

// ─── Auth middleware ──────────────────────────────────────────────────────────

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.requireToken {
			next(w, r)
			return
		}
		if extractRequestToken(r) != s.token {
			writeError(w, http.StatusUnauthorized, 40101, "unauthorized", nil)
			return
		}
		next(w, r)
	}
}

func extractRequestToken(r *http.Request) string {
	hdr := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(hdr, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(hdr, "Bearer "))
	}
	if ck, err := r.Cookie("auth_token"); err == nil {
		return strings.TrimSpace(ck.Value)
	}
	return ""
}

// ─── Core handlers ────────────────────────────────────────────────────────────

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, 40501, "method not allowed", nil)
		return
	}
	writeOK(w, map[string]interface{}{"status": "healthy", "ts": time.Now().UTC().Format(time.RFC3339)})
}

func (s *Server) handleCoreStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, 40501, "method not allowed", nil)
		return
	}
	writeOK(w, s.svc.CoreStatus())
}

func (s *Server) handleCoreStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, 40501, "method not allowed", nil)
		return
	}
	status := s.svc.StartCore()
	if !status.Running {
		message := "core failed to start; check profile selection, listen ports, and engine config"
		if strings.TrimSpace(status.Error) != "" {
			message = status.Error
		}
		writeError(
			w,
			http.StatusConflict,
			40901,
			message,
			status,
		)
		s.publishEvent("core.start_failed", status)
		return
	}
	writeOK(w, status)
	s.publishEvent("core.started", status)
}

func (s *Server) handleCoreStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, 40501, "method not allowed", nil)
		return
	}
	status := s.svc.StopCore()
	writeOK(w, status)
	s.publishEvent("core.stopped", status)
}

func (s *Server) handleCoreRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, 40501, "method not allowed", nil)
		return
	}
	status := s.svc.RestartCore()
	writeOK(w, status)
	s.publishEvent("core.restarted", status)
}

func (s *Server) handleCoreErrorClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, 40501, "method not allowed", nil)
		return
	}
	status := s.svc.ClearCoreError()
	writeOK(w, status)
	s.publishEvent("core.error_cleared", status)
}

// ─── Profile handlers ─────────────────────────────────────────────────────────

// handleProfiles handles GET /api/profiles and POST /api/profiles.
func (s *Server) handleProfiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeOK(w, s.svc.ListProfiles())
	case http.MethodPost:
		var p domain.ProfileItem
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeError(w, http.StatusUnprocessableEntity, 42201, "invalid json", nil)
			return
		}
		created, err := s.svc.CreateProfile(p)
		if jsonErr(w, err) {
			return
		}
		writeOK(w, created)
		s.publishEvent("profile.updated", map[string]interface{}{"id": created.ID, "created": true})
	default:
		writeError(w, http.StatusMethodNotAllowed, 40501, "method not allowed", nil)
	}
}

// handleProfileImport handles POST /api/profiles/import (URI string body).
func (s *Server) handleProfileImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, 40501, "method not allowed", nil)
		return
	}
	var body struct {
		URI string `json:"uri"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.URI) == "" {
		writeError(w, http.StatusUnprocessableEntity, 42201, "missing uri field", nil)
		return
	}
	p, err := s.svc.ImportProfileFromURI(strings.TrimSpace(body.URI))
	if jsonErr(w, err) {
		return
	}
	writeOK(w, p)
	s.publishEvent("profile.updated", map[string]interface{}{"id": p.ID, "created": true})
}

func (s *Server) handleProfilesDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, 40501, "method not allowed", nil)
		return
	}
	var body struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.IDs) == 0 {
		writeError(w, http.StatusUnprocessableEntity, 42201, "missing ids field", nil)
		return
	}
	if err := s.svc.DeleteProfiles(body.IDs); err != nil {
		if err == service.ErrNotFound {
			writeError(w, http.StatusNotFound, 40401, "profile not found", nil)
			return
		}
		if err == service.ErrInvalidMode {
			writeError(w, http.StatusUnprocessableEntity, 42201, "invalid ids", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, 50001, "internal error", nil)
		return
	}
	writeOK(w, map[string]int{"deleted": len(body.IDs)})
	s.publishEvent("profile.updated", map[string]interface{}{"ids": body.IDs, "deleted": len(body.IDs)})
}

// handleProfileOps routes /api/profiles/{id}[/action].
func (s *Server) handleProfileOps(w http.ResponseWriter, r *http.Request) {
	// path: api/profiles/{id}[/select|/delay]
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		writeError(w, http.StatusNotFound, 40401, "not found", nil)
		return
	}
	profileID := parts[2]

	if len(parts) == 4 {
		switch parts[3] {
		case "select":
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, 40501, "method not allowed", nil)
				return
			}
			if err := s.svc.SelectProfile(profileID); err != nil {
				if err == service.ErrNotFound {
					writeError(w, http.StatusNotFound, 40401, "profile not found", nil)
					return
				}
				writeError(w, http.StatusInternalServerError, 50001, "internal error", nil)
				return
			}
			writeOK(w, map[string]string{"selected": profileID})
			s.publishEvent("profile.selected", map[string]string{"selected": profileID})
			return
		case "delay":
			if r.Method != http.MethodGet {
				writeError(w, http.StatusMethodNotAllowed, 40501, "method not allowed", nil)
				return
			}
			writeOK(w, s.svc.TestProfileDelay(profileID))
			return
		}
	}

	// /api/profiles/{id} — GET, PUT, DELETE
	switch r.Method {
	case http.MethodGet:
		p, err := s.svc.GetProfile(profileID)
		if jsonErr(w, err) {
			return
		}
		writeOK(w, p)
	case http.MethodPut:
		var p domain.ProfileItem
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeError(w, http.StatusUnprocessableEntity, 42201, "invalid json", nil)
			return
		}
		updated, err := s.svc.UpdateProfile(profileID, p)
		if jsonErr(w, err) {
			return
		}
		writeOK(w, updated)
		s.publishEvent("profile.updated", map[string]interface{}{"id": profileID})
	case http.MethodDelete:
		if err := s.svc.DeleteProfile(profileID); err != nil {
			if err == service.ErrNotFound {
				writeError(w, http.StatusNotFound, 40401, "profile not found", nil)
				return
			}
			writeError(w, http.StatusInternalServerError, 50001, "internal error", nil)
			return
		}
		writeOK(w, map[string]string{"deleted": profileID})
		s.publishEvent("profile.updated", map[string]interface{}{"id": profileID, "deleted": true})
	default:
		writeError(w, http.StatusMethodNotAllowed, 40501, "method not allowed", nil)
	}
}

// ─── Subscription handlers ────────────────────────────────────────────────────

func (s *Server) handleSubscriptions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeOK(w, s.svc.ListSubscriptions())
	case http.MethodPost:
		var req domain.SubscriptionUpsertRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusUnprocessableEntity, 42201, "invalid json", nil)
			return
		}
		item, err := s.svc.CreateSubscription(req)
		if jsonErr(w, err) {
			return
		}
		writeOK(w, item)
		s.publishEvent("subscription.updated", map[string]interface{}{"id": item.ID, "created": true})
	default:
		writeError(w, http.StatusMethodNotAllowed, 40501, "method not allowed", nil)
	}
}

func (s *Server) handleSubscriptionsUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, 40501, "method not allowed", nil)
		return
	}
	updated := s.svc.UpdateSubscriptions()
	writeOK(w, map[string]int{"updated": updated})
	s.publishEvent("subscription.updated", map[string]int{"updated": updated})
}

func (s *Server) handleSubscriptionOps(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		writeError(w, http.StatusNotFound, 40401, "not found", nil)
		return
	}
	subID := parts[2]

	// /api/subscriptions/{id}/update
	if len(parts) == 4 && parts[3] == "update" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, 40501, "method not allowed", nil)
			return
		}
		if err := s.svc.UpdateSubscriptionByID(subID); err != nil {
			if err == service.ErrNotFound {
				writeError(w, http.StatusNotFound, 40401, "not found", nil)
				return
			}
			writeError(w, http.StatusInternalServerError, 50001, "update failed", nil)
			return
		}
		writeOK(w, map[string]int{"updated": 1})
		s.publishEvent("subscription.updated", map[string]interface{}{"id": subID, "updated": 1})
		return
	}

	switch r.Method {
	case http.MethodPut:
		var req domain.SubscriptionUpsertRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusUnprocessableEntity, 42201, "invalid json", nil)
			return
		}
		item, err := s.svc.UpdateSubscription(subID, req)
		if jsonErr(w, err) {
			return
		}
		writeOK(w, item)
		s.publishEvent("subscription.updated", map[string]interface{}{"id": subID})
	case http.MethodDelete:
		if err := s.svc.DeleteSubscription(subID); err != nil {
			if err == service.ErrNotFound {
				writeError(w, http.StatusNotFound, 40401, "not found", nil)
				return
			}
			writeError(w, http.StatusInternalServerError, 50001, "delete failed", nil)
			return
		}
		writeOK(w, map[string]string{"deleted": subID})
		s.publishEvent("subscription.updated", map[string]interface{}{"id": subID, "deleted": true})
	default:
		writeError(w, http.StatusMethodNotAllowed, 40501, "method not allowed", nil)
	}
}

// ─── Network & proxy handlers ─────────────────────────────────────────────────

func (s *Server) handleNetworkAvailability(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, 40501, "method not allowed", nil)
		return
	}
	writeOK(w, s.svc.NetworkAvailability())
}

func (s *Server) handleSystemProxyApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, 40501, "method not allowed", nil)
		return
	}
	var req domain.SystemProxyApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, 42201, "invalid json", nil)
		return
	}
	data, err := s.svc.ApplySystemProxy(req.Mode, req.Exceptions)
	if err != nil {
		if err == service.ErrInvalidMode {
			writeError(w, http.StatusUnprocessableEntity, 42201, "invalid mode", nil)
			return
		}
		if errors.Is(err, service.ErrSystemProxyUnsupported) {
			writeError(w, http.StatusUnprocessableEntity, 42201, err.Error(), nil)
			return
		}
		writeError(w, http.StatusInternalServerError, 50001, err.Error(), nil)
		return
	}
	writeOK(w, data)
	s.publishEvent("proxy.changed", data)
}

// ─── Config & routing handlers ────────────────────────────────────────────────

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeOK(w, s.svc.GetConfig())
	case http.MethodPut:
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusUnprocessableEntity, 42201, "invalid json", nil)
			return
		}
		updated := s.svc.UpdateConfig(body)
		writeOK(w, updated)
		s.publishEvent("config.updated", map[string]interface{}{
			"updated": true,
			"config":  updated,
			"status":  s.svc.CoreStatus(),
		})
	default:
		writeError(w, http.StatusMethodNotAllowed, 40501, "method not allowed", nil)
	}
}

func (s *Server) handleRouting(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeOK(w, s.svc.GetRoutingConfig())
	case http.MethodPut:
		var rc domain.RoutingConfig
		if err := json.NewDecoder(r.Body).Decode(&rc); err != nil {
			writeError(w, http.StatusUnprocessableEntity, 42201, "invalid json", nil)
			return
		}
		writeOK(w, s.svc.UpdateRoutingConfig(rc))
		s.publishEvent("routing.updated", map[string]bool{"updated": true})
	default:
		writeError(w, http.StatusMethodNotAllowed, 40501, "method not allowed", nil)
	}
}

// ─── Stats & log handlers ─────────────────────────────────────────────────────

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, 40501, "method not allowed", nil)
		return
	}
	writeOK(w, s.svc.GetStats())
}

// handleLogsStream streams xray core log lines as SSE.
func (s *Server) handleLogsStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, 40501, "method not allowed", nil)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, 50001, "streaming not supported", nil)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	ch, cancel := s.svc.SubscribeCoreLogs()
	defer cancel()

	_, _ = fmt.Fprintf(w, "event: ready\ndata: {\"ok\":true}\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case line, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(line)
			_, _ = fmt.Fprintf(w, "event: log\ndata: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// ─── Metadata events SSE ──────────────────────────────────────────────────────

func (s *Server) handleEventsStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, 40501, "method not allowed", nil)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, 50001, "streaming not supported", nil)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	id, ch := s.addSubscriber()
	defer s.removeSubscriber(id)

	_, _ = fmt.Fprintf(w, "event: ready\ndata: {\"ok\":true}\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-ch:
			_, _ = fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
			flusher.Flush()
		}
	}
}

// ─── SSE event bus ────────────────────────────────────────────────────────────

func (s *Server) addSubscriber() (int, chan []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextSubID
	s.nextSubID++
	ch := make(chan []byte, 16)
	s.subscribers[id] = ch
	return id, ch
}

func (s *Server) removeSubscriber(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ch, ok := s.subscribers[id]; ok {
		delete(s.subscribers, id)
		close(ch)
	}
}

func (s *Server) publishEvent(eventType string, data interface{}) {
	payload, err := json.Marshal(map[string]interface{}{
		"event": eventType,
		"ts":    time.Now().UTC().Format(time.RFC3339),
		"data":  data,
	})
	if err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ch := range s.subscribers {
		select {
		case ch <- payload:
		default:
		}
	}
}

// ─── Response helpers ─────────────────────────────────────────────────────────

func writeOK(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusOK, domain.APIEnvelope{Code: 0, Message: "ok", Data: data})
}

func writeError(w http.ResponseWriter, status, code int, message string, details interface{}) {
	writeJSON(w, status, domain.APIEnvelope{Code: code, Message: message, Details: details})
}

func writeJSON(w http.ResponseWriter, status int, body domain.APIEnvelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// jsonErr writes standardised error responses for service errors.
// Returns true if an error was written.
func jsonErr(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	switch err {
	case service.ErrNotFound:
		writeError(w, http.StatusNotFound, 40401, "not found", nil)
	case service.ErrInvalidMode:
		writeError(w, http.StatusUnprocessableEntity, 42201, "invalid request", nil)
	default:
		writeError(w, http.StatusInternalServerError, 50001, err.Error(), nil)
	}
	return true
}
