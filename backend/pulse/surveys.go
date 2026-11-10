package pulse

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strings"
	"time"
)

type scanner interface{ Scan(...any) error }

const surveyColumns = "s.id,s.title,s.description,s.definition,s.status,s.expected_count,s.excellent,s.good,to_char(s.opens_at AT TIME ZONE 'UTC','YYYY-MM-DD\"T\"HH24:MI:SS\"Z\"'),to_char(s.closes_at AT TIME ZONE 'UTC','YYYY-MM-DD\"T\"HH24:MI:SS\"Z\"'),s.created_at::text,s.employee_visible,COALESCE((SELECT (report->>'response_count')::integer FROM survey_reports WHERE survey_id=s.id),(SELECT count(*)::integer FROM responses r WHERE r.survey_id=s.id))"

func scanSurvey(row scanner) (Survey, error) {
	var s Survey
	var title, desc, def []byte
	e := row.Scan(&s.ID, &title, &desc, &def, &s.Status, &s.Expected, &s.Excellent, &s.Good, &s.OpensAt, &s.ClosesAt, &s.CreatedAt, &s.EmployeeVisible, &s.Responses)
	if e != nil {
		return s, e
	}
	json.Unmarshal(title, &s.Title)
	json.Unmarshal(desc, &s.Description)
	json.Unmarshal(def, &s.Definition)
	return s, nil
}
func (s *Server) survey(ctx context.Context, id string) (Survey, error) {
	return scanSurvey(s.db.QueryRow(ctx, "SELECT "+surveyColumns+" FROM surveys s WHERE s.id=$1", id))
}
func (s *Server) listSurveys(w http.ResponseWriter, r *http.Request) {
	rows, e := s.db.Query(r.Context(), "SELECT "+surveyColumns+" FROM surveys s ORDER BY s.created_at DESC LIMIT 200")
	if e != nil {
		fail(w, 500, "database_error")
		return
	}
	defer rows.Close()
	out := []Survey{}
	for rows.Next() {
		v, e := scanSurvey(rows)
		if e != nil {
			fail(w, 500, "database_error")
			return
		}
		out = append(out, v)
	}
	writeJSON(w, 200, out)
}
func validSurvey(v Survey) bool {
	if !v.Title.valid(180) || !v.Description.valid(2000) || v.Definition.validate() != nil || v.Expected < 0 || v.Expected > 10000000 || v.Good < -100 || v.Excellent > 100 || v.Excellent <= v.Good {
		return false
	}
	var start, end time.Time
	var e error
	if v.OpensAt != nil {
		start, e = time.Parse(time.RFC3339, *v.OpensAt)
		if e != nil {
			return false
		}
	}
	if v.ClosesAt != nil {
		end, e = time.Parse(time.RFC3339, *v.ClosesAt)
		if e != nil {
			return false
		}
	}
	return start.IsZero() || end.IsZero() || end.After(start)
}
func (s *Server) createSurvey(w http.ResponseWriter, r *http.Request) {
	var in Survey
	if !decode(w, r, &in) {
		return
	}
	if !validSurvey(in) {
		fail(w, 400, "invalid_survey")
		return
	}
	in.ID = newID()
	_, e := s.db.Exec(r.Context(), "INSERT INTO surveys(id,title,description,definition,expected_count,excellent,good,opens_at,closes_at,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8::timestamptz,$9::timestamptz,$10)", in.ID, marshal(in.Title), marshal(in.Description), marshal(in.Definition), in.Expected, in.Excellent, in.Good, in.OpensAt, in.ClosesAt, actor(r).ID)
	if e != nil {
		fail(w, 500, "database_error")
		return
	}
	s.audit(r.Context(), actor(r).ID, "survey_created", "survey", in.ID)
	writeJSON(w, 201, map[string]string{"id": in.ID})
}
func (s *Server) updateSurvey(w http.ResponseWriter, r *http.Request) {
	var in Survey
	if !decode(w, r, &in) {
		return
	}
	if !validSurvey(in) {
		fail(w, 400, "invalid_survey")
		return
	}
	id := r.PathValue("id")
	tag, e := s.db.Exec(r.Context(), "UPDATE surveys SET title=$1,description=$2,definition=$3,expected_count=$4,excellent=$5,good=$6,opens_at=$7::timestamptz,closes_at=$8::timestamptz WHERE id=$9 AND status='draft'", marshal(in.Title), marshal(in.Description), marshal(in.Definition), in.Expected, in.Excellent, in.Good, in.OpensAt, in.ClosesAt, id)
	if e != nil {
		fail(w, 500, "database_error")
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, 409, "survey_locked")
		return
	}
	s.audit(r.Context(), actor(r).ID, "survey_updated", "survey", id)
	writeJSON(w, 200, map[string]bool{"ok": true})
}
func (s *Server) surveyStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Status string `json:"status"`
	}
	if !decode(w, r, &in) {
		return
	}
	from := "draft"
	if in.Status == "closed" {
		from = "published"
	} else if in.Status != "published" {
		fail(w, 400, "invalid_status")
		return
	}
	id := r.PathValue("id")
	tag, e := s.db.Exec(r.Context(), "UPDATE surveys SET status=$1 WHERE id=$2 AND status=$3", in.Status, id, from)
	if e != nil {
		fail(w, 500, "database_error")
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, 409, "invalid_transition")
		return
	}
	s.audit(r.Context(), actor(r).ID, "survey_"+in.Status, "survey", id)
	writeJSON(w, 200, map[string]bool{"ok": true})
}

