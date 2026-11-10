package pulse

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

func (s *Server) changeVisibility(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Visible bool `json:"visible"`
	}
	if !decode(w, r, &in) {
		return
	}
	id := r.PathValue("id")
	if id == demoSurveyID && in.Visible {
		fail(w, 409, "results_suppressed")
		return
	}
	if in.Visible {
		report, err := s.computeAnalytics(r, id)
		if err != nil || report.Overall.Suppressed {
			fail(w, 409, "results_suppressed")
			return
		}
	}
	tag, err := s.db.Exec(r.Context(), "UPDATE surveys SET employee_visible=$1 WHERE id=$2 AND status='closed'", in.Visible, id)
	if err != nil {
		fail(w, 500, "database_error")
		return
	}
	if tag.RowsAffected() != 1 {
		fail(w, 409, "results_suppressed")
		return
	}
	s.audit(r.Context(), actor(r).ID, "survey_visibility_changed", "survey", id)
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) employeeOverview(w http.ResponseWriter, r *http.Request) {
	v, err := scanSurvey(s.db.QueryRow(r.Context(), "SELECT "+surveyColumns+" FROM surveys s WHERE s.status='closed' AND s.employee_visible ORDER BY s.created_at DESC LIMIT 1"))
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, 200, nil)
		return
	}
	if err != nil {
		fail(w, 500, "database_error")
		return
	}
	report, err := s.computeAnalytics(r, v.ID)
	if err != nil {
		fail(w, 500, "database_error")
		return
	}
	if report.Overall.Suppressed {
		writeJSON(w, 200, nil)
		return
	}
	writeJSON(w, 200, map[string]any{"title": v.Title, "overall": report.Overall})
}
