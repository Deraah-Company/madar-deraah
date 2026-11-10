package main

import (
	"context"
	"crypto/pbkdf2"
	"crypto/sha256"
	"crypto/subtle"
	"deraah/trainer/pulse"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed web/* default-survey.json demo-bulletin.pdf
var content embed.FS

type server struct {
	db              *pgxpool.Pool
	origin, dataDir string
	static          http.Handler
	pulse           http.Handler
	mu              sync.Mutex
	attempts        map[string]bucket
}
type bucket struct {
	Until time.Time
	Count int
}
type fault struct {
	Status  int
	Message string
}

func (f fault) Error() string { return f.Message }
func bad(s string) error      { return fault{400, s} }
func forbidden() error        { return fault{403, "الصلاحية غير متاحة لهذا الحساب"} }
func missing() error          { return fault{404, "العنصر غير موجود"} }
func loginRequired() error    { return fault{401, "سجّل الدخول أولاً"} }
func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
func respond(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}
func respondError(w http.ResponseWriter, e error) {
	var f fault
	if errors.As(e, &f) {
		respond(w, f.Status, map[string]string{"error": f.Message})
		return
	}
	log.Printf("request failed: %v", e)
	respond(w, 500, map[string]string{"error": "تعذر إتمام العملية"})
}
func decode(r *http.Request, v any) error {
	d := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		return bad("راجع البيانات المدخلة")
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return bad("بيانات إضافية غير صالحة")
	}
	return nil
}
func payload(raw json.RawMessage, v any) error {
	if json.Unmarshal(raw, v) != nil {
		return bad("راجع البيانات المدخلة")
	}
	return nil
}

func (s *server) read(ctx context.Context) (Data, error) {
	var d Data
	var raw []byte
	if e := s.db.QueryRow(ctx, "SELECT document FROM pathways_data WHERE singleton = true").Scan(&raw); e != nil {
		return d, e
	}
	return d, json.Unmarshal(raw, &d)
}
func (s *server) write(ctx context.Context, fn func(*Data) error) error {
	tx, e := s.db.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var raw []byte
	var d Data
	if e = tx.QueryRow(ctx, "SELECT document FROM pathways_data WHERE singleton = true FOR UPDATE").Scan(&raw); e != nil {
		return e
	}
	if e = json.Unmarshal(raw, &d); e != nil {
		return e
	}
	if e = fn(&d); e != nil {
		return e
	}
	raw, e = json.Marshal(d)
	if e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE pathways_data SET document=$1 WHERE singleton=true", raw); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func digest(s string) string { b := sha256.Sum256([]byte(s)); return hex.EncodeToString(b[:]) }
func passwordKey(password, salt string) string {
	b, e := pbkdf2.Key(sha256.New, password, []byte(salt), 210000, 32)
	if e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
func secret(password string) string { salt := nonce(); return salt + ":" + passwordKey(password, salt) }
func passwordMatches(password, stored string) bool {
	salt, h, ok := strings.Cut(stored, ":")
	return ok && subtle.ConstantTimeCompare([]byte(passwordKey(password, salt)), []byte(h)) == 1
}
func token(r *http.Request) string {
	c, e := r.Cookie("deraah_pathways")
	if e != nil || len(c.Value) != 48 {
		return ""
	}
	return digest(c.Value)
}
func signedIn(d *Data, r *http.Request) (Account, error) {
	k, ok := d.Sessions[token(r)]
	a, exists := d.Accounts[k.Account]
	if !ok || !exists || a.Status != "active" || a.Epoch != k.Epoch || time.Now().After(k.Expires) {
		return Account{}, loginRequired()
	}
	return a, nil
}
func can(a Account, perm string) bool { return permissions(a)[perm] }
func audit(d *Data, a Account, action, target string) {
	d.Audit = append(d.Audit, Audit{timestamp(), a.Name, action, target})
}
func (s *server) limiter(r *http.Request, scope string, max int) bool {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	k := scope + "/" + ip
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if len(s.attempts) > 4096 {
		for key, v := range s.attempts {
			if now.After(v.Until) {
				delete(s.attempts, key)
			}
		}
	}
	b := s.attempts[k]
	if now.After(b.Until) {
		b = bucket{now.Add(time.Minute), 0}
	}
	if b.Count >= max {
		return false
	}
	b.Count++
	s.attempts[k] = b
	return true
}
func (s *server) validOrigin(r *http.Request) bool {
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" || origin == s.origin {
		return true
	}
	a, e := url.Parse(origin)
	b, f := url.Parse(s.origin)
	local := func(h string) bool { return h == "localhost" || h == "127.0.0.1" || h == "::1" }
	return e == nil && f == nil && a.Scheme == b.Scheme && a.Port() == b.Port() && local(a.Hostname()) && local(b.Hostname())
}
func (s *server) serve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; media-src 'self'; frame-ancestors 'none'; object-src 'none'; base-uri 'none'")
	if r.Method != "GET" && r.Method != "HEAD" && !s.validOrigin(r) {
		respondError(w, forbidden())
		return
	}
	if r.URL.Path == "/health" {
		if e := s.db.Ping(r.Context()); e != nil {
			respond(w, 503, map[string]string{"status": "unavailable"})
			return
		}
		respond(w, 200, map[string]string{"status": "ok"})
		return
	}
	if s.pulse != nil && (r.URL.Path == "/pulse" || strings.HasPrefix(r.URL.Path, "/pulse/api/") || strings.HasPrefix(r.URL.Path, "/pulse/r/")) {
		d, err := s.read(r.Context())
		if err != nil {
			respondError(w, err)
			return
		}
		if !programOpen(&d, "voice") {
			respondError(w, missing())
			return
		}
		if strings.HasPrefix(r.URL.Path, "/pulse/api/") {
			s.pulse.ServeHTTP(w, r)
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			http.Error(w, "Method not allowed", 405)
			return
		}
		page, err := content.ReadFile("web/pulse.html")
		if err != nil {
			respondError(w, err)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(page)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/files/") {
		if e := s.api(w, r); e != nil {
			respondError(w, e)
		}
		return
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	if r.URL.Path == "/" || r.URL.Path == "/login" || r.URL.Path == "/settings" || r.URL.Path == "/knowledge" || r.URL.Path == "/knowledge/apply" || r.URL.Path == "/knowledge/assessment" || r.URL.Path == "/pulse" || r.URL.Path == "/pulse/history" || r.URL.Path == "/family" || r.URL.Path == "/survey" {
		b, _ := content.ReadFile("web/index.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(b)
		return
	}
	s.static.ServeHTTP(w, r)
}
func (s *server) api(w http.ResponseWriter, r *http.Request) error {
	switch r.Method + " " + r.URL.Path {
	case "POST /api/login":
		return s.login(w, r)
	case "POST /api/register":
		return s.register(w, r)
	case "POST /api/public/knowledge":
		return s.publicTrainerSubmit(w, r)
	case "POST /api/survey/submit":
		return s.answerSurvey(w, r)
	case "POST /api/upload":
		return s.upload(w, r)
	case "POST /api/family/save":
		return s.saveFamily(w, r)
	case "POST /api/action":
		return s.action(w, r)
	case "POST /api/logout":
		if e := s.write(r.Context(), func(d *Data) error { delete(d.Sessions, token(r)); return nil }); e != nil {
			return e
		}
		s.cookie(w, "", -1)
		respond(w, 200, map[string]bool{"ok": true})
		return nil
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		return missing()
	}
	d, e := s.read(r.Context())
	if e != nil {
		return e
	}
	if r.URL.Path == "/api/public" {
		respond(w, 200, map[string]any{"registration": d.Registration, "departments": d.Departments, "programs": programAvailability(&d)})
		return nil
	}
	if r.URL.Path == "/api/public/knowledge" {
		if !programOpen(&d, "trainer") {
			return missing()
		}
		respond(w, 200, publicTrainerView(&d, r))
		return nil
	}
	if r.URL.Path == "/api/survey" {
		if !programOpen(&d, "voice") {
			return missing()
		}
		return publicSurvey(w, r, &d)
	}
	a, e := signedIn(&d, r)
	if e != nil {
		return e
	}
	if strings.HasPrefix(r.URL.Path, "/files/family/") {
		if !programOpen(&d, "family") {
			return missing()
		}
		return s.serveFamilyFile(w, r, &d, a)
	}
	if strings.HasPrefix(r.URL.Path, "/files/") {
		if !programOpen(&d, "trainer") {
			return missing()
		}
		return s.serveAsset(w, r, &d, a)
	}
	switch r.URL.Path {
	case "/api/trainer/assessments":
		if !programOpen(&d, "trainer") || !can(a, "trainer.assess") {
			return forbidden()
		}
		rows := []map[string]any{}
		for i := range d.Assessments {
			rows = append(rows, assessmentView(&d, &d.Assessments[i], a, true))
		}
		respond(w, 200, map[string]any{"items": rows, "template": feedbackTemplate})
	case "/api/trainer/assessment":
		if !programOpen(&d, "trainer") || !can(a, "trainer.apply") {
			return forbidden()
		}
		item := assessmentFor(&d, r.URL.Query().Get("id"))
		if item == nil {
			return missing()
		}
		respond(w, 200, assessmentView(&d, item, a, false))
	case "/api/notifications":
		respond(w, 200, notificationView(&d, a))
	case "/api/portal":
		respond(w, 200, portal(&d, a))
	case "/api/trainer":
		if !programOpen(&d, "trainer") {
			return missing()
		}
		if !can(a, "trainer.view") {
			return forbidden()
		}
		respond(w, 200, trainerView(&d, a))
	case "/api/voice":
		if !programOpen(&d, "voice") {
			return missing()
		}
		if !can(a, "voice.view") {
			return forbidden()
		}
		respond(w, 200, voiceView(&d, a))
	case "/api/family":
		if !programOpen(&d, "family") {
			return missing()
		}
		if !can(a, "family.view") {
			return forbidden()
		}
		respond(w, 200, familyView(&d, a))
	default:
		return missing()
	}
	return nil
}
func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	db, e := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if e != nil {
		log.Fatal(e)
	}
	defer db.Close()
	if _, e = db.Exec(ctx, "CREATE TABLE IF NOT EXISTS pathways_data (singleton boolean PRIMARY KEY CHECK (singleton), document jsonb NOT NULL)"); e != nil {
		log.Fatal(e)
	}
	var existing bool
	if e = db.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pathways_data)").Scan(&existing); e != nil {
		log.Fatal(e)
	}
	if !existing {
		password := os.Getenv("ADMIN_PASSWORD")
		if len(password) < 12 {
			log.Fatal("Set ADMIN_PASSWORD to at least 12 characters on first start")
		}
		d := initialData(env("ADMIN_EMAIL", "admin@deraah.local"), password)
		raw, _ := json.Marshal(d)
		if _, e = db.Exec(ctx, "INSERT INTO pathways_data VALUES(true,$1) ON CONFLICT DO NOTHING", raw); e != nil {
			log.Fatal(e)
		}
	}
	assets, _ := fs.Sub(content, "web")
	s := &server{db: db, origin: env("APP_ORIGIN", "http://localhost:8093"), dataDir: env("DATA_DIR", "data"), static: http.FileServer(http.FS(assets)), attempts: map[string]bucket{}}
	d, e := s.read(ctx)
	if e != nil {
		log.Fatal(e)
	}
	var systemAdmin Account
	for _, account := range d.Accounts {
		if account.System && account.Status == "active" {
			systemAdmin = account
			break
		}
	}
	if systemAdmin.ID == "" {
		log.Fatal("Pulse requires an active portal administrator")
	}
	handler, e := pulse.New(ctx, db, s.origin, pulseActor(systemAdmin), s.resolvePulseActor)
	if e != nil {
		log.Fatal(e)
	}
	s.pulse = http.StripPrefix("/pulse", handler)
	srv := &http.Server{Addr: env("HTTP_ADDR", ":8080"), Handler: http.HandlerFunc(s.serve), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 45 * time.Second, WriteTimeout: 45 * time.Second, IdleTimeout: 90 * time.Second}
	log.Printf("Pathways running at %s", s.origin)
	log.Fatal(srv.ListenAndServe())
}