type Segment struct {
	ID    string    `json:"id"`
	Label LocalText `json:"label"`
	Site  string    `json:"site"`
	Track string    `json:"track"`
}

type inviteSurvey struct {
	Survey
	Segment        Segment
	InviteID, Mode string
	Used, Active   bool
	Open           bool
}

func (s *Server) resolveInvite(ctx context.Context, token string) (inviteSurvey, error) {
	var out inviteSurvey
	if len(token) != 43 {
		return out, pgx.ErrNoRows
	}
	var id string
	e := s.db.QueryRow(ctx, "SELECT i.id,i.survey_id,i.segment_id,i.mode,i.used,i.active,g.label,g.site,g.track FROM invites i JOIN segments g ON g.id=i.segment_id WHERE i.token_hash=$1", digest(token)).Scan(&out.InviteID, &id, &out.Segment.ID, &out.Mode, &out.Used, &out.Active, &out.Segment.Label, &out.Segment.Site, &out.Segment.Track)
	if e != nil {
		return out, e
	}
	out.Survey, e = s.survey(ctx, id)
	if e != nil {
		return out, e
	}
	out.Open = out.Status == "published" && out.Active && !(out.Mode == "single" && out.Used)
	now := time.Now()
	if out.OpensAt != nil {
		t, _ := time.Parse(time.RFC3339, *out.OpensAt)
		out.Open = out.Open && !now.Before(t)
	}
	if out.ClosesAt != nil {
		t, _ := time.Parse(time.RFC3339, *out.ClosesAt)
		out.Open = out.Open && now.Before(t)
	}
	return out, nil
}
func (s *Server) publicSurvey(w http.ResponseWriter, r *http.Request) {
	if !s.rate(w, r, "survey-get:"+ipKey(r), 300, time.Minute) {
		return
	}
	v, e := s.resolveInvite(r.Context(), r.PathValue("token"))
	if e != nil {
		fail(w, 404, "invite_not_found")
		return
	}
	if !v.Open {
		fail(w, 410, "survey_unavailable")
		return
	}
	nonce := ""
	if c, e := r.Cookie("voice_browser"); e == nil && len(c.Value) == 43 {
		nonce = c.Value
	}
	if nonce == "" {
		nonce = randomToken(32)
		s.setCookie(w, "voice_browser", nonce, 365*86400)
	}
	cats := []Category{}
	for _, c := range v.Definition.Categories {
		if v.Segment.Track == "all" || c.Track == "all" || c.Track == v.Segment.Track {
			cats = append(cats, c)
		}
	}
	writeJSON(w, 200, map[string]any{"id": v.ID, "title": v.Title, "description": v.Description, "definition": Definition{cats}, "retention_days": s.cfg.Retention})
}
func (s *Server) submitResponse(w http.ResponseWriter, r *http.Request) {
	if !s.rate(w, r, "survey-submit:"+ipKey(r), 120, time.Hour) {
		return
	}
	var in struct {
		Enps    *int            `json:"enps"`
		Answers map[string]*int `json:"answers"`
		Reason  string          `json:"reason"`
		Comment string          `json:"comment"`
		Tenure  string          `json:"tenure"`
		Consent bool            `json:"consent"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !in.Consent || in.Enps == nil || *in.Enps < 0 || *in.Enps > 10 || len(in.Reason) > 8000 || len(in.Comment) > 8000 || !validTenure(in.Tenure) {
		fail(w, 400, "invalid_response")
		return
	}
	cookie, e := r.Cookie("voice_browser")
	if e != nil || len(cookie.Value) != 43 {
		fail(w, 400, "browser_cookie_required")
		return
	}
	v, e := s.resolveInvite(r.Context(), r.PathValue("token"))
	if e != nil {
		fail(w, 404, "invite_not_found")
		return
	}
	if !v.Open {
		fail(w, 410, "survey_unavailable")
		return
	}
	if e = validateAnswers(v.Definition, v.Segment.Track, in.Answers); e != nil {
		fail(w, 400, e.Error())
		return
	}
	tx, e := s.db.Begin(r.Context())
	if e != nil {
		fail(w, 500, "database_error")
		return
	}
	defer tx.Rollback(r.Context())
	var status string
	var open bool
	e = tx.QueryRow(r.Context(), "SELECT status,(opens_at IS NULL OR opens_at<=now()) AND (closes_at IS NULL OR closes_at>now()) FROM surveys WHERE id=$1 FOR SHARE", v.ID).Scan(&status, &open)
	if e != nil || status != "published" || !open {
		fail(w, 410, "survey_unavailable")
		return
	}
	var active, used bool
	tx.QueryRow(r.Context(), "SELECT active,used FROM invites WHERE id=$1 FOR UPDATE", v.InviteID).Scan(&active, &used)
	if !active || (v.Mode == "single" && used) {
		fail(w, 409, "already_submitted")
		return
	}
	tag, e := tx.Exec(r.Context(), "INSERT INTO response_claims(survey_id,claim_hash) VALUES($1,$2) ON CONFLICT DO NOTHING", v.ID, digest(v.ID+":"+cookie.Value))
	if e != nil {
		fail(w, 500, "database_error")
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, 409, "already_submitted")
		return
	}
	if e = storeResponse(r.Context(), tx, v.ID, v.Segment.ID, *in.Enps, in.Answers, in.Reason, in.Comment); e != nil {
		fail(w, 500, "database_error")
		return
	}
	if v.Mode == "single" {
		if _, e = tx.Exec(r.Context(), "UPDATE invites SET used=true WHERE id=$1", v.InviteID); e != nil {
			fail(w, 500, "database_error")
			return
		}
	}
	if e = tx.Commit(r.Context()); e != nil {
		fail(w, 500, "database_error")
		return
	}
	writeJSON(w, 201, map[string]bool{"ok": true})
}
func validTenure(s string) bool {
	return s == "" || s == "lt1" || s == "1to3" || s == "3to5" || s == "gt5"
}

func storeResponse(ctx context.Context, tx pgx.Tx, surveyID, segmentID string, enps int, answers map[string]*int, comments ...string) error {
	id := newID()
	if _, err := tx.Exec(ctx, "INSERT INTO responses(id,survey_id,segment_id,enps,answers) VALUES($1,$2,$3,$4,$5)", id, surveyID, segmentID, enps, marshal(answers)); err != nil {
		return err
	}
	for _, comment := range comments {
		if text := strings.TrimSpace(comment); text != "" {
			if _, err := tx.Exec(ctx, "INSERT INTO comments(id,response_id,body) VALUES($1,$2,$3)", newID(), id, text); err != nil {
				return err
			}
		}
	}
	return nil
}
