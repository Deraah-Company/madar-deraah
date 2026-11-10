package main

import (
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
)

type publicTrainerCampaign struct {
	ID        string                `json:"id"`
	Title     string                `json:"title"`
	Goal      string                `json:"goal"`
	Policy    string                `json:"policy"`
	Deadline  string                `json:"deadline"`
	Questions []ApplicationQuestion `json:"questions"`
}

func publicTrainerView(d *Data, r *http.Request) map[string]any {
	campaigns := []publicTrainerCampaign{}
	for _, c := range d.Campaigns {
		if campaignAccepting(&c) {
			campaigns = append(campaigns, publicTrainerCampaign{c.ID, c.Title, c.Goal, c.Policy, c.Deadline, c.Questions})
		}
	}
	var account any
	if a, err := signedIn(d, r); err == nil {
		account = map[string]any{"name": a.Name, "department": a.Department, "canApply": can(a, "trainer.apply")}
	}
	return map[string]any{"campaigns": campaigns, "departments": d.Departments, "account": account}
}

func (s *server) publicTrainerSubmit(w http.ResponseWriter, r *http.Request) error {
	if !s.limiter(r, "trainer-public-apply", 40) {
		return fault{429, "محاولات كثيرة. حاول بعد قليل"}
	}
	r.Body = http.MaxBytesReader(w, r.Body, 11<<20)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		return bad("الطلب غير صالح أو المرفق أكبر من 10 ميجابايت")
	}
	defer r.MultipartForm.RemoveAll()
	in := trainerApplicationInput{
		Campaign: r.FormValue("campaign"), Topic: r.FormValue("topic"),
		Outline: r.FormValue("outline"), Department: r.FormValue("department"),
		Answers: map[string]string{},
	}
	if in.Department == "" {
		return bad("اختر الإدارة")
	}
	for key, values := range r.MultipartForm.Value {
		if id, ok := strings.CutPrefix(key, "extra-"); ok && len(values) == 1 {
			in.Answers[id] = values[0]
		}
	}
	var asset Asset
	if files := r.MultipartForm.File["file"]; len(files) > 0 {
		if len(files) != 1 {
			return bad("أرفق ملفاً واحداً")
		}
		var err error
		asset, err = s.saveTrainerMaterial(r)
		if err != nil {
			return err
		}
	}
	var id string
	err := s.write(r.Context(), func(d *Data) error {
		if !programOpen(d, "trainer") {
			return missing()
		}
		a, err := signedIn(d, r)
		var guest *GuestApplicant
		if err == nil {
			if !can(a, "trainer.apply") {
				return forbidden()
			}
		} else {
			guest = &GuestApplicant{
				Name: strings.TrimSpace(r.FormValue("name")), Email: strings.ToLower(strings.TrimSpace(r.FormValue("email"))),
				Employee: strings.ToUpper(strings.TrimSpace(r.FormValue("employee"))),
			}
			addr, emailErr := mail.ParseAddress(guest.Email)
			if emailErr != nil || addr.Address != guest.Email || !bounded(guest.Email, 254) || !bounded(guest.Name, 100) || !bounded(guest.Employee, 32) {
				return bad("أكمل الاسم والرقم الوظيفي والبريد الإلكتروني")
			}
			a = Account{ID: "guest-" + digest(guest.Employee), Profile: Profile{Name: guest.Name}}
		}
		if asset.ID != "" {
			asset.Owner = a.ID
			d.Assets[asset.ID] = asset
			in.Material = asset.ID
		}
		id, err = submitTrainerApplication(d, a, in, guest)
		return err
	})
	if err != nil {
		if asset.ID != "" {
			_ = os.Remove(filepath.Join(s.dataDir, asset.ID))
		}
		return err
	}
	respond(w, 201, map[string]string{"id": id})
	return nil
}
