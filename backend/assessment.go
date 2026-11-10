package main

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"time"
)

type AssessmentQuestion struct {
	Kind    string   `json:"kind"`
	Text    string   `json:"text"`
	Options []string `json:"options,omitempty"`
	Answer  int      `json:"answer"`
	Scale   int      `json:"scale,omitempty"`
}

type AssessmentResult struct {
	Score   float64 `json:"score"`
	Choices []int   `json:"choices"`
	Note    string  `json:"note,omitempty"`
	At      string  `json:"at"`
}

type Assessment struct {
	ID        string                      `json:"id"`
	Kind      string                      `json:"kind"`
	Title     string                      `json:"title"`
	Session   string                      `json:"session"`
	Status    string                      `json:"status"`
	Minutes   int                         `json:"minutes"`
	Questions []AssessmentQuestion        `json:"questions"`
	Started   map[string]time.Time        `json:"started,omitempty"`
	Results   map[string]AssessmentResult `json:"results,omitempty"`
}

const recommendationQuestion = "ما مدى احتمالية أن توصي بهذه الجلسة لزميل في العمل؟"

var feedbackTemplate = []AssessmentQuestion{
	{Kind: "rating", Text: "كانت أهداف الجلسة واضحة من البداية."},
	{Kind: "rating", Text: "المحتوى مناسب لمواقف أواجهها في العمل."},
	{Kind: "rating", Text: "الأمثلة والأنشطة ساعدتني على الفهم والتطبيق."},
	{Kind: "rating", Text: "رتّب المدرب الشرح وقدّمه بوضوح."},
	{Kind: "rating", Text: "أتاح المدرب فرصة للمشاركة والإجابة عن الأسئلة."},
	{Kind: "rating", Text: "أستطيع تطبيق خطوة محددة مما تعلمته في عملي."},
	{Kind: "nps", Text: recommendationQuestion},
}

func assessmentFor(d *Data, id string) *Assessment {
	for i := range d.Assessments {
		if d.Assessments[i].ID == id {
			return &d.Assessments[i]
		}
	}
	return nil
}

func sessionAssessment(d *Data, id, kind string) *Assessment {
	for i := range d.Assessments {
		a := &d.Assessments[i]
		if a.Session == id && a.Kind == kind {
			return a
		}
	}
	return nil
}

func assessmentEligible(d *Data, user Account, a *Assessment) bool {
	if !can(user, "trainer.apply") {
		return false
	}
	if s := trainingFor(d, a.Session); s != nil {
		ap := appFor(d, s.Application)
		return s.Status != "closed" && ap != nil && ap.User != user.ID
	}
	return a.Session == ""
}

func validateAssessment(a *Assessment) error {
	if !bounded(a.Title, 160) || !includes([]string{"survey", "quiz"}, a.Kind) || a.Minutes < 0 || a.Minutes > 240 {
		return bad("راجع عنوان الأداة ومدتها")
	}
	if a.Kind == "survey" {
		a.Minutes = 0
		if len(a.Questions) == 0 {
			a.Questions = append([]AssessmentQuestion(nil), feedbackTemplate...)
		}
	}
	if len(a.Questions) < 1 || len(a.Questions) > 40 {
		return bad("أضف من سؤال واحد إلى 40 سؤالاً")
	}
	hasNPS := false
	for i := range a.Questions {
		q := &a.Questions[i]
		if !bounded(q.Text, 300) {
			return bad("راجع صياغة الأسئلة")
		}
		if a.Kind == "survey" {
			switch q.Kind {
			case "", "rating":
				if q.Scale != 0 && q.Scale != 5 && q.Scale != 10 {
					return bad("اختر مقياساً من 1 إلى 5 أو من 1 إلى 10")
				}
				scale := 0
				if q.Scale == 10 {
					scale = 10
				}
				*q = AssessmentQuestion{Kind: "rating", Text: q.Text, Scale: scale}
			case "nps":
				if hasNPS {
					return bad("يكفي سؤال توصية واحد لكل استبيان")
				}
				hasNPS = true
				*q = AssessmentQuestion{Kind: "nps", Text: q.Text}
			default:
				return bad("نوع سؤال الاستبيان غير مدعوم")
			}
			continue
		}
		if q.Kind == "truefalse" {
			q.Options = []string{"صح", "غلط"}
		}
		if !includes([]string{"choice", "truefalse"}, q.Kind) || len(q.Options) < 2 || len(q.Options) > 6 || q.Answer < 0 || q.Answer >= len(q.Options) {
			return bad("اختر الإجابة الصحيحة وأضف من خيارين إلى ستة خيارات")
		}
		seen := map[string]bool{}
		for i, option := range q.Options {
			option = strings.TrimSpace(option)
			if !bounded(option, 180) || seen[option] {
				return bad("خيارات كل سؤال مطلوبة ويجب أن تكون مختلفة")
			}
			seen[option] = true
			q.Options[i] = option
		}
	}
	return nil
}

