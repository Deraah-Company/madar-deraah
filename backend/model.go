package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math"
	"sort"
	"strings"
	"time"
)

type Department struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type Profile struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	Title      string `json:"title"`
	Employee   string `json:"employee"`
	Department string `json:"department"`
	Bio        string `json:"bio"`
	Photo      string `json:"photo"`
}
type Account struct {
	Profile
	ID      string          `json:"id"`
	Status  string          `json:"status"`
	System  bool            `json:"system"`
	Trainer string          `json:"trainer"`
	Voice   string          `json:"voice"`
	Family  string          `json:"family,omitempty"`
	Grants  map[string]bool `json:"grants"`
	Secret  string          `json:"secret,omitempty"`
	Epoch   int             `json:"epoch,omitempty"`
}
type SessionKey struct {
	Account string
	Epoch   int
	Expires time.Time
}
type Audit struct {
	At     string `json:"at"`
	By     string `json:"by"`
	Action string `json:"action"`
	Target string `json:"target"`
}
type Asset struct {
	ID    string `json:"id"`
	Owner string `json:"owner"`
	Kind  string `json:"kind"`
	Name  string `json:"name"`
	MIME  string `json:"mime"`
}
type Criterion struct {
	ID     string  `json:"id"`
	Label  string  `json:"label"`
	Weight float64 `json:"weight"`
}
type ApplicationQuestion struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Required bool   `json:"required"`
}
type Campaign struct {
	ID        string                `json:"id"`
	Title     string                `json:"title"`
	Goal      string                `json:"goal"`
	Policy    string                `json:"policy"`
	Opens     string                `json:"opens"`
	Deadline  string                `json:"deadline"`
	Status    string                `json:"status"`
	Panel     []string              `json:"panel"`
	Rubric    []Criterion           `json:"rubric"`
	Questions []ApplicationQuestion `json:"questions"`
}
type Evaluation struct {
	By      string         `json:"by"`
	Ratings map[string]int `json:"ratings"`
	Note    string         `json:"note"`
	Score   float64        `json:"score"`
	At      string         `json:"at"`
}
type GuestApplicant struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Employee string `json:"employee"`
}

type Application struct {
	Guest       *GuestApplicant   `json:"guest,omitempty"`
	ID          string            `json:"id"`
	User        string            `json:"user"`
	Campaign    string            `json:"campaign"`
	Topic       string            `json:"topic"`
	Outline     string            `json:"outline"`
	Material    string            `json:"material"`
	Department  string            `json:"department"`
	Status      string            `json:"status"`
	At          string            `json:"at"`
	Appointment string            `json:"appointment"`
	Decision    string            `json:"decision"`
	Reviews     []Evaluation      `json:"reviews"`
	Answers     map[string]string `json:"answers"`
}
type QuizItem struct {
	Question string   `json:"question"`
	Options  []string `json:"options"`
	Answer   int      `json:"answer"`
}
type Training struct {
	Published     *bool              `json:"published,omitempty"`
	ID            string             `json:"id"`
	Application   string             `json:"application"`
	Title         string             `json:"title"`
	Objective     string             `json:"objective"`
	At            string             `json:"at"`
	Duration      int                `json:"duration"`
	Status        string             `json:"status"`
	Quiz          []QuizItem         `json:"quiz"`
	Feedback      map[string]float64 `json:"feedback"`
	FeedbackItems map[string][]int   `json:"feedbackItems"`
	Tests         map[string]float64 `json:"tests"`
	AdminScore    *float64           `json:"adminScore"`
	AdminNote     string             `json:"adminNote"`
}
type SurveyQuestion struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Theme string `json:"theme"`
}
type Survey struct {
	ID         string           `json:"id"`
	Cycle      string           `json:"cycle"`
	Title      string           `json:"title"`
	Benefit    string           `json:"benefit"`
	Status     string           `json:"status"`
	Token      string           `json:"token"`
	Published  bool             `json:"published"`
	Questions  []SurveyQuestion `json:"questions"`
	Count      int              `json:"count"`
	Totals     map[string]int   `json:"totals"`
	Promoters  int              `json:"promoters"`
	Detractors int              `json:"detractors"`
}
type Plan struct {
	ID        string `json:"id"`
	Survey    string `json:"survey"`
	Theme     string `json:"theme"`
	Action    string `json:"action"`
	Owner     string `json:"owner"`
	Stage     string `json:"stage"`
	Progress  *int   `json:"progress"`
	Published bool   `json:"published"`
}
type Leaderboard struct {
	Visible bool           `json:"visible"`
	Weights map[string]int `json:"weights"`
	Title   string         `json:"title"`
}
type Data struct {
	Assessments       []Assessment
	Notifications     []Notification
	NotificationReads map[string]map[string]bool
	DemoInstalled     bool
	Registration      bool
	Accounts          map[string]Account
	Sessions          map[string]SessionKey
	Departments       []Department
	Audit             []Audit
	Assets            map[string]Asset
	Campaigns         []Campaign
	Applications      []Application
	Trainings         []Training
	Surveys           []Survey
	Plans             []Plan
	Board             Leaderboard
	Bulletins         []Bulletin
	FamilyFeatured    string
	FamilyIntro       *FamilyFile
	ClosedPrograms    map[string]bool
}

