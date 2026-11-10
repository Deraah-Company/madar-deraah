package main

import (
	"encoding/json"
	"math"
	"reflect"
	"time"
)

func dateValue(v string) bool     { _, e := time.Parse("2006-01-02", v); return e == nil }
func datetimeValue(v string) bool { _, e := time.Parse("2006-01-02T15:04", v); return e == nil }
func today() string               { return time.Now().UTC().Add(3 * time.Hour).Format("2006-01-02") }
func validCampaign(c Campaign, d *Data) error {
	if !bounded(c.Title, 120) || !bounded(c.Goal, 2000) || !bounded(c.Policy, 4000) || !dateValue(c.Opens) || !dateValue(c.Deadline) || c.Opens > c.Deadline {
		return bad("أدخل الهدف والسياسات وفترة تقديم صحيحة")
	}
	if c.Status != "draft" && c.Status != "open" && c.Status != "closed" {
		return bad("حالة التقديم غير صالحة")
	}
	if len(c.Panel) > 20 {
		return bad("الحد الأقصى للجنة التقديم 20 عضواً")
	}
	seen := map[string]bool{}
	for _, id := range c.Panel {
		a, ok := d.Accounts[id]
		if !ok || a.Status != "active" || !can(a, "trainer.evaluate") || seen[id] {
			return bad("كل عضو لجنة يجب أن يكون نشطاً ولديه صلاحية التقييم")
		}
		seen[id] = true
	}
	if len(c.Rubric) < 1 || len(c.Rubric) > 20 {
		return bad("أضف من 1 إلى 20 معياراً")
	}
	sum := 0.0
	seen = map[string]bool{}
	for _, k := range c.Rubric {
		if !bounded(k.ID, 80) || !bounded(k.Label, 120) || k.Weight <= 0 || seen[k.ID] {
			return bad("راجع بنود التقييم وأوزانها")
		}
		seen[k.ID] = true
		sum += k.Weight
	}
	if math.Abs(sum-100) > .001 {
		return bad("مجموع أوزان لجنة التقديم لازم يكون 100")
	}
	if len(c.Questions) > 20 {
		return bad("نموذج التقديم لا يتجاوز 20 سؤالاً إضافياً")
	}
	seen = map[string]bool{}
	for _, q := range c.Questions {
		if !bounded(q.ID, 80) || !bounded(q.Label, 300) || seen[q.ID] {
			return bad("راجع أسئلة نموذج التقديم")
		}
		seen[q.ID] = true
	}
	return nil
}
func trainerAction(d *Data, a Account, action string, raw json.RawMessage) error {
	switch action {
	case "trainer.assessmentSave", "trainer.assessmentStatus", "trainer.assessmentStart", "trainer.assessmentSubmit":
		return assessmentAction(d, a, action, raw)
	case "trainer.sessionPublish", "trainer.sessionClose":
		if !can(a, "trainer.session") {
			return forbidden()
		}
		var in struct{ ID string }
		if err := payload(raw, &in); err != nil {
			return err
		}
		s := trainingFor(d, in.ID)
		if s == nil {
			return missing()
		}
		if action == "trainer.sessionPublish" {
			if s.Status == "closed" || !datetimeValue(s.At) {
				return bad("راجع موعد الجلسة قبل الإعلان")
			}
			if s.Published != nil && !*s.Published {
				ap := appFor(d, s.Application)
				if ap == nil || ap.Status != "accepted" {
					return bad("اعتمد مقدم الجلسة أولاً")
				}
				*s.Published = true
				announceTraining(d, s, ap)
			}
		} else {
			if !trainingVisible(*s) {
				return bad("أعلن الجلسة أولاً")
			}
			s.Status = "closed"
			for i := range d.Assessments {
				if d.Assessments[i].Session == s.ID && d.Assessments[i].Status == "open" {
					d.Assessments[i].Status = "closed"
				}
			}
		}
		audit(d, a, action, in.ID)
		return nil
	}
	requirements := map[string]string{"trainer.campaign": "trainer.campaign", "trainer.apply": "trainer.apply", "trainer.appointment": "trainer.decide", "trainer.evaluate": "trainer.evaluate", "trainer.decide": "trainer.decide", "trainer.session": "trainer.session", "trainer.feedback": "trainer.apply", "trainer.quiz": "trainer.apply", "trainer.rate": "trainer.rate", "trainer.board": "trainer.board"}
	perm, ok := requirements[action]
	if !ok {
		return missing()
	}
	if !can(a, perm) {
		return forbidden()
	}
	switch action {
	case "trainer.campaign":
		var in Campaign
		if e := payload(raw, &in); e != nil {
			return e
		}
		if e := validCampaign(in, d); e != nil {
			return e
		}
		if in.ID == "" {
			in.ID = nonce()
			d.Campaigns = append(d.Campaigns, in)
		} else {
			old := campaignFor(d, in.ID)
			if old == nil {
				return missing()
			}
			for _, ap := range d.Applications {
				if ap.Campaign != in.ID {
					continue
				}
				if !reflect.DeepEqual(old.Rubric, in.Rubric) {
					return bad("معايير الجولة تُحفظ بعد استلام أول ترشيح. أنشئ جولة جديدة لتغييرها")
				}
				if len(old.Questions) != 0 || len(in.Questions) != 0 {
					if !reflect.DeepEqual(old.Questions, in.Questions) {
						return bad("تُحفظ أسئلة النموذج بعد استلام أول طلب")
					}
				}
				if len(ap.Reviews) > 0 && !reflect.DeepEqual(old.Panel, in.Panel) {
					return bad("لا يمكن تغيير اللجنة بعد تسجيل أول تقييم")
				}
			}
			*old = in
		}
		audit(d, a, action, in.ID)
	case "trainer.apply":
		var in trainerApplicationInput
		if e := payload(raw, &in); e != nil {
			return e
		}
		_, err := submitTrainerApplication(d, a, in, nil)
		return err
	case "trainer.appointment", "trainer.evaluate", "trainer.decide":
		var in struct {
			ID          string         `json:"id"`
			Appointment string         `json:"appointment"`
			Ratings     map[string]int `json:"ratings"`
			Note        string         `json:"note"`
			Status      string         `json:"status"`
		}
		if e := payload(raw, &in); e != nil {
			return e
		}
		ap := appFor(d, in.ID)
		if ap == nil {
			return missing()
		}
		c := campaignFor(d, ap.Campaign)
		if c == nil {
			return missing()
		}
		if ap.Status != "submitted" {
			return bad("تم البت في الطلب بالفعل")
		}
		switch action {
		case "trainer.appointment":
			if !datetimeValue(in.Appointment) {
				return bad("موعد لجنة التقديم غير صالح")
			}
			if ap.Appointment != in.Appointment {
				ap.Appointment = in.Appointment
				notify(d, Notification{Kind: "interview", Reference: ap.ID, Recipient: ap.User,
					Title: "موعد عرضك أمام اللجنة", Detail: ap.Topic, At: ap.Appointment, Href: "/knowledge?tab=calendar", Popup: true})
			}
		case "trainer.evaluate":
			if !includes(c.Panel, a.ID) || a.ID == ap.User {
				return forbidden()
			}
			score, e := rubric(c.Rubric, in.Ratings)
			if e != nil {
				return bad(e.Error())
			}
			if len([]rune(in.Note)) > 2000 {
				return bad("الملاحظة طويلة")
			}
			review := Evaluation{a.ID, in.Ratings, in.Note, score, timestamp()}
			found := false
			for i, v := range ap.Reviews {
				if v.By == a.ID {
					ap.Reviews[i] = review
					found = true
					break
				}
			}
			if !found {
				ap.Reviews = append(ap.Reviews, review)
			}
		case "trainer.decide":
			if presentScore(*ap, c) == nil {
				return bad("يجب استكمال تقييم جميع أعضاء اللجنة أولاً")
			}
			if in.Status != "accepted" && in.Status != "declined" || !bounded(in.Note, 1000) {
				return bad("اختر القرار وأضف سببه")
			}
			ap.Status = in.Status
			ap.Decision = in.Note
		}
		audit(d, a, action, ap.ID)
	case "trainer.session":
		var in Training
		if e := payload(raw, &in); e != nil {
			return e
		}
		ap := appFor(d, in.Application)
		if ap == nil || ap.Status != "accepted" {
			return bad("اختر سفيراً من الطلبات المقبولة")
		}
		if !bounded(in.Title, 160) || !bounded(in.Objective, 1000) || !datetimeValue(in.At) || in.Duration < 10 || in.Duration > 240 || in.Status != "open" && in.Status != "closed" && in.Status != "scheduled" {
			return bad("أكمل عنوان الجلسة وموعدها ومدتها وحالتها")
		}
		if len(in.Quiz) > 20 {
			return bad("الاختبار لا يتجاوز 20 سؤالاً")
		}
		for _, q := range in.Quiz {
			if !bounded(q.Question, 300) || len(q.Options) < 2 || len(q.Options) > 6 || q.Answer < 0 || q.Answer >= len(q.Options) {
				return bad("راجع أسئلة الاختبار والإجابة الصحيحة")
			}
			for _, o := range q.Options {
				if !bounded(o, 180) {
					return bad("أكمل خيارات الاختبار")
				}
			}
		}
		in.Feedback = map[string]float64{}
		in.FeedbackItems = map[string][]int{}
		in.Tests = map[string]float64{}
		in.AdminScore = nil
		in.AdminNote = ""
		if in.ID == "" {
			if in.Published != nil {
				visible := false
				in.Published, in.Status = &visible, "scheduled"
			}
			in.ID = nonce()
			d.Trainings = append(d.Trainings, in)
		} else {
			old := trainingFor(d, in.ID)
			if old == nil {
				return missing()
			}
			in.Published = old.Published
			if old.Published != nil {
				if old.Status == "closed" {
					return bad("الجلسة منتهية")
				}
				in.Status = old.Status
			}
			hasResults := len(old.Tests)+len(old.Feedback) > 0 || old.AdminScore != nil
			if hasResults && old.Application != in.Application {
				return bad("لا يمكن تغيير السفير بعد تسجيل النتائج")
			}
			if len(old.Tests) > 0 && !reflect.DeepEqual(old.Quiz, in.Quiz) {
				return bad("الاختبار لا يتغير بعد أول محاولة")
			}
			if old.Status == "closed" && in.Status != "closed" && hasResults {
				return bad("أنشئ جلسة جديدة بعد إقفال النتائج")
			}
			in.Tests = old.Tests
			in.Feedback = old.Feedback
			in.FeedbackItems = old.FeedbackItems
			in.AdminScore = old.AdminScore
			in.AdminNote = old.AdminNote
			changed := old.At != in.At || old.Title != in.Title || old.Application != in.Application
			*old = in
			if changed && old.Published != nil && *old.Published {
				announceTraining(d, old, ap)
			}
		}
		audit(d, a, action, in.ID)
	case "trainer.feedback", "trainer.quiz", "trainer.rate":
		var in struct {
			ID      string  `json:"id"`
			Ratings []int   `json:"ratings"`
			Answers []int   `json:"answers"`
			Score   float64 `json:"score"`
			Note    string  `json:"note"`
		}
		if e := payload(raw, &in); e != nil {
			return e
		}
		s := trainingFor(d, in.ID)
		if s == nil {
			return missing()
		}
		ap := appFor(d, s.Application)
		if ap == nil || a.ID == ap.User {
			return forbidden()
		}
		if action == "trainer.rate" {
			if !trainingVisible(*s) || s.Status == "scheduled" && s.Published == nil || in.Score < 0 || in.Score > 100 || !bounded(in.Note, 1000) {
				return bad("سجل تقييم الجلسة من 100 مع ملاحظة")
			}
			s.AdminScore = &in.Score
			s.AdminNote = in.Note
			audit(d, a, action, s.ID)
			return nil
		}
		if s.Status != "open" {
			return bad("تقييم هذه الجلسة غير متاح حالياً")
		}
		kind := "quiz"
		if action == "trainer.feedback" {
			kind = "survey"
		}
		if sessionAssessment(d, s.ID, kind) != nil {
			return bad("استخدم رابط الأداة المرتبطة بالجلسة")
		}
		if action == "trainer.feedback" {
			if _, done := s.Feedback[a.ID]; done {
				return bad("سبق لك تقييم الجلسة")
			}
			if len(in.Ratings) != 4 {
				return bad("أكمل عناصر الاستبيان الأربعة")
			}
			total := 0
			for _, v := range in.Ratings {
				if v < 1 || v > 5 {
					return bad("درجة العنصر من 1 إلى 5")
				}
				total += v
			}
			s.Feedback[a.ID] = float64(total) * 5
			if s.FeedbackItems == nil {
				s.FeedbackItems = map[string][]int{}
			}
			s.FeedbackItems[a.ID] = in.Ratings
			return nil
		}
		if _, done := s.Tests[a.ID]; done {
			return bad("سبق لك أداء الاختبار")
		}
		if len(s.Quiz) == 0 || len(s.Quiz) != len(in.Answers) {
			return bad("أكمل الاختبار")
		}
		correct := 0
		for i, v := range in.Answers {
			if v < 0 || v >= len(s.Quiz[i].Options) {
				return bad("إجابة غير صالحة")
			}
			if v == s.Quiz[i].Answer {
				correct++
			}
		}
		s.Tests[a.ID] = round(float64(correct) * 100 / float64(len(s.Quiz)))
	case "trainer.board":
		var in Leaderboard
		if e := payload(raw, &in); e != nil {
			return e
		}
		if !bounded(in.Title, 100) {
			return bad("عنوان لوحة المنافسة مطلوب")
		}
		sum := 0
		for k, v := range in.Weights {
			if !includes([]string{"panel", "survey", "quiz", "manager"}, k) || v < 0 || v > 100 {
				return bad("الأوزان غير صالحة")
			}
			sum += v
		}
		if sum != 100 {
			return bad("مجموع أوزان لوحة المنافسة لازم يكون 100")
		}
		d.Board = in
		audit(d, a, action, "")
	}
	return nil
}

