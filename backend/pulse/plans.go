package pulse

import (
	"net/http"
	"strings"
	"time"
)

type Plan struct {
	ID       string `json:"id"`
	SurveyID string `json:"survey_id"`
	Domain   string `json:"domain"`
	Title    string `json:"title"`
	OwnerID  string `json:"owner_id"`
	Owner    string `json:"owner"`
	DueDate  string `json:"due_date"`
	Status   string `json:"status"`
	Notes    string `json:"notes"`
}

func (s *Server) listPlans(w http.ResponseWriter, r *http.Request) {
	rows, e := s.db.Query(r.Context(), "SELECT p.id,p.survey_id,p.domain,p.title,p.owner_id,u.name,p.due_date::text,p.status,p.notes FROM action_plans p JOIN users u ON u.id=p.owner_id ORDER BY p.due_date LIMIT 500")
	if e != nil {
		fail(w, 500, "database_error")
		return
	}
	defer rows.Close()
	out := []Plan{}
	for rows.Next() {
		var p Plan
		rows.Scan(&p.ID, &p.SurveyID, &p.Domain, &p.Title, &p.OwnerID, &p.Owner, &p.DueDate, &p.Status, &p.Notes)
		out = append(out, p)
	}
	writeJSON(w, 200, out)
}
func (s *Server) savePlan(w http.ResponseWriter, r *http.Request, update bool) {
	var p Plan
	if !decode(w, r, &p) {
		return
	}
	_, e := time.Parse("2006-01-02", p.DueDate)
	if strings.TrimSpace(p.Title) == "" || len(p.Title) > 500 || len(p.Notes) > 8000 || e != nil || (p.Status != "planned" && p.Status != "in_progress" && p.Status != "done") {
		fail(w, 400, "invalid_plan")
		return
	}
	v, e := s.survey(r.Context(), p.SurveyID)
	if e != nil {
		fail(w, 400, "invalid_survey")
		return
	}
	domainOK := p.Domain == "general"
	for _, c := range v.Definition.Categories {
		domainOK = domainOK || c.ID == p.Domain
	}
	if !domainOK {
		fail(w, 400, "invalid_domain")
		return
	}
	var active bool
	e = s.db.QueryRow(r.Context(), "SELECT active AND role IN ('admin','analyst') FROM users WHERE id=$1", p.OwnerID).Scan(&active)
	if e != nil || !active {
		fail(w, 400, "invalid_owner")
		return
	}
	if update {
		p.ID = r.PathValue("id")
		tag, e := s.db.Exec(r.Context(), "UPDATE action_plans SET survey_id=$1,domain=$2,title=$3,owner_id=$4,due_date=$5::date,status=$6,notes=$7 WHERE id=$8", p.SurveyID, p.Domain, p.Title, p.OwnerID, p.DueDate, p.Status, p.Notes, p.ID)
		if e != nil {
			fail(w, 500, "database_error")
			return
		}
		if tag.RowsAffected() == 0 {
			fail(w, 404, "not_found")
			return
		}
	} else {
		p.ID = newID()
		if _, e = s.db.Exec(r.Context(), "INSERT INTO action_plans(id,survey_id,domain,title,owner_id,due_date,status,notes) VALUES($1,$2,$3,$4,$5,$6::date,$7,$8)", p.ID, p.SurveyID, p.Domain, p.Title, p.OwnerID, p.DueDate, p.Status, p.Notes); e != nil {
			fail(w, 500, "database_error")
			return
		}
	}
	s.audit(r.Context(), actor(r).ID, "plan_saved", "plan", p.ID)
	writeJSON(w, 200, map[string]string{"id": p.ID})
}
func (s *Server) createPlan(w http.ResponseWriter, r *http.Request) { s.savePlan(w, r, false) }
func (s *Server) updatePlan(w http.ResponseWriter, r *http.Request) { s.savePlan(w, r, true) }