var programKeys = []string{"trainer", "voice", "family"}

func programOpen(d *Data, area string) bool { return !d.ClosedPrograms[area] }

func programAvailability(d *Data) map[string]bool {
	available := make(map[string]bool, len(programKeys))
	for _, key := range programKeys {
		available[key] = programOpen(d, key)
	}
	return available
}

var permissionLabels = map[string]string{
	"trainer.assess": "إدارة استبيانات الجلسات والاختبارات وروابطها",
	"voice.review":   "مراجعة نصوص التعليقات المجهولة",
	"trainer.view":   "دخول سفراء المعرفة", "trainer.apply": "التقديم وحضور الجلسات", "trainer.campaign": "إدارة فتح التقديم والسياسات واللجان", "trainer.evaluate": "تقييم المرشحين المكلّف بهم", "trainer.decide": "متابعة الطلبات والبت فيها", "trainer.session": "إدارة الجلسات والاختبارات", "trainer.report": "عرض تقارير الإدارات والجلسات", "trainer.rate": "تقييم المدرب بعد الجلسة", "trainer.board": "إدارة لوحة المنافسة",
	"voice.view": "دخول نبض درعه", "voice.campaign": "إدارة جولات الاستبيان وروابطها", "voice.results": "عرض النتائج المجمعة", "voice.publish": "نشر النتائج للموظفين", "voice.plan": "إدارة خطط التحسين ونشرها",
	"family.view": "دخول Deraah Family", "family.comment": "التعليق على النشرات", "family.manage": "رفع النشرات وتعديلها وإدارتها", "family.moderate": "حذف التعليقات",
}
var trainerRoles = map[string][]string{
	"none": {}, "employee": {"view", "apply"}, "reviewer": {"view", "apply", "evaluate"}, "coordinator": {"view", "apply", "campaign", "decide", "session", "report"}, "admin": {"view", "apply", "campaign", "evaluate", "decide", "session", "report", "rate", "board"},
}
var voiceRoles = map[string][]string{
	"none": {}, "employee": {"view"}, "analyst": {"view", "results"}, "coordinator": {"view", "campaign", "results", "plan"}, "admin": {"view", "campaign", "results", "publish", "plan"},
}
var familyRoles = map[string][]string{
	"none": {}, "employee": {"view", "comment"}, "moderator": {"view", "comment", "moderate"}, "admin": {"view", "comment", "manage", "moderate"},
}

