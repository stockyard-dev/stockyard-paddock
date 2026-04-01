package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct{ conn *sql.DB }

func Open(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	conn, err := sql.Open("sqlite", filepath.Join(dataDir, "paddock.db"))
	if err != nil {
		return nil, err
	}
	conn.Exec("PRAGMA journal_mode=WAL")
	conn.Exec("PRAGMA busy_timeout=5000")
	conn.SetMaxOpenConns(4)
	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, err
	}
	return db, nil
}

func (db *DB) Conn() *sql.DB { return db.conn }
func (db *DB) Close() error  { return db.conn.Close() }

func (db *DB) migrate() error {
	_, err := db.conn.Exec(`
CREATE TABLE IF NOT EXISTS monitors (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    method TEXT DEFAULT 'GET',
    interval_seconds INTEGER DEFAULT 300,
    timeout_ms INTEGER DEFAULT 10000,
    expected_status INTEGER DEFAULT 200,
    headers_json TEXT DEFAULT '{}',
    enabled INTEGER DEFAULT 1,
    last_checked_at TEXT DEFAULT '',
    last_status TEXT DEFAULT 'unknown',
    last_response_ms INTEGER DEFAULT 0,
    created_at TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS checks (
    id TEXT PRIMARY KEY,
    monitor_id TEXT NOT NULL,
    status TEXT NOT NULL,
    response_ms INTEGER DEFAULT 0,
    status_code INTEGER DEFAULT 0,
    error TEXT DEFAULT '',
    checked_at TEXT DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_checks_monitor ON checks(monitor_id);
CREATE INDEX IF NOT EXISTS idx_checks_time ON checks(checked_at);

CREATE TABLE IF NOT EXISTS alerts (
    id TEXT PRIMARY KEY,
    monitor_id TEXT NOT NULL,
    webhook_url TEXT NOT NULL,
    on_down INTEGER DEFAULT 1,
    on_recovery INTEGER DEFAULT 1,
    enabled INTEGER DEFAULT 1,
    created_at TEXT DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_alerts_monitor ON alerts(monitor_id);

CREATE TABLE IF NOT EXISTS incidents (
    id TEXT PRIMARY KEY,
    monitor_id TEXT NOT NULL,
    started_at TEXT NOT NULL,
    ended_at TEXT DEFAULT '',
    duration_seconds INTEGER DEFAULT 0,
    checks_failed INTEGER DEFAULT 1
);
CREATE INDEX IF NOT EXISTS idx_incidents_monitor ON incidents(monitor_id);
`)
	return err
}

// --- Monitor types ---

type Monitor struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	URL            string `json:"url"`
	Method         string `json:"method"`
	IntervalSec    int    `json:"interval_seconds"`
	TimeoutMs      int    `json:"timeout_ms"`
	ExpectedStatus int    `json:"expected_status"`
	HeadersJSON    string `json:"headers_json,omitempty"`
	Enabled        bool   `json:"enabled"`
	LastCheckedAt  string `json:"last_checked_at"`
	LastStatus     string `json:"last_status"`
	LastResponseMs int    `json:"last_response_ms"`
	CreatedAt      string `json:"created_at"`
}

func (db *DB) CreateMonitor(name, url, method string, intervalSec, timeoutMs, expectedStatus int) (*Monitor, error) {
	id := "mon_" + genID(8)
	now := time.Now().UTC().Format(time.RFC3339)
	if method == "" {
		method = "GET"
	}
	if intervalSec <= 0 {
		intervalSec = 300
	}
	if timeoutMs <= 0 {
		timeoutMs = 10000
	}
	if expectedStatus <= 0 {
		expectedStatus = 200
	}
	_, err := db.conn.Exec(`INSERT INTO monitors (id,name,url,method,interval_seconds,timeout_ms,expected_status,created_at)
		VALUES (?,?,?,?,?,?,?,?)`, id, name, url, method, intervalSec, timeoutMs, expectedStatus, now)
	if err != nil {
		return nil, err
	}
	return &Monitor{ID: id, Name: name, URL: url, Method: method, IntervalSec: intervalSec,
		TimeoutMs: timeoutMs, ExpectedStatus: expectedStatus, Enabled: true,
		LastStatus: "unknown", CreatedAt: now}, nil
}

