package pulse

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type apiError struct {
	Error string `json:"error"`
}

func fail(w http.ResponseWriter, status int, code string) { writeJSON(w, status, apiError{code}) }
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		fail(w, 415, "json_required")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 256*1024)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		fail(w, 400, "invalid_json")
		return false
	}
	if e := d.Decode(new(any)); e != io.EOF {
		fail(w, 400, "invalid_json")
		return false
	}
	return true
}
func (s *Server) audit(ctx context.Context, uid, action, typ, id string) {
	_, e := s.db.Exec(ctx, "INSERT INTO audit_log(actor_id,action,entity_type,entity_id) VALUES(NULLIF($1,''),$2,$3,$4)", uid, action, typ, id)
	if e != nil {
		slog.Error("audit write failed")
	}
}

type limitEntry struct {
	count int
	until time.Time
}
type Limiter struct {
	mu   sync.Mutex
	data map[string]limitEntry
}

func newLimiter() *Limiter { return &Limiter{data: map[string]limitEntry{}} }
func (l *Limiter) allow(key string, max int, d time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	v := l.data[key]
	if now.After(v.until) {
		v = limitEntry{until: now.Add(d)}
	}
	if v.count >= max {
		return false
	}
	if len(l.data) >= 10000 {
		for k, v := range l.data {
			if now.After(v.until) {
				delete(l.data, k)
			}
		}
		if len(l.data) >= 10000 {
			return false
		}
	}
	v.count++
	l.data[key] = v
	return true
}
func (l *Limiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()
	for k, v := range l.data {
		if time.Now().After(v.until) {
			delete(l.data, k)
		}
	}
}
func ipKey(r *http.Request) string {
	h, _, e := net.SplitHostPort(r.RemoteAddr)
	if e != nil {
		h = r.RemoteAddr
	}
	return digest(h)
}
func (s *Server) rate(w http.ResponseWriter, r *http.Request, key string, max int, d time.Duration) bool {
	if !s.limit.allow(key, max, d) {
		w.Header().Set("Retry-After", strconv.Itoa(int(d.Seconds())))
		fail(w, 429, "rate_limited")
		return false
	}
	return true
}

func (s *Server) listOwners(w http.ResponseWriter, r *http.Request) {
	rows, e := s.db.Query(r.Context(), "SELECT id,name FROM users WHERE active AND role IN ('admin','analyst') ORDER BY name LIMIT 500")
	if e != nil {
		fail(w, 500, "database_error")
		return
	}
	defer rows.Close()
	out := []map[string]string{}
	for rows.Next() {
		var id, name string
		rows.Scan(&id, &name)
		out = append(out, map[string]string{"id": id, "name": name})
	}
	writeJSON(w, 200, out)
}
func (s *Server) housekeeping(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	runs := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.db.Exec(ctx, "UPDATE surveys SET status='closed' WHERE status='published' AND closes_at IS NOT NULL AND closes_at<=now()")
			rows, err := s.db.Query(ctx, "SELECT id FROM surveys WHERE status='closed' AND NOT EXISTS(SELECT 1 FROM survey_reports WHERE survey_id=surveys.id)")
			if err == nil {
				ids := []string{}
				for rows.Next() {
					var id string
					rows.Scan(&id)
					ids = append(ids, id)
				}
				rows.Close()
				for _, id := range ids {
					s.computeAnalytics((&http.Request{}).WithContext(ctx), id)
				}
			}
			if runs%60 == 0 {
				s.db.Exec(ctx, "DELETE FROM responses WHERE submitted_on<current_date-$1::integer AND EXISTS(SELECT 1 FROM survey_reports WHERE survey_id=responses.survey_id)", s.cfg.Retention)
				s.db.Exec(ctx, "DELETE FROM response_claims WHERE EXISTS(SELECT 1 FROM survey_reports sr WHERE sr.survey_id=response_claims.survey_id AND sr.created_at<now()-make_interval(days=>$1))", s.cfg.Retention)
				s.limit.cleanup()
			}
			runs++
		}
	}
}
