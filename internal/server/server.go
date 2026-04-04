package server

import (
	"encoding/json"
	"net/http"

	"github.com/stockyard-dev/stockyard-paddock/internal/store"
)

type Server struct {
	db     *store.DB
	mux    *http.ServeMux
	limits Limits
}

func New(db *store.DB, limits Limits) *Server {
	s := &Server{db: db, mux: http.NewServeMux(), limits: limits}

	// Components
	s.mux.HandleFunc("GET /api/components", s.listComponents)
	s.mux.HandleFunc("POST /api/components", s.createComponent)
	s.mux.HandleFunc("GET /api/components/{id}", s.getComponent)
	s.mux.HandleFunc("PUT /api/components/{id}", s.updateComponent)
	s.mux.HandleFunc("PATCH /api/components/{id}/status", s.patchComponentStatus)
	s.mux.HandleFunc("DELETE /api/components/{id}", s.deleteComponent)

	// Incidents
	s.mux.HandleFunc("GET /api/incidents", s.listIncidents)
	s.mux.HandleFunc("POST /api/incidents", s.createIncident)
	s.mux.HandleFunc("GET /api/incidents/{id}", s.getIncident)
	s.mux.HandleFunc("PUT /api/incidents/{id}", s.updateIncident)
	s.mux.HandleFunc("DELETE /api/incidents/{id}", s.deleteIncident)

	// Incident updates
	s.mux.HandleFunc("POST /api/incidents/{id}/updates", s.createIncidentUpdate)

	// Subscribers
	s.mux.HandleFunc("GET /api/subscribers", s.listSubscribers)
	s.mux.HandleFunc("POST /api/subscribers", s.subscribe)
	s.mux.HandleFunc("DELETE /api/subscribers/{email}", s.unsubscribe)

	// Public status page (read-only, no auth needed)
	s.mux.HandleFunc("GET /api/status", s.publicStatus)

	// General
	s.mux.HandleFunc("GET /api/stats", s.stats)
	s.mux.HandleFunc("GET /api/health", s.health)
	s.mux.HandleFunc("GET /api/tier", func(w http.ResponseWriter, r *http.Request) {
		wj(w, 200, map[string]any{"tier": s.limits.Tier, "upgrade_url": "https://stockyard.dev/paddock/"})
	})

	// UI
	s.mux.HandleFunc("GET /ui", s.dashboard)
	s.mux.HandleFunc("GET /ui/", s.dashboard)
	s.mux.HandleFunc("GET /", s.root)

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }
func wj(w http.ResponseWriter, c int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(c)
	json.NewEncoder(w).Encode(v)
}
func we(w http.ResponseWriter, c int, m string) { wj(w, c, map[string]string{"error": m}) }
func (s *Server) root(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/ui", 302)
}

// ── Components ──

func (s *Server) listComponents(w http.ResponseWriter, r *http.Request) {
	wj(w, 200, map[string]any{"components": s.db.ListComponents()})
}

func (s *Server) createComponent(w http.ResponseWriter, r *http.Request) {
	if s.limits.MaxItems > 0 && len(s.db.ListComponents()) >= s.limits.MaxItems {
		we(w, 402, "Free tier limit reached. Upgrade at https://stockyard.dev/paddock/")
		return
	}
	var c store.Component
	json.NewDecoder(r.Body).Decode(&c)
	if c.Name == "" {
		we(w, 400, "name required")
		return
	}
	s.db.CreateComponent(&c)
	wj(w, 201, s.db.GetComponent(c.ID))
}

func (s *Server) getComponent(w http.ResponseWriter, r *http.Request) {
	c := s.db.GetComponent(r.PathValue("id"))
	if c == nil {
		we(w, 404, "not found")
		return
	}
	wj(w, 200, c)
}

func (s *Server) updateComponent(w http.ResponseWriter, r *http.Request) {
	existing := s.db.GetComponent(r.PathValue("id"))
	if existing == nil {
		we(w, 404, "not found")
		return
	}
	var patch store.Component
	json.NewDecoder(r.Body).Decode(&patch)
	patch.ID = existing.ID
	patch.CreatedAt = existing.CreatedAt
	if patch.Name == "" {
		patch.Name = existing.Name
	}
	if patch.Status == "" {
		patch.Status = existing.Status
	}
	s.db.UpdateComponent(&patch)
	wj(w, 200, s.db.GetComponent(patch.ID))
}

