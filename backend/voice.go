package main

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
)

func initialData(email, password string) Data {
	d := Data{Registration: true, Accounts: map[string]Account{}, Sessions: map[string]SessionKey{}, Assets: map[string]Asset{}, Audit: []Audit{}, Campaigns: []Campaign{}, Applications: []Application{}, Trainings: []Training{}, Plans: []Plan{}, Board: Leaderboard{true, map[string]int{"panel": 40, "survey": 25, "quiz": 20, "manager": 15}, "لوحة المنافسة"}}
	for i, name := range []string{"التدريب والتطوير", "الموارد البشرية", "المبيعات", "العمليات", "خدمة العملاء", "التسويق", "المالية", "تقنية المعلومات", "سلاسل الإمداد", "الجودة والامتثال"} {
		d.Departments = append(d.Departments, Department{string(rune('a' + i)), name})
	}
	id := nonce()
	d.Accounts[id] = Account{Profile: Profile{Name: "مدير النظام", Email: strings.ToLower(strings.TrimSpace(email)), Title: "مدير النظام", Employee: "SYS-ADMIN", Department: "a"}, ID: id, Status: "active", System: true, Trainer: "admin", Voice: "admin", Secret: secret(password)}
	var original struct {
		Categories []struct {
			Name      struct{ Ar string }
			Questions []struct {
				ID   string
				Text struct{ Ar string }
			}
		}
	}
	raw, _ := content.ReadFile("default-survey.json")
	_ = json.Unmarshal(raw, &original)
	questions := []SurveyQuestion{}
	for _, c := range original.Categories {
		for _, q := range c.Questions {
			if q.ID != "" && q.Text.Ar != "" && c.Name.Ar != "" {
				questions = append(questions, SurveyQuestion{q.ID, q.Text.Ar, c.Name.Ar})
			}
		}
	}
	for _, cycle := range []string{"semi", "final"} {
		title := "نبض درعه · منتصف العام"
		if cycle == "final" {
			title = "نبض درعه · نهاية العام"
		}
		d.Surveys = append(d.Surveys, Survey{ID: nonce(), Cycle: cycle, Title: title, Benefit: "مشاركتك تساعدنا نحدد أولويات تحسين بيئة العمل. بعد مراجعة النتائج المجمعة، تختار الإدارة خطوات تحسين وتعرض تقدمها على المنصة.", Status: "draft", Token: nonce(), Questions: questions, Totals: map[string]int{}})
	}
	return d
}