func (db *DB) ListMonitors() ([]Monitor, error) {
	rows, err := db.conn.Query(`SELECT id,name,url,method,interval_seconds,timeout_ms,expected_status,
		headers_json,enabled,last_checked_at,last_status,last_response_ms,created_at
		FROM monitors ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Monitor
	for rows.Next() {
		var m Monitor
		var en int
		if err := rows.Scan(&m.ID, &m.Name, &m.URL, &m.Method, &m.IntervalSec, &m.TimeoutMs,
			&m.ExpectedStatus, &m.HeadersJSON, &en, &m.LastCheckedAt, &m.LastStatus,
			&m.LastResponseMs, &m.CreatedAt); err != nil {
			continue
		}
		m.Enabled = en == 1
		out = append(out, m)
	}
	return out, rows.Err()
}

func (db *DB) GetMonitor(id string) (*Monitor, error) {
	var m Monitor
	var en int
	err := db.conn.QueryRow(`SELECT id,name,url,method,interval_seconds,timeout_ms,expected_status,
		headers_json,enabled,last_checked_at,last_status,last_response_ms,created_at
		FROM monitors WHERE id=?`, id).
		Scan(&m.ID, &m.Name, &m.URL, &m.Method, &m.IntervalSec, &m.TimeoutMs,
			&m.ExpectedStatus, &m.HeadersJSON, &en, &m.LastCheckedAt, &m.LastStatus,
			&m.LastResponseMs, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	m.Enabled = en == 1
	return &m, nil
}

func (db *DB) UpdateMonitor(id string, name, url, method *string, intervalSec, timeoutMs, expectedStatus *int, enabled *bool) (*Monitor, error) {
	if name != nil {
		db.conn.Exec("UPDATE monitors SET name=? WHERE id=?", *name, id)
	}
	if url != nil {
		db.conn.Exec("UPDATE monitors SET url=? WHERE id=?", *url, id)
	}
	if method != nil {
		db.conn.Exec("UPDATE monitors SET method=? WHERE id=?", *method, id)
	}
	if intervalSec != nil {
		db.conn.Exec("UPDATE monitors SET interval_seconds=? WHERE id=?", *intervalSec, id)
	}
	if timeoutMs != nil {
		db.conn.Exec("UPDATE monitors SET timeout_ms=? WHERE id=?", *timeoutMs, id)
	}
	if expectedStatus != nil {
		db.conn.Exec("UPDATE monitors SET expected_status=? WHERE id=?", *expectedStatus, id)
	}
	if enabled != nil {
		en := 0
		if *enabled {
			en = 1
		}
		db.conn.Exec("UPDATE monitors SET enabled=? WHERE id=?", en, id)
	}
	return db.GetMonitor(id)
}

func (db *DB) DeleteMonitor(id string) error {
	db.conn.Exec("DELETE FROM checks WHERE monitor_id=?", id)
	db.conn.Exec("DELETE FROM alerts WHERE monitor_id=?", id)
	db.conn.Exec("DELETE FROM incidents WHERE monitor_id=?", id)
	_, err := db.conn.Exec("DELETE FROM monitors WHERE id=?", id)
	return err
}

func (db *DB) UpdateMonitorStatus(id, status string, responseMs int) {
	now := time.Now().UTC().Format(time.RFC3339)
	db.conn.Exec("UPDATE monitors SET last_checked_at=?, last_status=?, last_response_ms=? WHERE id=?",
		now, status, responseMs, id)
}

// --- Checks ---

type Check struct {
	ID         string `json:"id"`
	MonitorID  string `json:"monitor_id"`
	Status     string `json:"status"`
	ResponseMs int    `json:"response_ms"`
	StatusCode int    `json:"status_code"`
	Error      string `json:"error,omitempty"`
	CheckedAt  string `json:"checked_at"`
}

func (db *DB) RecordCheck(monitorID, status string, responseMs, statusCode int, errMsg string) (*Check, error) {
	id := "chk_" + genID(10)
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.conn.Exec(`INSERT INTO checks (id,monitor_id,status,response_ms,status_code,error,checked_at)
		VALUES (?,?,?,?,?,?,?)`, id, monitorID, status, responseMs, statusCode, errMsg, now)
	if err != nil {
		return nil, err
	}
	return &Check{ID: id, MonitorID: monitorID, Status: status, ResponseMs: responseMs,
		StatusCode: statusCode, Error: errMsg, CheckedAt: now}, nil
}

func (db *DB) ListChecks(monitorID string, limit int) ([]Check, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := db.conn.Query(`SELECT id,monitor_id,status,response_ms,status_code,error,checked_at
		FROM checks WHERE monitor_id=? ORDER BY checked_at DESC LIMIT ?`, monitorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Check
	for rows.Next() {
		var c Check
		rows.Scan(&c.ID, &c.MonitorID, &c.Status, &c.ResponseMs, &c.StatusCode, &c.Error, &c.CheckedAt)
		out = append(out, c)
	}
	return out, rows.Err()
}

// --- Alerts ---

type Alert struct {
	ID         string `json:"id"`
	MonitorID  string `json:"monitor_id"`
	WebhookURL string `json:"webhook_url"`
	OnDown     bool   `json:"on_down"`
	OnRecovery bool   `json:"on_recovery"`
	Enabled    bool   `json:"enabled"`
	CreatedAt  string `json:"created_at"`
}

func (db *DB) CreateAlert(monitorID, webhookURL string, onDown, onRecovery bool) (*Alert, error) {
	id := "alt_" + genID(8)
	d, r := 0, 0
	if onDown {
		d = 1
	}
	if onRecovery {
		r = 1
	}
	_, err := db.conn.Exec(`INSERT INTO alerts (id,monitor_id,webhook_url,on_down,on_recovery)
		VALUES (?,?,?,?,?)`, id, monitorID, webhookURL, d, r)
	if err != nil {
		return nil, err
	}
	return &Alert{ID: id, MonitorID: monitorID, WebhookURL: webhookURL,
		OnDown: onDown, OnRecovery: onRecovery, Enabled: true}, nil
}

func (db *DB) ListAlerts(monitorID string) ([]Alert, error) {
	rows, err := db.conn.Query(`SELECT id,monitor_id,webhook_url,on_down,on_recovery,enabled,created_at
		FROM alerts WHERE monitor_id=? AND enabled=1`, monitorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Alert
	for rows.Next() {
		var a Alert
		var d, r, en int
		rows.Scan(&a.ID, &a.MonitorID, &a.WebhookURL, &d, &r, &en, &a.CreatedAt)
		a.OnDown = d == 1
		a.OnRecovery = r == 1
		a.Enabled = en == 1
		out = append(out, a)
	}
	return out, rows.Err()
}

func (db *DB) DeleteAlert(id string) error {
	_, err := db.conn.Exec("DELETE FROM alerts WHERE id=?", id)
	return err
}

// --- Incidents ---

type Incident struct {
	ID              string `json:"id"`
	MonitorID       string `json:"monitor_id"`
	StartedAt       string `json:"started_at"`
	EndedAt         string `json:"ended_at,omitempty"`
	DurationSeconds int    `json:"duration_seconds"`
	ChecksFailed    int    `json:"checks_failed"`
}

func (db *DB) OpenIncident(monitorID string) (*Incident, error) {
	id := "inc_" + genID(8)
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.conn.Exec(`INSERT INTO incidents (id,monitor_id,started_at) VALUES (?,?,?)`, id, monitorID, now)
	if err != nil {
		return nil, err
	}
	return &Incident{ID: id, MonitorID: monitorID, StartedAt: now, ChecksFailed: 1}, nil
}

func (db *DB) ActiveIncident(monitorID string) (*Incident, error) {
	var inc Incident
	err := db.conn.QueryRow(`SELECT id,monitor_id,started_at,ended_at,duration_seconds,checks_failed
		FROM incidents WHERE monitor_id=? AND ended_at='' ORDER BY started_at DESC LIMIT 1`, monitorID).
		Scan(&inc.ID, &inc.MonitorID, &inc.StartedAt, &inc.EndedAt, &inc.DurationSeconds, &inc.ChecksFailed)
	if err != nil {
		return nil, err
	}
	return &inc, nil
}

func (db *DB) IncrementIncident(id string) {
	db.conn.Exec("UPDATE incidents SET checks_failed=checks_failed+1 WHERE id=?", id)
}

func (db *DB) CloseIncident(id string) {
	now := time.Now().UTC().Format(time.RFC3339)
	db.conn.Exec("UPDATE incidents SET ended_at=? WHERE id=?", now, id)
	// Calculate duration
	var startedAt string
	db.conn.QueryRow("SELECT started_at FROM incidents WHERE id=?", id).Scan(&startedAt)
	if s, err := time.Parse(time.RFC3339, startedAt); err == nil {
		dur := int(time.Since(s).Seconds())
		db.conn.Exec("UPDATE incidents SET duration_seconds=? WHERE id=?", dur, id)
	}
}

// --- Stats ---

func (db *DB) Stats() map[string]any {
	var monitors, checks, up, down int
	db.conn.QueryRow("SELECT COUNT(*) FROM monitors").Scan(&monitors)
	db.conn.QueryRow("SELECT COUNT(*) FROM checks").Scan(&checks)
	db.conn.QueryRow("SELECT COUNT(*) FROM monitors WHERE last_status='up'").Scan(&up)
	db.conn.QueryRow("SELECT COUNT(*) FROM monitors WHERE last_status='down'").Scan(&down)
	var incidents int
	db.conn.QueryRow("SELECT COUNT(*) FROM incidents WHERE ended_at=''").Scan(&incidents)
	return map[string]any{"monitors": monitors, "checks": checks, "up": up, "down": down, "active_incidents": incidents}
}

func (db *DB) UptimePercent(monitorID string, days int) float64 {
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02 15:04:05")
	var total, upCount int
	db.conn.QueryRow("SELECT COUNT(*) FROM checks WHERE monitor_id=? AND checked_at>=?", monitorID, cutoff).Scan(&total)
	db.conn.QueryRow("SELECT COUNT(*) FROM checks WHERE monitor_id=? AND checked_at>=? AND status='up'", monitorID, cutoff).Scan(&upCount)
	if total == 0 {
		return 100.0
	}
	return float64(upCount) / float64(total) * 100
}

func (db *DB) Cleanup(days int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02 15:04:05")
	res, err := db.conn.Exec("DELETE FROM checks WHERE checked_at < ?", cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// --- Status page data ---

type StatusEntry struct {
	Name       string  `json:"name"`
	URL        string  `json:"url"`
	Status     string  `json:"status"`
	ResponseMs int     `json:"response_ms"`
	Uptime24h  float64 `json:"uptime_24h"`
	Uptime7d   float64 `json:"uptime_7d"`
	Uptime30d  float64 `json:"uptime_30d"`
	LastCheck  string  `json:"last_checked_at"`
}

func (db *DB) StatusPageData() ([]StatusEntry, error) {
	monitors, err := db.ListMonitors()
	if err != nil {
		return nil, err
	}
	var entries []StatusEntry
	for _, m := range monitors {
		if !m.Enabled {
			continue
		}
		entries = append(entries, StatusEntry{
			Name:       m.Name,
			URL:        m.URL,
			Status:     m.LastStatus,
			ResponseMs: m.LastResponseMs,
			Uptime24h:  db.UptimePercent(m.ID, 1),
			Uptime7d:   db.UptimePercent(m.ID, 7),
			Uptime30d:  db.UptimePercent(m.ID, 30),
			LastCheck:  m.LastCheckedAt,
		})
	}
	return entries, nil
}

func genID(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
