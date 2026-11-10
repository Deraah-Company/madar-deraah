package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func accountByEmail(d *Data, email string) (Account, bool) {
	for _, a := range d.Accounts {
		if strings.EqualFold(a.Email, strings.TrimSpace(email)) {
			return a, true
		}
	}
	return Account{}, false
}
func goodPassword(password string) bool { return len(password) >= 12 && len(password) <= 256 }
func validPhoto(photo string) bool {
	if photo == "" {
		return true
	}
	if len(photo) > 700000 {
		return false
	}
	prefix, data, ok := strings.Cut(photo, ",")
	if !ok {
		return false
	}
	if prefix != "data:image/png;base64" && prefix != "data:image/jpeg;base64" {
		return false
	}
	raw, e := base64.StdEncoding.DecodeString(data)
	if e != nil {
		return false
	}
	info, kind, e := image.DecodeConfig(bytes.NewReader(raw))
	return e == nil && prefix == "data:image/"+kind+";base64" && info.Width > 0 && info.Width <= 1800 && info.Height > 0 && info.Height <= 1800
}
func setProfile(d *Data, a *Account, p Profile) error {
	p.Name = strings.TrimSpace(p.Name)
	p.Email = strings.ToLower(strings.TrimSpace(p.Email))
	p.Title = strings.TrimSpace(p.Title)
	p.Employee = strings.ToUpper(strings.TrimSpace(p.Employee))
	p.Bio = strings.TrimSpace(p.Bio)
	address, e := mail.ParseAddress(p.Email)
	if e != nil || address.Address != p.Email || !bounded(p.Name, 100) || !bounded(p.Title, 120) || !bounded(p.Employee, 32) || deptName(d, p.Department) == "" || len([]rune(p.Bio)) > 500 || !validPhoto(p.Photo) {
		return bad("أكمل الاسم والبريد والمسمى والرقم الوظيفي والإدارة. الصورة JPG أو PNG بحجم لا يتجاوز 500 كيلوبايت.")
	}
	for _, other := range d.Accounts {
		if other.ID != a.ID && (other.Email == p.Email || other.Employee == p.Employee) {
			return bad("البريد أو الرقم الوظيفي مستخدم بالفعل")
		}
	}
	a.Profile = p
	return nil
}
func shownAccount(d *Data, a Account) Account {
	a = clean(a)
	if a.Bio == "" {
		a.Bio = a.Title
		if name := deptName(d, a.Department); name != "" {
			a.Bio += " · " + name
		}
	}
	return a
}
func (s *server) cookie(w http.ResponseWriter, value string, age int) {
	http.SetCookie(w, &http.Cookie{Name: "deraah_pathways", Value: value, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: strings.HasPrefix(s.origin, "https://"), MaxAge: age})
}
func (s *server) login(w http.ResponseWriter, r *http.Request) error {
	if !s.limiter(r, "login", 16) {
		return fault{429, "محاولات كثيرة. حاول بعد دقيقة"}
	}
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if e := decode(r, &in); e != nil {
		return e
	}
	if len(in.Password) > 256 {
		return bad("تحقق من البريد وكلمة المرور")
	}
	d, e := s.read(r.Context())
	if e != nil {
		return e
	}
	a, exists := accountByEmail(&d, in.Email)
	key := a.Secret
	if !exists {
		key = "invalid:" + strings.Repeat("0", 64)
	}
	if !passwordMatches(in.Password, key) || !exists {
		return fault{401, "البريد أو كلمة المرور غير صحيحة"}
	}
	if a.Status != "active" {
		if a.Status == "pending" {
			return fault{403, "طلب التسجيل بانتظار الموافقة"}
		}
		return fault{403, "الحساب موقوف. تواصل مع مدير النظام"}
	}
	plain := nonce()
	e = s.write(r.Context(), func(d *Data) error {
		current, ok := d.Accounts[a.ID]
		if !ok || current.Status != "active" || current.Epoch != a.Epoch {
			return loginRequired()
		}
		for k, v := range d.Sessions {
			if time.Now().After(v.Expires) {
				delete(d.Sessions, k)
			}
		}
		d.Sessions[digest(plain)] = SessionKey{a.ID, a.Epoch, time.Now().Add(12 * time.Hour)}
		return nil
	})
	if e != nil {
		return e
	}
	s.cookie(w, plain, 43200)
	respond(w, 200, map[string]bool{"ok": true})
	return nil
}
func (s *server) register(w http.ResponseWriter, r *http.Request) error {
	if !s.limiter(r, "register", 8) {
		return fault{429, "حاول بعد دقيقة"}
	}
	var in struct {
		Profile
		Password string `json:"password"`
	}
	if e := decode(r, &in); e != nil {
		return e
	}
	if !goodPassword(in.Password) {
		return bad("كلمة المرور 12 حرفاً على الأقل")
	}
	e := s.write(r.Context(), func(d *Data) error {
		if !d.Registration {
			return forbidden()
		}
		a := Account{ID: nonce(), Status: "pending", Trainer: "employee", Voice: "employee", Secret: secret(in.Password)}
		if e := setProfile(d, &a, in.Profile); e != nil {
			return e
		}
		d.Accounts[a.ID] = a
		audit(d, a, "register", a.ID)
		return nil
	})
	if e != nil {
		return e
	}
	respond(w, 201, map[string]bool{"pending": true})
	return nil
}
func portal(d *Data, a Account) map[string]any {
	programs := programAvailability(d)
	out := map[string]any{"me": shownAccount(d, a), "permissions": permissions(a), "departments": d.Departments, "programs": programs}
	if a.System {
		users := []Account{}
		effective := map[string]map[string]bool{}
		for _, u := range d.Accounts {
			users = append(users, shownAccount(d, u))
			effective[u.ID] = permissions(u)
		}
		out["users"] = users
		out["userPermissions"] = effective
		out["registration"] = d.Registration
		out["demoInstalled"] = d.DemoInstalled
		out["permissionsCatalog"] = permissionLabels
		at := len(d.Audit) - 100
		if at < 0 {
			at = 0
		}
		out["audit"] = d.Audit[at:]
		out["overview"] = map[string]int{"campaigns": len(d.Campaigns), "applications": len(d.Applications), "sessions": len(d.Trainings), "surveys": len(d.Surveys), "bulletins": len(d.Bulletins)}
	}
	return out
}
func (s *server) action(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Action string          `json:"action"`
		Data   json.RawMessage `json:"data"`
	}
	if e := decode(r, &in); e != nil {
		return e
	}
	var removed []*FamilyFile
	e := s.write(r.Context(), func(d *Data) error {
		a, e := signedIn(d, r)
		if e != nil {
			return e
		}
		area, _, _ := strings.Cut(in.Action, ".")
		if includes(programKeys, area) && !programOpen(d, area) {
			return missing()
		}
		switch area {
		case "system", "profile":
			return accountAction(d, a, in.Action, in.Data)
		case "notice":
			if in.Action == "notice.read" {
				return readNotification(d, a, in.Data)
			}
		case "trainer":
			return trainerAction(d, a, in.Action, in.Data)
		case "voice":
			return voiceAction(d, a, in.Action, in.Data)
		case "family":
			removed, e = familyAction(d, a, in.Action, in.Data)
			return e
		}
		return missing()
	})
	if e != nil {
		return e
	}
	s.removeFamilyFiles(removed...)
	respond(w, 200, map[string]bool{"ok": true})
	return nil
}
func revoke(d *Data, id string) {
	for k, v := range d.Sessions {
		if v.Account == id {
			delete(d.Sessions, k)
		}
	}
}
func accountAction(d *Data, actor Account, action string, raw json.RawMessage) error {
	if action == "profile.update" {
		var p Profile
		if e := payload(raw, &p); e != nil {
			return e
		}
		p.Employee = actor.Employee
		p.Department = actor.Department
		if e := setProfile(d, &actor, p); e != nil {
			return e
		}
		d.Accounts[actor.ID] = actor
		return nil
	}
	if !actor.System {
		return forbidden()
	}
	switch action {
	case "system.demo":
		var in struct{ Password string }
		if e := payload(raw, &in); e != nil {
			return e
		}
		return installDemo(d, actor, in.Password)
	case "system.program":
		var in struct {
			Program string
			Open    bool
		}
		if e := payload(raw, &in); e != nil {
			return e
		}
		if !includes(programKeys, in.Program) {
			return bad("البرنامج غير صالح")
		}
		if d.ClosedPrograms == nil {
			d.ClosedPrograms = map[string]bool{}
		}
		d.ClosedPrograms[in.Program] = !in.Open
		audit(d, actor, action, in.Program)
		return nil
	case "system.registration":
		var in struct {
			Open bool `json:"open"`
		}
		if e := payload(raw, &in); e != nil {
			return e
		}
		d.Registration = in.Open
		audit(d, actor, action, "")
		return nil
	case "system.department":
		var in Department
		if e := payload(raw, &in); e != nil {
			return e
		}
		in.Name = strings.TrimSpace(in.Name)
		if !bounded(in.Name, 80) {
			return bad("اسم الإدارة مطلوب")
		}
		for _, v := range d.Departments {
			if v.Name == in.Name && v.ID != in.ID {
				return bad("الإدارة موجودة بالفعل")
			}
		}
		if in.ID == "" {
			in.ID = nonce()
			d.Departments = append(d.Departments, in)
		} else {
			found := false
			for i, v := range d.Departments {
				if v.ID == in.ID {
					d.Departments[i] = in
					found = true
				}
			}
			if !found {
				return missing()
			}
		}
		audit(d, actor, action, in.ID)
		return nil
	}
	var in struct {
		ID       string          `json:"id"`
		Profile  Profile         `json:"profile"`
		Status   string          `json:"status"`
		Password string          `json:"password"`
		Trainer  string          `json:"trainer"`
		Voice    string          `json:"voice"`
		Family   *string         `json:"family"`
		System   bool            `json:"system"`
		Grants   map[string]bool `json:"grants"`
	}
	if e := payload(raw, &in); e != nil {
		return e
	}
	target, ok := d.Accounts[in.ID]
	if !ok {
		return missing()
	}
	switch action {
	case "system.profile":
		if e := setProfile(d, &target, in.Profile); e != nil {
			return e
		}
	case "system.status":
		if target.ID == actor.ID {
			return bad("لا يمكن إيقاف حسابك الحالي")
		}
		if in.Status != "active" && in.Status != "suspended" {
			return bad("حالة الحساب غير صالحة")
		}
		target.Status = in.Status
		target.Epoch++
		revoke(d, target.ID)
	case "system.password":
		if !goodPassword(in.Password) {
			return bad("كلمة المرور 12 حرفاً على الأقل")
		}
		target.Secret = secret(in.Password)
		target.Epoch++
		revoke(d, target.ID)
	case "system.permissions":
		if target.ID == actor.ID && !in.System {
			return bad("لا يمكن إزالة صلاحية إدارة النظام من حسابك الحالي")
		}
		for k := range in.Grants {
			if _, ok := permissionLabels[k]; !ok {
				return bad("صلاحية غير معروفة")
			}
		}
		target.Trainer, target.Voice, target.Family = "none", "none", "none"
		target.System, target.Grants = in.System, in.Grants
	case "system.roles":
		if target.ID == actor.ID && !in.System {
			return bad("لا يمكن إزالة صلاحية إدارة النظام من حسابك الحالي")
		}
		if _, ok := trainerRoles[in.Trainer]; !ok {
			return bad("دور سفراء المعرفة غير صالح")
		}
		if _, ok := voiceRoles[in.Voice]; !ok {
			return bad("دور نبض درعه غير صالح")
		}
		for k := range in.Grants {
			if _, ok := permissionLabels[k]; !ok {
				return bad("صلاحية غير معروفة")
			}
		}
		target.Trainer = in.Trainer
		target.Voice = in.Voice
		if in.Family != nil {
			if _, ok := familyRoles[*in.Family]; !ok {
				return bad("دور Deraah Family غير صالح")
			}
			target.Family = *in.Family
		}
		target.System = in.System
		target.Grants = in.Grants
	case "system.delete":
		if target.ID == actor.ID {
			return bad("لا يمكن حذف حسابك الحالي")
		}
		for _, c := range d.Campaigns {
			if c.Status != "closed" && includes(c.Panel, target.ID) {
				return bad("أزل العضو من اللجان القائمة أو أغلق طلب التقديم أولاً")
			}
		}
		revoke(d, target.ID)
		delete(d.Accounts, target.ID)
		for id, asset := range d.Assets {
			if asset.Owner == target.ID {
				delete(d.Assets, id)
			}
		}
		for i := range d.Applications {
			if d.Applications[i].User == target.ID {
				d.Applications[i].Outline = ""
				d.Applications[i].Material = ""
				d.Applications[i].Answers = nil
			}
		}
		audit(d, actor, action, in.ID)
		return nil
	default:
		return missing()
	}
	d.Accounts[target.ID] = target
	audit(d, actor, action, in.ID)
	return nil
}