func surveyByToken(d *Data, token string) *Survey {
	if len(token) != 48 {
		return nil
	}
	for i := range d.Surveys {
		if d.Surveys[i].Token == token {
			return &d.Surveys[i]
		}
	}
	return nil
}
func publicSurvey(w http.ResponseWriter, r *http.Request, d *Data) error {
	s := surveyByToken(d, r.URL.Query().Get("token"))
	if s == nil || s.Status == "draft" {
		return missing()
	}
	respond(w, 200, map[string]any{"title": s.Title, "cycle": s.Cycle, "benefit": s.Benefit, "status": s.Status, "questions": s.Questions})
	return nil
}
func (s *server) answerSurvey(w http.ResponseWriter, r *http.Request) error {
	if !s.limiter(r, "survey", 150) {
		return fault{429, "محاولات كثيرة. حاول بعد قليل"}
	}
	var in struct {
		Token   string         `json:"token"`
		Ratings map[string]int `json:"ratings"`
		ENPS    *int           `json:"enps"`
	}
	if e := decode(r, &in); e != nil {
		return e
	}
	// Store only totals. No account, cookies, timestamp, IP, free text, or individual answer row is written.
	e := s.write(r.Context(), func(d *Data) error {
		if !programOpen(d, "voice") {
			return missing()
		}
		v := surveyByToken(d, in.Token)
		if v == nil || v.Status != "open" {
			return bad("الاستبيان غير متاح حالياً")
		}
		if in.ENPS == nil {
			return bad("أكمل جميع الأسئلة")
		}
		return recordSurveyAnswer(v, in.Ratings, *in.ENPS)
	})
	if e != nil {
		return e
	}
	respond(w, 201, map[string]bool{"ok": true})
	return nil
}
func recordSurveyAnswer(v *Survey, ratings map[string]int, enps int) error {
	if len(v.Questions) != len(ratings) || enps < 0 || enps > 10 {
		return bad("أكمل جميع الأسئلة")
	}
	for _, q := range v.Questions {
		if ratings[q.ID] < 1 || ratings[q.ID] > 5 {
			return bad("درجة العبارة من 1 إلى 5")
		}
	}
	for _, q := range v.Questions {
		v.Totals[q.ID] += ratings[q.ID]
	}
	v.Count++
	if enps >= 9 {
		v.Promoters++
	}
	if enps <= 6 {
		v.Detractors++
	}
	return nil
}
func voiceAction(d *Data, a Account, action string, raw json.RawMessage) error {
	perm := map[string]string{"voice.survey": "voice.campaign", "voice.publish": "voice.publish", "voice.plan": "voice.plan", "voice.planDelete": "voice.plan"}[action]
	if perm == "" {
		return missing()
	}
	if !can(a, perm) {
		return forbidden()
	}
	switch action {
	case "voice.survey":
		var in Survey
		if e := payload(raw, &in); e != nil {
			return e
		}
		old := surveyFor(d, in.ID)
		if old == nil {
			return missing()
		}
		if !bounded(in.Title, 140) || !bounded(in.Benefit, 1500) || len(in.Questions) < 1 || len(in.Questions) > 60 || !includes([]string{"draft", "open", "closed"}, in.Status) {
			return bad("أكمل العنوان وقيمة المشاركة والأسئلة والحالة")
		}
		seen := map[string]bool{}
		for _, q := range in.Questions {
			if !bounded(q.ID, 80) || !bounded(q.Label, 300) || !bounded(q.Theme, 120) || seen[q.ID] {
				return bad("راجع السؤال والمحور")
			}
			seen[q.ID] = true
		}
		if old.Status == "closed" && in.Status != "closed" {
			return bad("الجولة المقفلة لا تُفتح مجدداً بعد تجميع النتائج")
		}
		if old.Count > 0 && (!reflect.DeepEqual(in.Questions, old.Questions) || in.Status == "draft") {
			return bad("الأسئلة ثابتة بعد أول مشاركة")
		}
		old.Title = in.Title
		old.Benefit = in.Benefit
		old.Status = in.Status
		old.Questions = in.Questions
		audit(d, a, action, in.ID)
	case "voice.publish":
		var in struct {
			ID        string `json:"id"`
			Published bool   `json:"published"`
		}
		if e := payload(raw, &in); e != nil {
			return e
		}
		v := surveyFor(d, in.ID)
		if v == nil {
			return missing()
		}
		if in.Published && (v.Status != "closed" || v.Count == 0) {
			return bad("أقفل الجولة وسجل مشاركة واحدة على الأقل قبل النشر")
		}
		v.Published = in.Published
		audit(d, a, action, in.ID)
	case "voice.plan":
		var in Plan
		if e := payload(raw, &in); e != nil {
			return e
		}
		s := surveyFor(d, in.Survey)
		if s == nil {
			return missing()
		}
		themeOK := false
		for _, q := range s.Questions {
			themeOK = themeOK || q.Theme == in.Theme
		}
		if !themeOK || !bounded(in.Action, 2000) || !bounded(in.Owner, 120) || !includes([]string{"planned", "working", "done"}, in.Stage) || in.Progress != nil && (*in.Progress < 0 || *in.Progress > 100) {
			return bad("راجع المحور وخطوة التحسين والمسؤول والمرحلة")
		}
		if in.ID == "" {
			in.ID = nonce()
			d.Plans = append(d.Plans, in)
		} else {
			found := false
			for i, v := range d.Plans {
				if v.ID == in.ID {
					d.Plans[i] = in
					found = true
					break
				}
			}
			if !found {
				return missing()
			}
		}
		audit(d, a, action, in.ID)
	case "voice.planDelete":
		var in struct {
			ID string `json:"id"`
		}
		if e := payload(raw, &in); e != nil {
			return e
		}
		for i, v := range d.Plans {
			if v.ID == in.ID {
				d.Plans = append(d.Plans[:i], d.Plans[i+1:]...)
				audit(d, a, action, in.ID)
				return nil
			}
		}
		return missing()
	}
	return nil
}
func voiceView(d *Data, a Account) map[string]any {
	manage := can(a, "voice.campaign")
	publish := can(a, "voice.publish")
	plan := can(a, "voice.plan")
	see := can(a, "voice.results")
	surveys := []map[string]any{}
	plans := []Plan{}
	for _, s := range d.Surveys {
		if !manage && !publish && !plan && !see && !s.Published {
			continue
		}
		item := map[string]any{"id": s.ID, "cycle": s.Cycle, "title": s.Title, "status": s.Status, "benefit": s.Benefit, "published": s.Published}
		if manage {
			item["token"] = s.Token
			item["count"] = s.Count
		}
		if manage || plan {
			item["questions"] = s.Questions
		}
		if s.Count > 0 && s.Status == "closed" && (see || s.Published) {
			themeTotals := map[string]int{}
			themeQuestions := map[string]int{}
			for _, q := range s.Questions {
				themeTotals[q.Theme] += s.Totals[q.ID]
				themeQuestions[q.Theme]++
			}
			themes := map[string]float64{}
			for k, v := range themeTotals {
				themes[k] = round(float64(v) * 20 / float64(s.Count*themeQuestions[k]))
			}
			item["results"] = map[string]any{"count": s.Count, "enps": round(float64(s.Promoters-s.Detractors) * 100 / float64(s.Count)), "themes": themes}
		}
		surveys = append(surveys, item)
	}
	for _, p := range d.Plans {
		if plan || p.Published {
			plans = append(plans, p)
		}
	}
	return map[string]any{"surveys": surveys, "plans": plans}
}
