package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct{ db *sql.DB }

type Component struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"` // operational, degraded, partial_outage, major_outage, maintenance
	Group       string `json:"group"`
	Position    int    `json:"position"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type Incident struct {
	ID          string           `json:"id"`
	Title       string           `json:"title"`
	Status      string           `json:"status"` // investigating, identified, monitoring, resolved
	Impact      string           `json:"impact"` // none, minor, major, critical
	ComponentID string           `json:"component_id"`
	CreatedAt   string           `json:"created_at"`
	ResolvedAt  string           `json:"resolved_at,omitempty"`
	Updates     []IncidentUpdate `json:"updates,omitempty"`
}

type IncidentUpdate struct {
	ID         string `json:"id"`
	IncidentID string `json:"incident_id"`
	Status     string `json:"status"`
	Body       string `json:"body"`
	CreatedAt  string `json:"created_at"`
}

type Subscriber struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Verified  bool   `json:"verified"`
	CreatedAt string `json:"created_at"`
}

func Open(d string) (*DB, error) {
	if err := os.MkdirAll(d, 0755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(d, "paddock.db")+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}

	db.Exec(`CREATE TABLE IF NOT EXISTS components(
		id TEXT PRIMARY KEY, name TEXT NOT NULL, description TEXT DEFAULT '',
		status TEXT DEFAULT 'operational', grp TEXT DEFAULT '',
		position INTEGER DEFAULT 0,
		created_at TEXT DEFAULT(datetime('now')),
		updated_at TEXT DEFAULT(datetime('now')))`)

	db.Exec(`CREATE TABLE IF NOT EXISTS incidents(
		id TEXT PRIMARY KEY, title TEXT NOT NULL,
		status TEXT DEFAULT 'investigating', impact TEXT DEFAULT 'minor',
		component_id TEXT DEFAULT '',
		created_at TEXT DEFAULT(datetime('now')),
		resolved_at TEXT DEFAULT '')`)

	db.Exec(`CREATE TABLE IF NOT EXISTS incident_updates(
		id TEXT PRIMARY KEY, incident_id TEXT NOT NULL,
		status TEXT NOT NULL, body TEXT DEFAULT '',
		created_at TEXT DEFAULT(datetime('now')),
		FOREIGN KEY(incident_id) REFERENCES incidents(id))`)

	db.Exec(`CREATE TABLE IF NOT EXISTS subscribers(
		id TEXT PRIMARY KEY, email TEXT UNIQUE NOT NULL,
		verified INTEGER DEFAULT 0,
		created_at TEXT DEFAULT(datetime('now')))`)

	return &DB{db: db}, nil
}

func (d *DB) Close() error { return d.db.Close() }
func genID() string        { return fmt.Sprintf("%d", time.Now().UnixNano()) }
func now() string          { return time.Now().UTC().Format(time.RFC3339) }

// ── Components ──

func (d *DB) CreateComponent(c *Component) error {
	c.ID = genID()
	c.CreatedAt = now()
	c.UpdatedAt = c.CreatedAt
	if c.Status == "" {
		c.Status = "operational"
	}
	_, err := d.db.Exec(`INSERT INTO components(id,name,description,status,grp,position,created_at,updated_at)VALUES(?,?,?,?,?,?,?,?)`,
		c.ID, c.Name, c.Description, c.Status, c.Group, c.Position, c.CreatedAt, c.UpdatedAt)
	return err
}

func (d *DB) GetComponent(id string) *Component {
	var c Component
	if d.db.QueryRow(`SELECT id,name,description,status,grp,position,created_at,updated_at FROM components WHERE id=?`, id).
		Scan(&c.ID, &c.Name, &c.Description, &c.Status, &c.Group, &c.Position, &c.CreatedAt, &c.UpdatedAt) != nil {
		return nil
	}
	return &c
}

func (d *DB) ListComponents() []Component {
	rows, _ := d.db.Query(`SELECT id,name,description,status,grp,position,created_at,updated_at FROM components ORDER BY position ASC, created_at ASC`)
	if rows == nil {
		return []Component{}
	}
	defer rows.Close()
	var out []Component
	for rows.Next() {
		var c Component
		rows.Scan(&c.ID, &c.Name, &c.Description, &c.Status, &c.Group, &c.Position, &c.CreatedAt, &c.UpdatedAt)
		out = append(out, c)
	}
	if out == nil {
		return []Component{}
	}
	return out
}

func (d *DB) UpdateComponent(c *Component) error {
	c.UpdatedAt = now()
	_, err := d.db.Exec(`UPDATE components SET name=?,description=?,status=?,grp=?,position=?,updated_at=? WHERE id=?`,
		c.Name, c.Description, c.Status, c.Group, c.Position, c.UpdatedAt, c.ID)
	return err
}

func (d *DB) DeleteComponent(id string) error {
	_, err := d.db.Exec(`DELETE FROM components WHERE id=?`, id)
	return err
}

// ── Incidents ──

func (d *DB) CreateIncident(inc *Incident) error {
	inc.ID = genID()
	inc.CreatedAt = now()
	if inc.Status == "" {
		inc.Status = "investigating"
	}
	if inc.Impact == "" {
		inc.Impact = "minor"
	}
	_, err := d.db.Exec(`INSERT INTO incidents(id,title,status,impact,component_id,created_at)VALUES(?,?,?,?,?,?)`,
		inc.ID, inc.Title, inc.Status, inc.Impact, inc.ComponentID, inc.CreatedAt)
	return err
}

func (d *DB) GetIncident(id string) *Incident {
	var inc Incident
	if d.db.QueryRow(`SELECT id,title,status,impact,component_id,created_at,resolved_at FROM incidents WHERE id=?`, id).
		Scan(&inc.ID, &inc.Title, &inc.Status, &inc.Impact, &inc.ComponentID, &inc.CreatedAt, &inc.ResolvedAt) != nil {
		return nil
	}
	inc.Updates = d.ListIncidentUpdates(id)
	return &inc
}

func (d *DB) ListIncidents(activeOnly bool) []Incident {
	q := `SELECT id,title,status,impact,component_id,created_at,resolved_at FROM incidents`
	if activeOnly {
		q += ` WHERE status != 'resolved'`
	}
	q += ` ORDER BY created_at DESC`
	rows, _ := d.db.Query(q)
	if rows == nil {
		return []Incident{}
	}
	defer rows.Close()
	var out []Incident
	for rows.Next() {
		var inc Incident
		rows.Scan(&inc.ID, &inc.Title, &inc.Status, &inc.Impact, &inc.ComponentID, &inc.CreatedAt, &inc.ResolvedAt)
		inc.Updates = d.ListIncidentUpdates(inc.ID)
		out = append(out, inc)
	}
	if out == nil {
		return []Incident{}
	}
	return out
}

func (d *DB) UpdateIncident(inc *Incident) error {
	if inc.Status == "resolved" && inc.ResolvedAt == "" {
		inc.ResolvedAt = now()
	}
	_, err := d.db.Exec(`UPDATE incidents SET title=?,status=?,impact=?,component_id=?,resolved_at=? WHERE id=?`,
		inc.Title, inc.Status, inc.Impact, inc.ComponentID, inc.ResolvedAt, inc.ID)
	return err
}

func (d *DB) DeleteIncident(id string) error {
	d.db.Exec(`DELETE FROM incident_updates WHERE incident_id=?`, id)
	_, err := d.db.Exec(`DELETE FROM incidents WHERE id=?`, id)
	return err
}

// ── Incident Updates ──

func (d *DB) CreateIncidentUpdate(u *IncidentUpdate) error {
	u.ID = genID()
	u.CreatedAt = now()
	_, err := d.db.Exec(`INSERT INTO incident_updates(id,incident_id,status,body,created_at)VALUES(?,?,?,?,?)`,
		u.ID, u.IncidentID, u.Status, u.Body, u.CreatedAt)
	// Also update the parent incident status
	if err == nil {
		d.db.Exec(`UPDATE incidents SET status=? WHERE id=?`, u.Status, u.IncidentID)
		if u.Status == "resolved" {
			d.db.Exec(`UPDATE incidents SET resolved_at=? WHERE id=?`, now(), u.IncidentID)
		}
	}
	return err
}

func (d *DB) ListIncidentUpdates(incidentID string) []IncidentUpdate {
	rows, _ := d.db.Query(`SELECT id,incident_id,status,body,created_at FROM incident_updates WHERE incident_id=? ORDER BY created_at ASC`, incidentID)
	if rows == nil {
		return []IncidentUpdate{}
	}
	defer rows.Close()
	var out []IncidentUpdate
	for rows.Next() {
		var u IncidentUpdate
		rows.Scan(&u.ID, &u.IncidentID, &u.Status, &u.Body, &u.CreatedAt)
		out = append(out, u)
	}
	if out == nil {
		return []IncidentUpdate{}
	}
	return out
}

// ── Subscribers ──

func (d *DB) Subscribe(email string) error {
	_, err := d.db.Exec(`INSERT OR IGNORE INTO subscribers(id,email,created_at)VALUES(?,?,?)`, genID(), email, now())
	return err
}

func (d *DB) ListSubscribers() []Subscriber {
	rows, _ := d.db.Query(`SELECT id,email,verified,created_at FROM subscribers ORDER BY created_at DESC`)
	if rows == nil {
		return []Subscriber{}
	}
	defer rows.Close()
	var out []Subscriber
	for rows.Next() {
		var s Subscriber
		rows.Scan(&s.ID, &s.Email, &s.Verified, &s.CreatedAt)
		out = append(out, s)
	}
	if out == nil {
		return []Subscriber{}
	}
	return out
}

func (d *DB) Unsubscribe(email string) error {
	_, err := d.db.Exec(`DELETE FROM subscribers WHERE email=?`, email)
	return err
}

// ── Stats ──

func (d *DB) Stats() map[string]any {
	var components, incidents, subscribers int
	d.db.QueryRow(`SELECT COUNT(*) FROM components`).Scan(&components)
	d.db.QueryRow(`SELECT COUNT(*) FROM incidents WHERE status!='resolved'`).Scan(&incidents)
	d.db.QueryRow(`SELECT COUNT(*) FROM subscribers`).Scan(&subscribers)

	// Overall status
	overall := "operational"
	rows, _ := d.db.Query(`SELECT status FROM components`)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var s string
			rows.Scan(&s)
			switch s {
			case "major_outage":
				overall = "major_outage"
			case "partial_outage":
				if overall != "major_outage" {
					overall = "partial_outage"
				}
			case "degraded":
				if overall == "operational" {
					overall = "degraded"
				}
			case "maintenance":
				if overall == "operational" {
					overall = "maintenance"
				}
			}
		}
	}

	return map[string]any{
		"components":       components,
		"active_incidents": incidents,
		"subscribers":      subscribers,
		"overall_status":   overall,
	}
}
