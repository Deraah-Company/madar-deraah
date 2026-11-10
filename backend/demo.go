package main

import (
	"encoding/json"
	"fmt"
	"time"
)

const demoPDF = "demo-family-bulletin"
const demoVideo = "demo-family-video"

// installDemo runs inside the same transaction as other administrator actions.
func installDemo(d *Data, admin Account, password string) error {
	if d.DemoInstalled {
		return nil
	}
	if !goodPassword(password) {
		return bad("اختر كلمة مرور للحسابات التجريبية من 12 إلى 256 حرفاً")
	}
	if passwordMatches(password, admin.Secret) {
		return bad("اختر كلمة مرور مختلفة عن كلمة مرور مدير النظام")
	}

	now := time.Now()
	if len(d.Departments) == 0 {
		return bad("أضف إدارة واحدة على الأقل قبل إضافة البيانات التجريبية")
	}
	riyadh := now.In(time.FixedZone("Riyadh", 3*60*60))
	accounts := make([]Account, 0, 10)
	for i := 1; i <= 10; i++ {
		role := "employee"
		title := "موظف تجريبي"
		if i == 9 || i == 10 {
			role, title = "reviewer", "عضو لجنة تجريبي"
		}
		account := Account{ID: nonce(), Status: "active", Trainer: role, Voice: "employee", Family: "employee"}
		profile := Profile{
			Name: fmt.Sprintf("موظف تجريبي %02d", i), Email: fmt.Sprintf("demo%02d@deraah.test", i),
			Title: title, Employee: fmt.Sprintf("DEMO-%02d", i),
			Department: d.Departments[(i-1)%len(d.Departments)].ID,
		}
		if err := setProfile(d, &account, profile); err != nil {
			return err
		}
		account.Secret = secret(password)
		accounts = append(accounts, account)
	}
	for _, account := range accounts {
		d.Accounts[account.ID] = account
	}

	trainer := func(actor Account, action string, value any) error {
		raw, err := json.Marshal(value)
		if err != nil {
			return err
		}
		return trainerAction(d, actor, action, raw)
	}
	panel := accounts[8:]
	campaign := Campaign{
		Title: "تجريبي · شارك خبرتك مع درعه", Goal: "تجربة تقديم موضوع وتقييمه وتحويله إلى جلسة داخلية.",
		Policy: "تقيّم اللجنة وضوح الموضوع وقابليته للتطبيق قبل اتخاذ القرار.",
		Opens:  riyadh.Format("2006-01-02"), Deadline: riyadh.AddDate(0, 0, 30).Format("2006-01-02"),
		Status: "open", Panel: []string{panel[0].ID, panel[1].ID},
		Rubric:    []Criterion{{"clarity", "وضوح الفكرة", 50}, {"impact", "التطبيق في العمل", 50}},
		Questions: []ApplicationQuestion{{"benefit", "كيف يستفيد الزملاء من الموضوع؟", true}},
	}
	if err := trainer(admin, "trainer.campaign", campaign); err != nil {
		return err
	}
	campaignID := d.Campaigns[len(d.Campaigns)-1].ID
	topics := []string{"تسليم الوردية", "قراءة مؤشرات المبيعات", "التعامل مع اعتراضات العملاء", "تنظيم طلبات الفروع"}
	var accepted string
	for i, topic := range topics {
		if err := trainer(accounts[i], "trainer.apply", map[string]any{
			"campaign": campaignID, "topic": "تجريبي · " + topic, "outline": "عرض مثال من العمل ثم تطبيق قصير مع الزملاء.",
			"answers": map[string]string{"benefit": "يتعلم الزملاء تطبيق الخطوات على مواقف مشابهة في عملهم."},
		}); err != nil {
			return err
		}
		id := d.Applications[len(d.Applications)-1].ID
		if i < 3 {
			at := riyadh.AddDate(0, 0, i+1).Format("2006-01-02T15:04")
			if err := trainer(admin, "trainer.appointment", map[string]string{"id": id, "appointment": at}); err != nil {
				return err
			}
		}
		if i >= 2 {
			continue
		}
		rating, decision, note := 5, "accepted", "الموضوع واضح ويمكن تطبيقه في الوردية التالية."
		if i == 1 {
			rating, decision, note = 2, "declined", "يحتاج الموضوع إلى أمثلة عملية أكثر قبل إعادة التقديم."
		}
		for _, reviewer := range panel {
			if err := trainer(reviewer, "trainer.evaluate", map[string]any{
				"id": id, "ratings": map[string]int{"clarity": rating, "impact": rating - 1}, "note": note,
			}); err != nil {
				return err
			}
		}
		if err := trainer(admin, "trainer.decide", map[string]string{"id": id, "status": decision, "note": note}); err != nil {
			return err
		}
		if decision == "accepted" {
			accepted = id
		}
	}
	session := Training{
		Application: accepted, Title: "تجريبي · تسليم الوردية", Objective: "تسليم الملاحظات والمهام المعلقة بوضوح للفريق التالي.",
		At: riyadh.Format("2006-01-02T15:04"), Duration: 30, Status: "open",
		Quiz: []QuizItem{
			{"ماذا تسجل قبل تسليم الوردية؟", []string{"المهام المعلقة والمسؤول عنها", "عدد أفراد الفريق فقط"}, 0},
			{"كيف تتأكد أن زميلك استلم المهام؟", []string{"تكتفي بترك الملاحظات", "تراجعها معه وتتأكد من الخطوة التالية"}, 1},
		},
	}
	if err := trainer(admin, "trainer.session", session); err != nil {
		return err
	}
	sessionID := d.Trainings[len(d.Trainings)-1].ID
	for _, attendee := range accounts[4:6] {
		if err := trainer(attendee, "trainer.feedback", map[string]any{"id": sessionID, "ratings": []int{5, 5, 4, 4}}); err != nil {
			return err
		}
		if err := trainer(attendee, "trainer.quiz", map[string]any{"id": sessionID, "answers": []int{0, 1}}); err != nil {
			return err
		}
	}
	if err := trainer(admin, "trainer.rate", map[string]any{"id": sessionID, "score": 92.3, "note": "مثال تجريبي لتقييم السفير."}); err != nil {
		return err
	}

	questions := []SurveyQuestion{
		{"communication", "تصلني التحديثات التي أحتاجها في الوقت المناسب.", "التواصل الداخلي"},
		{"support", "أجد الدعم من فريقي عند الحاجة.", "دعم الفريق"},
		{"learning", "تتوفر لي فرص للتعلم وتطوير مهاراتي.", "التعلم والتطوير"},
	}
	survey := Survey{ID: nonce(), Cycle: "demo", Title: "تجريبي · تجربة العمل في درعه",
		Benefit: "تعرّف على طريقة قراءة النتائج المجمعة وربطها بخطة تحسين.", Status: "closed", Published: true,
		Token: nonce(), Questions: questions, Totals: map[string]int{}}
	for i, enps := range []int{10, 9, 10, 9, 9, 7, 8, 6, 4, 2} {
		ratings := map[string]int{}
		for j, q := range questions {
			ratings[q.ID] = 2 + (i+j)%4
		}
		if err := recordSurveyAnswer(&survey, ratings, enps); err != nil {
			return err
		}
	}
	openSurvey := Survey{ID: nonce(), Cycle: "demo", Title: "تجريبي · جرّب تعبئة الاستبيان",
		Benefit: "رابط مفتوح لتجربة المشاركة المجهولة بنفسك.", Status: "open", Token: nonce(),
		Questions: questions[:min(3, len(questions))], Totals: map[string]int{}}
	d.Surveys = append(d.Surveys, survey, openSurvey)
	progress := 65
	d.Plans = append(d.Plans, Plan{ID: nonce(), Survey: survey.ID, Theme: questions[0].Theme,
		Action: "تجريبي · مشاركة تحديث أسبوعي ومراجعة ملاحظات الفريق بعد شهر.",
		Owner:  "فريق التحسين التجريبي", Stage: "working", Progress: &progress, Published: true})

	at := now.UTC().Format(time.RFC3339)
	pdf := &FamilyFile{demoPDF, "Deraah Family - Demo Bulletin.pdf"}
	// Existing published bulletins keep precedence for the current issue.
	d.Bulletins = append([]Bulletin{
		{ID: nonce(), Title: "تجريبي · من أرشيف Deraah Family", Summary: "عدد سابق للتجربة والرجوع إليه من الأرشيف.",
			At: now.AddDate(0, -1, 0).UTC().Format(time.RFC3339), PDF: pdf},
		{ID: nonce(), Title: "تجريبي · أخبار فرق درعه", Summary: "مثال لنشرة داخلية. تصفّح ملف النشرة، وشاهد برومو مدار درعه كمرفق فيديو للتجربة، ثم جرّب التعليق.",
			At: at, PDF: pdf, Video: &FamilyFile{demoVideo, "Madar Deraah - Promo.mp4"}, Comments: []FamilyComment{
				{nonce(), accounts[4].ID, "فكرة ممتازة إننا نرجع للأعداد السابقة من الأرشيف.", at},
				{nonce(), accounts[5].ID, "أقترح نشارك مبادرات الفروع في العدد الجاي.", at},
			}},
	}, d.Bulletins...)
	d.DemoInstalled = true
	audit(d, admin, "system.demo", "")
	return nil
}