type trainerApplicationInput struct {
	Campaign   string            `json:"campaign"`
	Topic      string            `json:"topic"`
	Outline    string            `json:"outline"`
	Department string            `json:"department"`
	Material   string            `json:"material"`
	Answers    map[string]string `json:"answers"`
}

func campaignAccepting(c *Campaign) bool {
	day := today()
	return c.Status == "open" && c.Opens <= day && c.Deadline >= day
}

func submitTrainerApplication(d *Data, a Account, in trainerApplicationInput, guest *GuestApplicant) (string, error) {
	c := campaignFor(d, in.Campaign)
	if c == nil {
		return "", missing()
	}
	if !campaignAccepting(c) {
		return "", bad("باب التقديم غير مفتوح حالياً")
	}
	if in.Department == "" {
		in.Department = a.Department
	}
	if deptName(d, in.Department) == "" {
		return "", bad("اختر إدارة صحيحة")
	}
	if includes(c.Panel, a.ID) {
		return "", bad("عضو اللجنة لا يتقدم في الجولة التي يقيّمها")
	}
	if !bounded(in.Topic, 160) || !bounded(in.Outline, 3000) {
		return "", bad("أدخل عنوان الموضوع وخطة الشرح")
	}
	for _, ap := range d.Applications {
		if ap.Campaign == c.ID && ap.User == a.ID {
			return "", bad("سبق لك التقديم في هذه الجولة")
		}
	}
	if in.Material != "" {
		if file, ok := d.Assets[in.Material]; !ok || file.Owner != a.ID || file.Kind != "material" {
			return "", forbidden()
		}
	}
	answers := map[string]string{}
	for _, q := range c.Questions {
		answer := in.Answers[q.ID]
		if q.Required && !bounded(answer, 2000) || len([]rune(answer)) > 2000 {
			return "", bad("أكمل أسئلة نموذج التقديم المطلوبة")
		}
		answers[q.ID] = answer
	}
	id := nonce()
	d.Applications = append(d.Applications, Application{ID: id, User: a.ID, Campaign: c.ID, Topic: in.Topic, Outline: in.Outline, Material: in.Material, Department: in.Department, Guest: guest, Status: "submitted", At: timestamp(), Reviews: []Evaluation{}, Answers: answers})
	audit(d, a, "trainer.apply", id)
	return id, nil
}