func syncAssessment(d *Data, a *Assessment) {
	s := trainingFor(d, a.Session)
	if s == nil {
		return
	}
	dst := &s.Tests
	if a.Kind == "survey" {
		if !hasRatingQuestion(a) {
			return
		}
		dst = &s.Feedback
	}
	if *dst == nil {
		*dst = map[string]float64{}
	}
	ap := appFor(d, s.Application)
	for id, result := range a.Results {
		if ap == nil || id != ap.User {
			(*dst)[id] = result.Score
		}
	}
}

func assessmentAction(d *Data, actor Account, action string, raw json.RawMessage) error {
	if action == "trainer.assessmentSave" {
		if !can(actor, "trainer.assess") {
			return forbidden()
		}
		var in Assessment
		if err := payload(raw, &in); err != nil {
			return err
		}
		if err := validateAssessment(&in); err != nil {
			return err
		}
		if in.Session != "" {
			s := trainingFor(d, in.Session)
			if s == nil {
				return missing()
			}
			if other := sessionAssessment(d, in.Session, in.Kind); other != nil && other.ID != in.ID {
				return bad("الجلسة مرتبطة بالفعل بأداة من نفس النوع")
			}
		}
		old := assessmentFor(d, in.ID)
		if in.ID != "" && old == nil {
			return missing()
		}
		in.Status, in.Started, in.Results = "draft", nil, nil
		if old == nil {
			if s := trainingFor(d, in.Session); s != nil && s.Status == "closed" {
				return bad("اربط أداة منتهية إذا كانت الجلسة انتهت")
			}
			in.ID = nonce()
			d.Assessments = append(d.Assessments, in)
		} else {
			if old.Kind != in.Kind || old.Session != "" && old.Session != in.Session && (old.Status != "draft" || len(old.Results) > 0) {
				return bad("لا يمكن تغيير نوع الأداة أو فصلها بعد إتاحتها")
			}
			if old.Status != "draft" && (old.Minutes != in.Minutes || !reflect.DeepEqual(old.Questions, in.Questions)) {
				return bad("لا يمكن تغيير الأسئلة أو المدة بعد إتاحة الأداة")
			}
			if s := trainingFor(d, in.Session); s != nil && s.Status == "closed" && old.Status != "closed" {
				return bad("أنهِ الأداة قبل ربطها بجلسة منتهية")
			}
			in.Status, in.Started, in.Results = old.Status, old.Started, old.Results
			*old = in
		}
		syncAssessment(d, &in)
		audit(d, actor, action, in.ID)
		return nil
	}
	var in struct {
		ID      string
		Status  string
		Choices []*int
		Note    string
	}
	if err := payload(raw, &in); err != nil {
		return err
	}
	a := assessmentFor(d, in.ID)
	if a == nil {
		return missing()
	}
	if action == "trainer.assessmentStatus" {
		if !can(actor, "trainer.assess") {
			return forbidden()
		}
		switch in.Status {
		case "open":
			if a.Status == "closed" {
				return bad("الأداة المنتهية لا تستقبل مشاركات جديدة")
			}
			if s := trainingFor(d, a.Session); s != nil && s.Status == "closed" {
				return bad("الجلسة المرتبطة منتهية")
			}
		case "closed":
			if a.Status == "draft" {
				return bad("أتِح الأداة قبل إنهائها")
			}
		default:
			return bad("حالة الأداة غير صحيحة")
		}
		if a.Status != in.Status {
			a.Status = in.Status
			audit(d, actor, action, a.ID)
		}
		return nil
	}
	if !assessmentEligible(d, actor, a) {
		return forbidden()
	}
	if a.Status != "open" {
		return bad("المشاركة غير متاحة الآن")
	}
	if _, done := a.Results[actor.ID]; done {
		return bad("سبق إرسال مشاركتك")
	}
	now := time.Now().UTC()
	if action == "trainer.assessmentStart" {
		if a.Kind != "quiz" {
			return bad("هذه الأداة ليست اختباراً")
		}
		if a.Started == nil {
			a.Started = map[string]time.Time{}
		}
		if a.Started[actor.ID].IsZero() {
			a.Started[actor.ID] = now
		}
		return nil
	}
	if action != "trainer.assessmentSubmit" {
		return missing()
	}
	if a.Kind == "quiz" {
		start := a.Started[actor.ID]
		if start.IsZero() || a.Minutes > 0 && !now.Before(start.Add(time.Duration(a.Minutes)*time.Minute)) {
			return bad("ابدأ الاختبار أولاً أو تحقق من انتهاء وقته")
		}
	}
	if len(in.Choices) != len(a.Questions) || len([]rune(in.Note)) > 1000 {
		return bad("أكمل كل الأسئلة")
	}
	total := 0.0
	scored := 0
	choices := make([]int, len(in.Choices))
	for i, response := range in.Choices {
		if response == nil {
			return bad("أكمل كل الأسئلة")
		}
		choice := *response
		choices[i] = choice
		q := a.Questions[i]
		if a.Kind == "survey" {
			minimum, maximum := surveyRange(q)
			if choice < minimum || choice > maximum {
				return bad("اختر درجة ضمن مقياس كل سؤال")
			}
			if q.Kind != "nps" {
				total += float64(choice) / float64(maximum)
				scored++
			}
		} else {
			if choice < 0 || choice >= len(q.Options) {
				return bad("راجع إجابات الاختبار")
			}
			scored++
			if choice == q.Answer {
				total++
			}
		}
	}
	score := 0.0
	if scored > 0 {
		score = round(total * 100 / float64(scored))
	}
	if a.Results == nil {
		a.Results = map[string]AssessmentResult{}
	}
	a.Results[actor.ID] = AssessmentResult{score, choices, strings.TrimSpace(in.Note), timestamp()}
	syncAssessment(d, a)
	return nil
}

