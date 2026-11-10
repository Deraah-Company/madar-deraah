package pulse

import (
	"encoding/csv"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type Metric struct {
	Count      *int     `json:"count"`
	ENPS       *float64 `json:"enps"`
	Promoters  *int     `json:"promoters"`
	Passives   *int     `json:"passives"`
	Detractors *int     `json:"detractors"`
	Suppressed bool     `json:"suppressed"`
}
type Response struct {
	ID, SegmentID, Site, Track string
	Label                      LocalText
	Enps                       int
	Answers                    map[string]*int
}
type Driver struct {
	ID          string    `json:"id"`
	Name        LocalText `json:"name"`
	Average     *float64  `json:"average"`
	Favorable   *float64  `json:"favorable"`
	Respondents *int      `json:"respondents"`
	Track       string    `json:"track"`
}
type Theme struct {
	Word  string `json:"word"`
	Count int    `json:"count"`
}
type Analytics struct {
	SurveyID      string   `json:"survey_id"`
	Status        string   `json:"status"`
	ResponseCount int      `json:"response_count"`
	Expected      int      `json:"expected_count"`
	Release       string   `json:"release"`
	Overall       Metric   `json:"overall"`
	Drivers       []Driver `json:"drivers"`
	Themes        []Theme  `json:"themes"`
}

func round(v float64) float64 { return math.Round(v*100) / 100 }
func aggregate(rows []Response) Metric {
	if len(rows) == 0 {
		return Metric{Suppressed: true}
	}
	n := len(rows)
	p, d, a := 0, 0, 0
	for _, r := range rows {
		if r.Enps >= 9 {
			p++
		} else if r.Enps <= 6 {
			d++
		} else {
			a++
		}
	}
	e := round(float64(p-d) * 100 / float64(n))
	return Metric{Count: &n, ENPS: &e, Promoters: &p, Passives: &a, Detractors: &d}
}
func (s *Server) computeCoreAnalytics(r *http.Request, id string) (Analytics, error) {
	v, e := s.survey(r.Context(), id)
	a := Analytics{SurveyID: id, Status: v.Status, ResponseCount: v.Responses, Expected: v.Expected, Release: "closed_only", Overall: Metric{Suppressed: true}, Drivers: []Driver{}, Themes: []Theme{}}
	if e != nil {
		return a, e
	}
	if v.Status != "closed" {
		return a, nil
	}
	a.Release = "released"
	rows, e := s.db.Query(r.Context(), "SELECT r.id,r.segment_id,r.enps,r.answers,g.label,g.site,g.track FROM responses r JOIN segments g ON g.id=r.segment_id WHERE r.survey_id=$1", id)
	if e != nil {
		return a, e
	}
	all := []Response{}
	for rows.Next() {
		var v Response
		var b, label []byte
		if e = rows.Scan(&v.ID, &v.SegmentID, &v.Enps, &b, &label, &v.Site, &v.Track); e != nil {
			rows.Close()
			return a, e
		}
		json.Unmarshal(b, &v.Answers)
		json.Unmarshal(label, &v.Label)
		all = append(all, v)
	}
	rows.Close()
	a.Overall = aggregate(all)
	if a.Overall.Suppressed {
		return a, nil
	}
	for _, c := range v.Definition.Categories {
		sum, positive, total := 0.0, 0, 0
		n := 0
		for _, r := range all {
			if r.Track != "all" && c.Track != "all" && c.Track != r.Track {
				continue
			}
			local, count := 0, 0
			for _, q := range c.Questions {
				if val := r.Answers[q.ID]; val != nil {
					local += *val
					count++
					total++
					if *val >= 4 {
						positive++
					}
				}
			}
			if count > 0 {
				n++
				sum += float64(local) / float64(count)
			}
		}
		d := Driver{ID: c.ID, Name: c.Name, Track: c.Track}
		if n > 0 {
			avg := round(sum / float64(n))
			fav := round(float64(positive) * 100 / float64(total))
			d.Average = &avg
			d.Favorable = &fav
			d.Respondents = &n
		}
		a.Drivers = append(a.Drivers, d)
	}
	return a, nil
}

var stopwords = map[string]bool{"the": true, "and": true, "for": true, "that": true, "this": true, "with": true, "have": true, "work": true, "company": true, "من": true, "على": true, "إلى": true, "الى": true, "هذا": true, "هذه": true, "التي": true, "الذي": true, "الشركة": true, "العمل": true, "هناك": true, "ولكن": true, "أكثر": true, "about": true, "would": true, "they": true, "their": true, "when": true, "because": true}

func (s *Server) analytics(w http.ResponseWriter, r *http.Request) {
	a, e := s.computeAnalytics(r, r.PathValue("id"))
	if e != nil {
		fail(w, 404, "not_found")
		return
	}
	writeJSON(w, 200, a)
}
func csvSafe(v string) string {
	if strings.ContainsAny(strings.TrimLeft(v, " \t\r\n")[:min(1, len(strings.TrimLeft(v, " \t\r\n")))], "=+-@") {
		return "'" + v
	}
	return v
}
func (s *Server) exportAnalytics(w http.ResponseWriter, r *http.Request) {
	a, e := s.computeAnalytics(r, r.PathValue("id"))
	if e != nil {
		fail(w, 404, "not_found")
		return
	}
	if a.Release != "released" || a.Overall.Suppressed {
		fail(w, 409, "results_suppressed")
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=employee-voice-aggregates.csv")
	w.Write([]byte{0xEF, 0xBB, 0xBF})
	c := csv.NewWriter(w)
	defer c.Flush()
	c.Write([]string{"type", "label_ar", "label_en", "respondents", "enps", "average_1_to_5", "favorable_percent"})
	c.Write([]string{"overall", "الإجمالي", "Overall", strconv.Itoa(*a.Overall.Count), strconv.FormatFloat(*a.Overall.ENPS, 'f', 2, 64), "", ""})
	for _, d := range a.Drivers {
		if d.Average != nil {
			c.Write([]string{"driver", csvSafe(d.Name.Ar), csvSafe(d.Name.En), strconv.Itoa(*d.Respondents), "", strconv.FormatFloat(*d.Average, 'f', 2, 64), strconv.FormatFloat(*d.Favorable, 'f', 2, 64)})
		}
	}
	s.audit(r.Context(), actor(r).ID, "aggregate_export", "survey", a.SurveyID)
}
func (s *Server) listComments(w http.ResponseWriter, r *http.Request) {
	v, e := s.survey(r.Context(), r.PathValue("id"))
	if e != nil {
		fail(w, 404, "not_found")
		return
	}
	if v.Status != "closed" || v.Responses == 0 {
		fail(w, 409, "results_suppressed")
		return
	}
	rows, e := s.db.Query(r.Context(), "SELECT c.id,c.body,c.status FROM comments c JOIN responses r ON r.id=c.response_id WHERE r.survey_id=$1 ORDER BY c.id", v.ID)
	if e != nil {
		fail(w, 500, "database_error")
		return
	}
	defer rows.Close()
	out := []surveyComment{}
	for rows.Next() {
		var row surveyComment
		if err := rows.Scan(&row.ID, &row.Body, &row.Status); err != nil {
			fail(w, 500, "database_error")
			return
		}
		out = append(out, row)
	}
	if rows.Err() != nil {
		fail(w, 500, "database_error")
		return
	}
	a := actor(r)
	if _, err := s.db.Exec(r.Context(), saveUser, a.ID, a.Email, a.Name, a.Role); err != nil {
		fail(w, 500, "database_error")
		return
	}
	if r.URL.Query().Get("format") == "xlsx" {
		s.audit(r.Context(), a.ID, "comments_exported", "survey", v.ID)
		writeCommentsXLSX(w, out)
		return
	}
	s.audit(r.Context(), a.ID, "comments_reviewed", "survey", v.ID)
	writeJSON(w, 200, out)
}
func (s *Server) moderateComment(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Status string `json:"status"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Status != "approved" && in.Status != "rejected" {
		fail(w, 400, "invalid_status")
		return
	}
	tag, e := s.db.Exec(r.Context(), "UPDATE comments c SET status=$1,reviewed_by=$2 FROM responses r,surveys s WHERE c.id=$3 AND c.response_id=r.id AND r.survey_id=s.id AND s.status='closed' ", in.Status, actor(r).ID, r.PathValue("id"))
	if e != nil {
		fail(w, 500, "database_error")
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, 409, "results_suppressed")
		return
	}
	s.audit(r.Context(), actor(r).ID, "comment_"+in.Status, "comment", r.PathValue("id"))
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) populateThemes(r *http.Request, a *Analytics) error {
	id := a.SurveyID
	// Only approved text contributes. A word must appear in >=k different responses.
	rows, e := s.db.Query(r.Context(), "SELECT c.response_id,c.body FROM comments c JOIN responses r ON r.id=c.response_id WHERE r.survey_id=$1 AND c.status='approved'", id)
	if e != nil {
		return e
	}
	defer rows.Close()
	freq := map[string]map[string]bool{}
	for rows.Next() {
		var rid, body string
		rows.Scan(&rid, &body)
		for _, w := range strings.FieldsFunc(strings.ToLower(body), func(r rune) bool { return !unicode.IsLetter(r) }) {
			if len([]rune(w)) < 3 || stopwords[w] {
				continue
			}
			if freq[w] == nil {
				freq[w] = map[string]bool{}
			}
			freq[w][rid] = true
		}
	}
	for w, ids := range freq {
		if len(ids) > 0 {
			a.Themes = append(a.Themes, Theme{w, len(ids)})
		}
	}
	sort.Slice(a.Themes, func(i, j int) bool {
		if a.Themes[i].Count == a.Themes[j].Count {
			return a.Themes[i].Word < a.Themes[j].Word
		}
		return a.Themes[i].Count > a.Themes[j].Count
	})
	if len(a.Themes) > 12 {
		a.Themes = a.Themes[:12]
	}
	return nil
}

func (s *Server) computeAnalytics(r *http.Request, id string) (Analytics, error) {
	var a Analytics
	var b []byte
	err := s.db.QueryRow(r.Context(), "SELECT report FROM survey_reports WHERE survey_id=$1", id).Scan(&b)
	if err == nil {
		if err = json.Unmarshal(b, &a); err != nil {
			return a, err
		}
	} else if err == pgx.ErrNoRows {
		a, err = s.computeCoreAnalytics(r, id)
		if err != nil {
			return a, err
		}
		if a.Status == "closed" {
			if _, err = s.db.Exec(r.Context(), "INSERT INTO survey_reports(survey_id,report,aggregate_only) VALUES($1,$2,true) ON CONFLICT DO NOTHING", id, marshal(a)); err != nil {
				return a, err
			}
			if err = s.db.QueryRow(r.Context(), "SELECT report FROM survey_reports WHERE survey_id=$1", id).Scan(&b); err != nil {
				return a, err
			}
			json.Unmarshal(b, &a)
		}
	} else {
		return a, err
	}
	if a.Status == "closed" && !a.Overall.Suppressed {
		err = s.populateThemes(r, &a)
	}
	return a, err
}
