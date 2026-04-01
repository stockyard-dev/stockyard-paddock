package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/stockyard-dev/stockyard-paddock/internal/store"
)

type Server struct {
	db     *store.DB
	mux    *http.ServeMux
	port   int
	limits Limits
	client *http.Client

	mu       sync.Mutex
	stopChs  map[string]chan struct{} // monitor_id → stop channel
}

func New(db *store.DB, port int, limits Limits) *Server {
	s := &Server{
		db:      db,
		mux:     http.NewServeMux(),
		port:    port,
		limits:  limits,
		client:  &http.Client{Timeout: 30 * time.Second},
		stopChs: make(map[string]chan struct{}),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	// Monitors
	s.mux.HandleFunc("POST /api/monitors", s.handleCreateMonitor)
	s.mux.HandleFunc("GET /api/monitors", s.handleListMonitors)
	s.mux.HandleFunc("GET /api/monitors/{id}", s.handleGetMonitor)
	s.mux.HandleFunc("PUT /api/monitors/{id}", s.handleUpdateMonitor)
	s.mux.HandleFunc("DELETE /api/monitors/{id}", s.handleDeleteMonitor)

	// Check history
	s.mux.HandleFunc("GET /api/monitors/{id}/history", s.handleCheckHistory)

	// Alerts
	s.mux.HandleFunc("POST /api/monitors/{id}/alerts", s.handleCreateAlert)
	s.mux.HandleFunc("GET /api/monitors/{id}/alerts", s.handleListAlerts)
	s.mux.HandleFunc("DELETE /api/alerts/{id}", s.handleDeleteAlert)

	// Status page
	s.mux.HandleFunc("GET /api/status", s.handleStatusAPI)
	s.mux.HandleFunc("GET /status", s.handleStatusPage)

	// Stats
	s.mux.HandleFunc("GET /api/stats", s.handleStats)

	// Health + UI
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /ui", s.handleUI)

	// Version
	s.mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"product": "stockyard-paddock", "version": "0.1.0"})
	})
}

func (s *Server) Start() error {
	// Start monitor goroutines for all existing monitors
	s.startAllMonitors()

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("[paddock] listening on %s", addr)
	return http.ListenAndServe(addr, s.mux)
}

// --- Monitor loop ---

func (s *Server) startAllMonitors() {
	monitors, err := s.db.ListMonitors()
	if err != nil {
		log.Printf("[paddock] error loading monitors: %v", err)
		return
	}
	for _, m := range monitors {
		if m.Enabled {
			s.startMonitorLoop(m.ID)
		}
	}
	log.Printf("[paddock] started %d monitor(s)", len(monitors))
}

func (s *Server) startMonitorLoop(monitorID string) {
	s.mu.Lock()
	if _, exists := s.stopChs[monitorID]; exists {
		s.mu.Unlock()
		return
	}
	stop := make(chan struct{})
	s.stopChs[monitorID] = stop
	s.mu.Unlock()

	go func() {
		// Initial check after 5 seconds
		select {
		case <-time.After(5 * time.Second):
		case <-stop:
			return
		}

		for {
			mon, err := s.db.GetMonitor(monitorID)
			if err != nil || !mon.Enabled {
				s.mu.Lock()
				delete(s.stopChs, monitorID)
				s.mu.Unlock()
				return
			}

			s.runCheck(mon)

			interval := time.Duration(mon.IntervalSec) * time.Second
			select {
			case <-time.After(interval):
			case <-stop:
				return
			}
		}
	}()
}

func (s *Server) stopMonitorLoop(monitorID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ch, ok := s.stopChs[monitorID]; ok {
		close(ch)
		delete(s.stopChs, monitorID)
	}
}