func assessmentView(d *Data, a *Assessment, user Account, manage bool) map[string]any {
	out := map[string]any{"id": a.ID, "kind": a.Kind, "title": a.Title, "session": a.Session, "status": a.Status, "minutes": a.Minutes}
	if manage {
		out["questions"] = a.Questions
		results := []map[string]any{}
		for id, r := range a.Results {
			name := d.Accounts[id].Name
			if name == "" {
				name = "حساب محذوف"
			}
			results = append(results, map[string]any{"id": id, "name": name, "score": r.Score, "note": r.Note, "at": r.At, "choices": r.Choices})
		}
		sort.Slice(results, func(i, j int) bool { return results[i]["at"].(string) > results[j]["at"].(string) })
		out["results"] = results
		if a.Kind == "survey" {
			out["analytics"] = summarizeSurvey(a)
		}
		return out
	}
	out["eligible"] = assessmentEligible(d, user, a)
	if result, ok := a.Results[user.ID]; ok {
		out["done"] = true
		if a.Kind == "quiz" {
			out["score"] = result.Score
		}
		return out
	}
	if a.Status != "open" || !assessmentEligible(d, user, a) {
		return out
	}
	start := a.Started[user.ID]
	out["started"] = !start.IsZero()
	if !start.IsZero() && a.Minutes > 0 {
		out["endsAt"] = start.Add(time.Duration(a.Minutes) * time.Minute)
	}
	if a.Kind == "survey" || !start.IsZero() {
		questions := []map[string]any{}
		for _, q := range a.Questions {
			questions = append(questions, map[string]any{"kind": q.Kind, "text": q.Text, "options": q.Options, "scale": q.Scale})
		}
		out["questions"] = questions
	}
	return out
}