func (s *server) upload(w http.ResponseWriter, r *http.Request) error {
	d, e := s.read(r.Context())
	if e != nil {
		return e
	}
	if !programOpen(&d, "trainer") {
		return missing()
	}
	a, e := signedIn(&d, r)
	if e != nil {
		return e
	}
	if !can(a, "trainer.apply") {
		return forbidden()
	}
	r.Body = http.MaxBytesReader(w, r.Body, 11<<20)
	if e = r.ParseMultipartForm(1 << 20); e != nil {
		return bad("حجم الملف لا يتجاوز 10 ميجابايت")
	}
	defer r.MultipartForm.RemoveAll()
	asset, e := s.saveTrainerMaterial(r)
	if e != nil {
		return e
	}
	path := filepath.Join(s.dataDir, asset.ID)
	e = s.write(r.Context(), func(d *Data) error {
		if !programOpen(d, "trainer") {
			return missing()
		}
		current, e := signedIn(d, r)
		if e != nil {
			return e
		}
		if !can(current, "trainer.apply") {
			return forbidden()
		}
		asset.Owner = current.ID
		d.Assets[asset.ID] = asset
		return nil
	})
	if e != nil {
		_ = os.Remove(path)
		return e
	}
	respond(w, 201, map[string]string{"id": asset.ID, "name": asset.Name})
	return nil
}
func (s *server) saveTrainerMaterial(r *http.Request) (Asset, error) {
	f, h, e := r.FormFile("file")
	if e != nil {
		return Asset{}, bad("اختر ملفاً")
	}
	defer f.Close()
	data, e := io.ReadAll(io.LimitReader(f, (10<<20)+1))
	if e != nil || len(data) == 0 || len(data) > 10<<20 {
		return Asset{}, bad("حجم الملف لا يتجاوز 10 ميجابايت")
	}
	ext := strings.ToLower(filepath.Ext(h.Filename))
	formats := map[string]string{".pdf": "application/pdf", ".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation", ".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document"}
	if formats[ext] == "" || ext == ".pdf" && !bytes.HasPrefix(data, []byte("%PDF-")) || ext != ".pdf" && !bytes.HasPrefix(data, []byte("PK\x03\x04")) {
		return Asset{}, bad("ارفع ملف PDF أو PPTX أو DOCX")
	}
	asset := Asset{nonce(), "", "material", filepath.Base(h.Filename), formats[ext]}
	if e = os.MkdirAll(s.dataDir, 0700); e != nil {
		return Asset{}, e
	}
	path := filepath.Join(s.dataDir, asset.ID)
	if e = os.WriteFile(path, data, 0600); e != nil {
		_ = os.Remove(path)
		return Asset{}, e
	}
	return asset, nil
}

func (s *server) serveAsset(w http.ResponseWriter, r *http.Request, d *Data, a Account) error {
	key := strings.TrimPrefix(r.URL.Path, "/files/")
	file, ok := d.Assets[key]
	if !ok || file.Kind != "material" {
		return missing()
	}
	allowed := a.System || file.Owner == a.ID && can(a, "trainer.apply")
	for _, app := range d.Applications {
		if app.Material != key {
			continue
		}
		c := campaignFor(d, app.Campaign)
		if can(a, "trainer.decide") || can(a, "trainer.evaluate") && c != nil && includes(c.Panel, a.ID) {
			allowed = true
		}
	}
	if !allowed {
		return forbidden()
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": file.Name}))
	w.Header().Set("Content-Type", file.MIME)
	w.Header().Set("Cache-Control", "private, no-store")
	http.ServeFile(w, r, filepath.Join(s.dataDir, key))
	return nil
}