func (s *Server) runCheck(mon *store.Monitor) {
	client := &http.Client{
		Timeout: time.Duration(mon.TimeoutMs) * time.Millisecond,
	}

	req, err := http.NewRequest(mon.Method, mon.URL, nil)
	if err != nil {
		s.recordResult(mon, "down", 0, 0, err.Error())
		return
	}
	req.Header.Set("User-Agent", "Stockyard-Paddock/0.1")

	start := time.Now()
	resp, err := client.Do(req)
	responseMs := int(time.Since(start).Milliseconds())

	if err != nil {
		s.recordResult(mon, "down", responseMs, 0, err.Error())
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	status := "up"
	if resp.StatusCode != mon.ExpectedStatus {
		status = "down"
	}

	s.recordResult(mon, status, responseMs, resp.StatusCode, "")
}

func (s *Server) recordResult(mon *store.Monitor, status string, responseMs, statusCode int, errMsg string) {
	prevStatus := mon.LastStatus

	s.db.RecordCheck(mon.ID, status, responseMs, statusCode, errMsg)
	s.db.UpdateMonitorStatus(mon.ID, status, responseMs)

	if status == "up" {
		log.Printf("[check] %s %s — UP (%dms)", mon.Name, mon.URL, responseMs)
	} else {
		log.Printf("[check] %s %s — DOWN (%s)", mon.Name, mon.URL, errMsg)
	}

	// Detect status transitions
	if prevStatus == "up" && status == "down" {
		// Just went down — open incident
		s.db.OpenIncident(mon.ID)
		s.fireAlerts(mon, "down", errMsg)
	} else if prevStatus == "down" && status == "down" {
		// Still down — increment incident
		if inc, err := s.db.ActiveIncident(mon.ID); err == nil {
			s.db.IncrementIncident(inc.ID)
		}
	} else if prevStatus == "down" && status == "up" {
		// Recovery
		if inc, err := s.db.ActiveIncident(mon.ID); err == nil {
			s.db.CloseIncident(inc.ID)
		}
		s.fireAlerts(mon, "recovery", "")
	}
}

func (s *Server) fireAlerts(mon *store.Monitor, event, errMsg string) {
	if !s.limits.AlertWebhooks {
		return
	}
	alerts, err := s.db.ListAlerts(mon.ID)
	if err != nil || len(alerts) == 0 {
		return
	}

	payload := map[string]any{
		"event":     event,
		"monitor":   mon.Name,
		"url":       mon.URL,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	if errMsg != "" {
		payload["error"] = errMsg
	}
	body, _ := json.Marshal(payload)

	for _, alert := range alerts {
		if event == "down" && !alert.OnDown {
			continue
		}
		if event == "recovery" && !alert.OnRecovery {
			continue
		}
		go func(url string) {
			req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Paddock-Event", event)
			resp, err := s.client.Do(req)
			if err != nil {
				log.Printf("[alert] error sending to %s: %v", url, err)
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}(alert.WebhookURL)
	}
}

// --- Monitor handlers ---

func (s *Server) handleCreateMonitor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name           string `json:"name"`
		URL            string `json:"url"`
		Method         string `json:"method"`
		IntervalSec    int    `json:"interval_seconds"`
		TimeoutMs      int    `json:"timeout_ms"`
		ExpectedStatus int    `json:"expected_status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.URL == "" {
		writeJSON(w, 400, map[string]string{"error": "url is required"})
		return
	}
	if req.Name == "" {
		req.Name = req.URL
	}

	// Check monitor limit
	if s.limits.MaxMonitors > 0 {
		monitors, _ := s.db.ListMonitors()
		if LimitReached(s.limits.MaxMonitors, len(monitors)) {
			writeJSON(w, 402, map[string]string{
				"error":   fmt.Sprintf("free tier limit: %d monitors max — upgrade to Pro", s.limits.MaxMonitors),
				"upgrade": "https://stockyard.dev/paddock/",
			})
			return
		}
	}

	// Enforce minimum interval
	if req.IntervalSec > 0 && req.IntervalSec < s.limits.MinIntervalSec {
		writeJSON(w, 402, map[string]string{
			"error":   fmt.Sprintf("minimum interval is %ds on free tier — upgrade to Pro for %ds", s.limits.MinIntervalSec, proLimits.MinIntervalSec),
			"upgrade": "https://stockyard.dev/paddock/",
		})
		return
	}

	mon, err := s.db.CreateMonitor(req.Name, req.URL, req.Method, req.IntervalSec, req.TimeoutMs, req.ExpectedStatus)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	// Start monitoring
	s.startMonitorLoop(mon.ID)

	writeJSON(w, 201, map[string]any{"monitor": mon})
}

func (s *Server) handleListMonitors(w http.ResponseWriter, r *http.Request) {
	monitors, err := s.db.ListMonitors()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if monitors == nil {
		monitors = []store.Monitor{}
	}
	writeJSON(w, 200, map[string]any{"monitors": monitors, "count": len(monitors)})
}

func (s *Server) handleGetMonitor(w http.ResponseWriter, r *http.Request) {
	mon, err := s.db.GetMonitor(r.PathValue("id"))
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "monitor not found"})
		return
	}
	uptime := s.db.UptimePercent(mon.ID, 30)
	checks, _ := s.db.ListChecks(mon.ID, 20)
	if checks == nil {
		checks = []store.Check{}
	}
	writeJSON(w, 200, map[string]any{"monitor": mon, "uptime_30d": uptime, "recent_checks": checks})
}

func (s *Server) handleUpdateMonitor(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.db.GetMonitor(id); err != nil {
		writeJSON(w, 404, map[string]string{"error": "monitor not found"})
		return
	}

	var req struct {
		Name           *string `json:"name"`
		URL            *string `json:"url"`
		Method         *string `json:"method"`
		IntervalSec    *int    `json:"interval_seconds"`
		TimeoutMs      *int    `json:"timeout_ms"`
		ExpectedStatus *int    `json:"expected_status"`
		Enabled        *bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}

	if req.IntervalSec != nil && *req.IntervalSec < s.limits.MinIntervalSec {
		writeJSON(w, 402, map[string]string{
			"error":   fmt.Sprintf("minimum interval is %ds — upgrade to Pro for %ds", s.limits.MinIntervalSec, proLimits.MinIntervalSec),
			"upgrade": "https://stockyard.dev/paddock/",
		})
		return
	}

	mon, err := s.db.UpdateMonitor(id, req.Name, req.URL, req.Method, req.IntervalSec, req.TimeoutMs, req.ExpectedStatus, req.Enabled)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	// Restart or stop monitor loop based on enabled state
	s.stopMonitorLoop(id)
	if mon.Enabled {
		s.startMonitorLoop(id)
	}

	writeJSON(w, 200, map[string]any{"monitor": mon})
}

func (s *Server) handleDeleteMonitor(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.db.GetMonitor(id); err != nil {
		writeJSON(w, 404, map[string]string{"error": "monitor not found"})
		return
	}
	s.stopMonitorLoop(id)
	s.db.DeleteMonitor(id)
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

// --- Check history ---

func (s *Server) handleCheckHistory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil {
			limit = n
		}
	}
	checks, err := s.db.ListChecks(id, limit)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if checks == nil {
		checks = []store.Check{}
	}
	writeJSON(w, 200, map[string]any{"checks": checks, "count": len(checks)})
}

// --- Alerts ---

func (s *Server) handleCreateAlert(w http.ResponseWriter, r *http.Request) {
	if !s.limits.AlertWebhooks {
		writeJSON(w, 402, map[string]string{
			"error":   "alert webhooks require Pro — upgrade at https://stockyard.dev/paddock/",
			"upgrade": "https://stockyard.dev/paddock/",
		})
		return
	}
	monID := r.PathValue("id")
	if _, err := s.db.GetMonitor(monID); err != nil {
		writeJSON(w, 404, map[string]string{"error": "monitor not found"})
		return
	}
	var req struct {
		WebhookURL string `json:"webhook_url"`
		OnDown     *bool  `json:"on_down"`
		OnRecovery *bool  `json:"on_recovery"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.WebhookURL == "" {
		writeJSON(w, 400, map[string]string{"error": "webhook_url is required"})
		return
	}
	onDown := true
	if req.OnDown != nil {
		onDown = *req.OnDown
	}
	onRecovery := true
	if req.OnRecovery != nil {
		onRecovery = *req.OnRecovery
	}
	alert, err := s.db.CreateAlert(monID, req.WebhookURL, onDown, onRecovery)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 201, map[string]any{"alert": alert})
}

func (s *Server) handleListAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := s.db.ListAlerts(r.PathValue("id"))
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if alerts == nil {
		alerts = []store.Alert{}
	}
	writeJSON(w, 200, map[string]any{"alerts": alerts})
}

func (s *Server) handleDeleteAlert(w http.ResponseWriter, r *http.Request) {
	s.db.DeleteAlert(r.PathValue("id"))
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

// --- Status page ---

func (s *Server) handleStatusAPI(w http.ResponseWriter, r *http.Request) {
	entries, err := s.db.StatusPageData()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if entries == nil {
		entries = []store.StatusEntry{}
	}
	allUp := true
	for _, e := range entries {
		if e.Status != "up" {
			allUp = false
			break
		}
	}
	overall := "operational"
	if !allUp {
		overall = "degraded"
	}
	if len(entries) == 0 {
		overall = "no_monitors"
	}
	writeJSON(w, 200, map[string]any{"status": overall, "services": entries, "checked_at": time.Now().UTC().Format(time.RFC3339)})
}

func (s *Server) handleStatusPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(statusPageHTML))
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.db.Stats())
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func itoa(n int) string { return strconv.Itoa(n) }

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