func permissions(a Account) map[string]bool {
	p := map[string]bool{}
	for _, k := range trainerRoles[a.Trainer] {
		p["trainer."+k] = true
	}
	for _, k := range voiceRoles[a.Voice] {
		p["voice."+k] = true
	}
	if a.Voice == "admin" {
		p["voice.review"] = true
	}
	if a.Trainer == "admin" || a.Trainer == "coordinator" {
		p["trainer.assess"] = true
	}
	family := a.Family
	if family == "" {
		family = "employee"
	}
	for _, k := range familyRoles[family] {
		p["family."+k] = true
	}
	for k, v := range a.Grants {
		if _, ok := permissionLabels[k]; ok {
			p[k] = v
		}
	}
	if a.System {
		for k := range permissionLabels {
			p[k] = true
		}
	}
	for k := range permissionLabels {
		area, _, _ := strings.Cut(k, ".")
		if !p[area+".view"] {
			p[k] = false
		}
	}
	return p
}
func clean(a Account) Account { a.Secret = ""; a.Epoch = 0; return a }
func nonce() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func timestamp() string { return time.Now().UTC().Format(time.RFC3339) }
func bounded(s string, n int) bool {
	return len([]rune(strings.TrimSpace(s))) > 0 && len([]rune(s)) <= n
}
func round(v float64) float64 { return math.Round(v*100) / 100 }
func mean(m map[string]float64) *float64 {
	if len(m) == 0 {
		return nil
	}
	sum := 0.0
	for _, v := range m {
		sum += v
	}
	v := round(sum / float64(len(m)))
	return &v
}
func rubric(cs []Criterion, ratings map[string]int) (float64, error) {
	if len(cs) == 0 || len(cs) != len(ratings) {
		return 0, errors.New("أكمل كل معايير التقييم")
	}
	sum := 0.0
	for _, c := range cs {
		v := ratings[c.ID]
		if v < 1 || v > 5 {
			return 0, errors.New("درجة البند من 1 إلى 5")
		}
		sum += float64(v) * c.Weight / 5
	}
	return round(sum), nil
}
func campaignFor(s *Data, id string) *Campaign {
	for i := range s.Campaigns {
		if s.Campaigns[i].ID == id {
			return &s.Campaigns[i]
		}
	}
	return nil
}
func appFor(s *Data, id string) *Application {
	for i := range s.Applications {
		if s.Applications[i].ID == id {
			return &s.Applications[i]
		}
	}
	return nil
}
func trainingFor(s *Data, id string) *Training {
	for i := range s.Trainings {
		if s.Trainings[i].ID == id {
			return &s.Trainings[i]
		}
	}
	return nil
}
func surveyFor(s *Data, id string) *Survey {
	for i := range s.Surveys {
		if s.Surveys[i].ID == id {
			return &s.Surveys[i]
		}
	}
	return nil
}
func includes(ids []string, id string) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}
func deptName(s *Data, id string) string {
	for _, v := range s.Departments {
		if v.ID == id {
			return v.Name
		}
	}
	return ""
}
func presentScore(a Application, c *Campaign) *float64 {
	if c == nil || len(c.Panel) == 0 {
		return nil
	}
	scores := map[string]float64{}
	for _, e := range a.Reviews {
		if includes(c.Panel, e.By) {
			scores[e.By] = e.Score
		}
	}
	if len(scores) != len(c.Panel) {
		return nil
	}
	return mean(scores)
}

type BoardRow struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	Photo      string              `json:"photo"`
	Department string              `json:"department"`
	Scores     map[string]*float64 `json:"scores"`
}

func boardFor(d *Data, a Account) []BoardRow {
	if d.Board.Visible || a.System {
		return leaderboard(d)
	}
	return []BoardRow{}
}

func leaderboard(s *Data) []BoardRow {
	rows := map[string]BoardRow{}
	all := map[string]map[string]map[string]float64{}
	add := func(user, metric, key string, v *float64) {
		if v == nil {
			return
		}
		if all[user] == nil {
			all[user] = map[string]map[string]float64{}
		}
		if all[user][metric] == nil {
			all[user][metric] = map[string]float64{}
		}
		all[user][metric][key] = *v
	}
	for _, a := range s.Applications {
		add(a.User, "panel", a.ID, presentScore(a, campaignFor(s, a.Campaign)))
	}
	for _, t := range s.Trainings {
		a := appFor(s, t.Application)
		if a == nil || a.Status != "accepted" || t.Status != "closed" {
			continue
		}
		add(a.User, "survey", t.ID, mean(t.Feedback))
		add(a.User, "quiz", t.ID, mean(t.Tests))
		add(a.User, "manager", t.ID, t.AdminScore)
	}
	for user, metrics := range all {
		a, ok := s.Accounts[user]
		if !ok || a.Status != "active" {
			continue
		}
		row := BoardRow{a.ID, a.Name, a.Photo, deptName(s, a.Department), map[string]*float64{}}
		sum, total, complete := 0.0, 0, true
		for _, k := range []string{"panel", "survey", "quiz", "manager"} {
			v := mean(metrics[k])
			row.Scores[k] = v
			w := s.Board.Weights[k]
			if w > 0 {
				total += w
				if v == nil {
					complete = false
				} else {
					sum += float64(w) * (*v)
				}
			}
		}
		if complete && total > 0 {
			v := round(sum / float64(total))
			row.Scores["combined"] = &v
		}
		rows[user] = row
	}
	out := []BoardRow{}
	for _, r := range rows {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