func trainerView(d *Data, a Account) map[string]any {
	campaigns := []Campaign{}
	apps := []map[string]any{}
	sessions := []map[string]any{}
	panel := []map[string]string{}
	manage := can(a, "trainer.campaign")
	decide := can(a, "trainer.decide")
	report := can(a, "trainer.report")
	sessionAdmin := can(a, "trainer.session")
	names := func(id string) string {
		if v, ok := d.Accounts[id]; ok {
			return v.Name
		}
		return "حساب محذوف"
	}
	for _, u := range d.Accounts {
		if u.Status == "active" && can(u, "trainer.evaluate") {
			panel = append(panel, map[string]string{"id": u.ID, "name": u.Name})
		}
	}
	for _, c := range d.Campaigns {
		if c.Status != "draft" || manage || decide || can(a, "trainer.evaluate") && includes(c.Panel, a.ID) {
			campaigns = append(campaigns, c)
		}
	}
	for _, ap := range d.Applications {
		c := campaignFor(d, ap.Campaign)
		name := names(ap.User)
		if ap.Guest != nil {
			name = ap.Guest.Name
		}
		assigned := c != nil && can(a, "trainer.evaluate") && includes(c.Panel, a.ID)
		if ap.User != a.ID && !decide && !assigned && !sessionAdmin {
			continue
		}
		if sessionAdmin && !decide && !assigned && ap.User != a.ID {
			if ap.Status == "accepted" {
				apps = append(apps, map[string]any{"id": ap.ID, "name": name, "topic": ap.Topic, "status": ap.Status, "campaign": ap.Campaign})
			}
			continue
		}
		reviews := []Evaluation{}
		for _, r := range ap.Reviews {
			if decide || r.By == a.ID && assigned {
				reviews = append(reviews, r)
			}
		}
		var score *float64
		if decide || ap.User == a.ID && ap.Status != "submitted" {
			score = presentScore(ap, c)
		}
		var previous map[string]any
		for _, past := range d.Applications {
			if past.User == ap.User && past.At < ap.At && past.Status != "submitted" && (previous == nil || past.At > previous["at"].(string)) {
				previous = map[string]any{"topic": past.Topic, "at": past.At, "status": past.Status, "score": presentScore(past, campaignFor(d, past.Campaign))}
			}
		}
		row := map[string]any{"id": ap.ID, "mine": ap.User == a.ID, "name": name, "campaign": ap.Campaign, "topic": ap.Topic, "outline": ap.Outline, "material": ap.Material, "department": deptName(d, ap.Department), "status": ap.Status, "at": ap.At, "appointment": ap.Appointment, "decision": ap.Decision, "assigned": assigned, "reviews": reviews, "reviewCount": len(ap.Reviews), "reviewComplete": presentScore(ap, c) != nil, "score": score, "answers": ap.Answers, "previous": previous}
		if decide && ap.Guest != nil {
			row["guest"] = ap.Guest
		}
		apps = append(apps, row)
	}
	for _, s := range d.Trainings {
		if !trainingVisible(s) && !sessionAdmin && !can(a, "trainer.assess") {
			continue
		}
		ap := appFor(d, s.Application)
		if ap == nil {
			continue
		}
		trainerName := names(ap.User)
		if ap.Guest != nil {
			trainerName = ap.Guest.Name
		}
		quiz := []map[string]any{}
		for _, q := range s.Quiz {
			item := map[string]any{"question": q.Question, "options": q.Options}
			if sessionAdmin {
				item["answer"] = q.Answer
			}
			quiz = append(quiz, item)
		}
		_, feedbackDone := s.Feedback[a.ID]
		feedbackCount := len(s.Feedback)
		if survey := sessionAssessment(d, s.ID, "survey"); survey != nil {
			if _, submitted := survey.Results[a.ID]; submitted {
				feedbackDone = true
			}
			for respondent := range survey.Results {
				if _, recorded := s.Feedback[respondent]; !recorded && respondent != ap.User {
					feedbackCount++
				}
			}
		}
		score, quizDone := s.Tests[a.ID]
		row := map[string]any{"id": s.ID, "application": s.Application, "trainer": trainerName, "mine": a.ID == ap.User, "title": s.Title, "objective": s.Objective, "at": s.At, "duration": s.Duration, "status": s.Status, "quiz": quiz, "feedbackDone": feedbackDone, "quizDone": quizDone}
		row["published"], row["legacy"] = trainingVisible(s), s.Published == nil
		for _, kind := range []string{"quiz", "survey"} {
			if linked := sessionAssessment(d, s.ID, kind); linked != nil {
				flag := "hasSurvey"
				if kind == "quiz" {
					flag = "hasQuiz"
				}
				row[flag] = true
				if linked.Status == "open" && assessmentEligible(d, a, linked) {
					row[kind+"Link"] = "/knowledge/assessment?id=" + linked.ID
				}
			}
		}
		if quizDone {
			row["myScore"] = score
		}
		if report || sessionAdmin || can(a, "trainer.rate") {
			row["feedbackScore"] = mean(s.Feedback)
			row["quizScore"] = mean(s.Tests)
			row["feedbackCount"] = feedbackCount
			row["quizCount"] = len(s.Tests)
			row["adminScore"] = s.AdminScore
			row["adminNote"] = s.AdminNote
		}
		if (report || sessionAdmin || can(a, "trainer.rate") || ap.User == a.ID && s.Status == "closed") && len(s.FeedbackItems) > 0 {
			dimensions := make([]float64, 4)
			for _, ratings := range s.FeedbackItems {
				for i, rating := range ratings {
					dimensions[i] += float64(rating)
				}
			}
			for i := range dimensions {
				dimensions[i] = round(dimensions[i] * 20 / float64(len(s.FeedbackItems)))
			}
			row["feedbackDimensions"] = dimensions
		}
		sessions = append(sessions, row)
	}
	counts := []map[string]any{}
	if report {
		for _, dept := range d.Departments {
			for _, c := range d.Campaigns {
				applied, accepted := 0, 0
				for _, ap := range d.Applications {
					if ap.Department == dept.ID && ap.Campaign == c.ID {
						applied++
						if ap.Status == "accepted" {
							accepted++
						}
					}
				}
				counts = append(counts, map[string]any{"id": dept.ID, "name": dept.Name, "campaign": c.ID, "applied": applied, "accepted": accepted})
			}
		}
	}
	rankings := []BoardRow{}
	if can(a, "trainer.decide") || can(a, "trainer.board") {
		rankings = boardFor(d, a)
	}
	return map[string]any{"campaigns": campaigns, "applications": apps, "sessions": sessions, "reviewers": panel, "departments": counts, "departmentOptions": d.Departments, "board": d.Board, "ranking": rankings, "calendar": calendarView(d, a)}
}