func (s *Server) patchComponentStatus(w http.ResponseWriter, r *http.Request) {
	c := s.db.GetComponent(r.PathValue("id"))
	if c == nil {
		we(w, 404, "not found")
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	valid := map[string]bool{"operational": true, "degraded": true, "partial_outage": true, "major_outage": true, "maintenance": true}
	if !valid[body.Status] {
		we(w, 400, "status must be: operational, degraded, partial_outage, major_outage, maintenance")
		return
	}
	c.Status = body.Status
	s.db.UpdateComponent(c)
	wj(w, 200, s.db.GetComponent(c.ID))
}

func (s *Server) deleteComponent(w http.ResponseWriter, r *http.Request) {
	if s.db.GetComponent(r.PathValue("id")) == nil {
		we(w, 404, "not found")
		return
	}
	s.db.DeleteComponent(r.PathValue("id"))
	wj(w, 200, map[string]string{"status": "deleted"})
}

// ── Incidents ──

func (s *Server) listIncidents(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("active") == "true"
	wj(w, 200, map[string]any{"incidents": s.db.ListIncidents(activeOnly)})
}

func (s *Server) createIncident(w http.ResponseWriter, r *http.Request) {
	var inc store.Incident
	json.NewDecoder(r.Body).Decode(&inc)
	if inc.Title == "" {
		we(w, 400, "title required")
		return
	}
	s.db.CreateIncident(&inc)
	wj(w, 201, s.db.GetIncident(inc.ID))
}

func (s *Server) getIncident(w http.ResponseWriter, r *http.Request) {
	inc := s.db.GetIncident(r.PathValue("id"))
	if inc == nil {
		we(w, 404, "not found")
		return
	}
	wj(w, 200, inc)
}

func (s *Server) updateIncident(w http.ResponseWriter, r *http.Request) {
	existing := s.db.GetIncident(r.PathValue("id"))
	if existing == nil {
		we(w, 404, "not found")
		return
	}
	var patch store.Incident
	json.NewDecoder(r.Body).Decode(&patch)
	patch.ID = existing.ID
	patch.CreatedAt = existing.CreatedAt
	if patch.Title == "" {
		patch.Title = existing.Title
	}
	if patch.Status == "" {
		patch.Status = existing.Status
	}
	if patch.Impact == "" {
		patch.Impact = existing.Impact
	}
	s.db.UpdateIncident(&patch)
	wj(w, 200, s.db.GetIncident(patch.ID))
}

func (s *Server) deleteIncident(w http.ResponseWriter, r *http.Request) {
	if s.db.GetIncident(r.PathValue("id")) == nil {
		we(w, 404, "not found")
		return
	}
	s.db.DeleteIncident(r.PathValue("id"))
	wj(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) createIncidentUpdate(w http.ResponseWriter, r *http.Request) {
	inc := s.db.GetIncident(r.PathValue("id"))
	if inc == nil {
		we(w, 404, "incident not found")
		return
	}
	var u store.IncidentUpdate
	json.NewDecoder(r.Body).Decode(&u)
	u.IncidentID = inc.ID
	if u.Status == "" {
		u.Status = inc.Status
	}
	if u.Body == "" {
		we(w, 400, "body required")
		return
	}
	s.db.CreateIncidentUpdate(&u)
	wj(w, 201, s.db.GetIncident(inc.ID))
}

// ── Subscribers ──

func (s *Server) listSubscribers(w http.ResponseWriter, r *http.Request) {
	wj(w, 200, map[string]any{"subscribers": s.db.ListSubscribers()})
}

func (s *Server) subscribe(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Email == "" {
		we(w, 400, "email required")
		return
	}
	s.db.Subscribe(body.Email)
	wj(w, 201, map[string]string{"status": "subscribed", "email": body.Email})
}

func (s *Server) unsubscribe(w http.ResponseWriter, r *http.Request) {
	s.db.Unsubscribe(r.PathValue("email"))
	wj(w, 200, map[string]string{"status": "unsubscribed"})
}

// ── Public Status ──

func (s *Server) publicStatus(w http.ResponseWriter, r *http.Request) {
	stats := s.db.Stats()
	wj(w, 200, map[string]any{
		"status":     stats["overall_status"],
		"components": s.db.ListComponents(),
		"incidents":  s.db.ListIncidents(true),
	})
}

// ── General ──

func (s *Server) stats(w http.ResponseWriter, r *http.Request) { wj(w, 200, s.db.Stats()) }
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	stats := s.db.Stats()
	wj(w, 200, map[string]any{"service": "paddock", "status": "ok", "components": stats["components"], "overall": stats["overall_status"]})
}
