package pulse

import (
	"context"
	"embed"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql default-survey.json
var bundled embed.FS

type Config struct {
	Origin    string
	Secure    bool
	Retention int
}

type Actor struct {
	User
	CanEdit    bool `json:"can_edit"`
	CanResults bool `json:"can_results"`
	CanPlan    bool `json:"can_plan"`
	CanReview  bool `json:"can_review"`
	CanPublish bool `json:"can_publish"`
}

type Server struct {
	db      *pgxpool.Pool
	cfg     Config
	limit   *Limiter
	resolve func(*http.Request) (Actor, error)
}

type actorKey struct{}

func actor(r *http.Request) Actor {
	return r.Context().Value(actorKey{}).(Actor)
}

func New(ctx context.Context, db *pgxpool.Pool, origin string, admin Actor, resolve func(*http.Request) (Actor, error)) (http.Handler, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(812045)"); err != nil {
		return nil, err
	}
	schema, err := bundled.ReadFile("schema.sql")
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, string(schema)); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, saveUser, admin.ID, admin.Email, admin.Name, admin.Role); err != nil {
		return nil, err
	}
	var exists bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM surveys)").Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		raw, err := bundled.ReadFile("default-survey.json")
		if err != nil {
			return nil, err
		}
		var definition Definition
		if err := json.Unmarshal(raw, &definition); err != nil {
			return nil, err
		}
		for _, title := range []LocalText{{"نبض درعه · منتصف العام", "Deraah Pulse · Midyear"}, {"نبض درعه · نهاية العام", "Deraah Pulse · Year-end"}} {
			description := LocalText{"استبيان تجربة الموظف في درعه", "Deraah employee experience survey"}
			if _, err = tx.Exec(ctx, "INSERT INTO surveys(id,title,description,definition,created_by) VALUES($1,$2,$3,$4,$5)", newID(), marshal(title), marshal(description), marshal(definition), admin.ID); err != nil {
				return nil, err
			}
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	s := &Server{db: db, cfg: Config{origin, strings.HasPrefix(origin, "https://"), 365}, limit: newLimiter(), resolve: resolve}
	go s.housekeeping(context.Background())
	return s.routes(), nil
}

const saveUser = "INSERT INTO users(id,email,name,password_hash,role) VALUES($1,$2,$3,'portal-account',$4) ON CONFLICT(id) DO UPDATE SET email=excluded.email,name=excluded.name,role=excluded.role"

func (s *Server) private(fn http.HandlerFunc, allowed func(Actor) bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a, err := s.resolve(r)
		if err != nil {
			fail(w, 401, "unauthorized")
			return
		}
		if !allowed(a) {
			fail(w, 403, "forbidden")
			return
		}
		if r.Method != http.MethodGet {
			if _, err := s.db.Exec(r.Context(), saveUser, a.ID, a.Email, a.Name, a.Role); err != nil {
				fail(w, 500, "database_error")
				return
			}
		}
		fn(w, r.WithContext(context.WithValue(r.Context(), actorKey{}, a)))
	}
}

func (s *Server) routes() http.Handler {
	m := http.NewServeMux()
	edit := func(a Actor) bool { return a.CanEdit }
	results := func(a Actor) bool { return a.CanResults }
	plan := func(a Actor) bool { return a.CanPlan }
	review := func(a Actor) bool { return a.CanReview }
	manage := func(a Actor) bool { return a.CanEdit || a.CanResults || a.CanPlan || a.CanReview || a.CanPublish }
	m.HandleFunc("GET /api/overview", s.private(s.employeeOverview, func(a Actor) bool { return true }))
	m.HandleFunc("PATCH /api/surveys/{id}/visibility", s.private(s.changeVisibility, func(a Actor) bool { return a.CanPublish }))
	m.HandleFunc("GET /api/auth/me", s.private(func(w http.ResponseWriter, r *http.Request) {
		a := actor(r)
		if _, err := s.db.Exec(r.Context(), saveUser, a.ID, a.Email, a.Name, a.Role); err != nil {
			fail(w, 500, "database_error")
			return
		}
		writeJSON(w, 200, a)
	}, func(a Actor) bool { return true }))
	m.HandleFunc("GET /api/surveys", s.private(s.listSurveys, manage))
	m.HandleFunc("POST /api/surveys", s.private(s.createSurvey, edit))
	m.HandleFunc("PUT /api/surveys/{id}", s.private(s.updateSurvey, edit))
	m.HandleFunc("POST /api/surveys/{id}/status", s.private(s.surveyStatus, edit))
	m.HandleFunc("POST /api/surveys/{id}/link", s.private(s.sharedSurveyLink, edit))
	m.HandleFunc("POST /api/surveys/demo", s.private(s.demoSurvey, edit))
	m.HandleFunc("GET /api/surveys/{id}/analytics", s.private(s.analytics, results))
	m.HandleFunc("GET /api/surveys/{id}/export", s.private(s.exportAnalytics, results))
	m.HandleFunc("GET /api/surveys/{id}/comments", s.private(s.listComments, review))
	m.HandleFunc("PATCH /api/comments/{id}", s.private(s.moderateComment, review))
	m.HandleFunc("GET /api/plans", s.private(s.listPlans, plan))
	m.HandleFunc("POST /api/plans", s.private(s.createPlan, plan))
	m.HandleFunc("PUT /api/plans/{id}", s.private(s.updatePlan, plan))
	m.HandleFunc("GET /api/owners", s.private(s.listOwners, plan))
	m.HandleFunc("GET /api/public/surveys/{token}", s.publicSurvey)
	m.HandleFunc("POST /api/public/surveys/{token}/responses", s.submitResponse)
	return m
}

func (s *Server) setCookie(w http.ResponseWriter, name, value string, age int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/pulse/api/public/", HttpOnly: true, Secure: s.cfg.Secure, SameSite: http.SameSiteLaxMode, MaxAge: age})
}
