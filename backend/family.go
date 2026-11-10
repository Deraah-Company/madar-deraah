package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type FamilyFile struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type Bulletin struct {
	ID       string          `json:"id"`
	Title    string          `json:"title"`
	Summary  string          `json:"summary"`
	At       string          `json:"at"`
	Hidden   bool            `json:"hidden"`
	PDF      *FamilyFile     `json:"pdf"`
	Video    *FamilyFile     `json:"video"`
	Comments []FamilyComment `json:"comments"`
}
type FamilyComment struct {
	ID     string
	Author string
	Text   string
	At     string
}

func familyIndex(d *Data, id string) int {
	return slices.IndexFunc(d.Bulletins, func(b Bulletin) bool { return b.ID == id })
}

func familyView(d *Data, a Account) map[string]any {
	rows := []map[string]any{}
	current := ""
	for i := len(d.Bulletins) - 1; i >= 0; i-- {
		b := d.Bulletins[i]
		if b.Hidden && !can(a, "family.manage") {
			continue
		}
		if !b.Hidden && (current == "" || b.ID == d.FamilyFeatured) {
			current = b.ID
		}
		comments := []map[string]string{}
		for _, c := range b.Comments {
			name := "موظف سابق"
			if account, ok := d.Accounts[c.Author]; ok {
				name = account.Name
			}
			comments = append(comments, map[string]string{"id": c.ID, "name": name, "text": c.Text, "at": c.At})
		}
		rows = append(rows, map[string]any{"id": b.ID, "title": b.Title, "summary": b.Summary, "at": b.At, "hidden": b.Hidden, "pdf": b.PDF, "video": b.Video, "comments": comments})
	}
	return map[string]any{"intro": d.FamilyIntro, "current": current, "bulletins": rows}
}

func familyAction(d *Data, a Account, action string, raw json.RawMessage) ([]*FamilyFile, error) {
	var input struct {
		ID, Comment, Text string
		Hidden            bool
	}
	if err := payload(raw, &input); err != nil {
		return nil, err
	}
	permission := map[string]string{"family.feature": "family.manage", "family.hide": "family.manage", "family.delete": "family.manage", "family.comment": "family.comment", "family.removeComment": "family.moderate"}[action]
	if permission == "" {
		return nil, missing()
	}
	if !can(a, permission) {
		return nil, forbidden()
	}
	i := familyIndex(d, input.ID)
	if i < 0 {
		return nil, missing()
	}
	b := &d.Bulletins[i]
	var removed []*FamilyFile
	switch action {
	case "family.feature":
		if b.Hidden {
			return nil, bad("أظهر النشرة قبل اختيارها كنشرة حالية")
		}
		d.FamilyFeatured = b.ID
	case "family.hide":
		b.Hidden = input.Hidden
		if b.Hidden && d.FamilyFeatured == b.ID {
			d.FamilyFeatured = ""
		}
	case "family.delete":
		removed = []*FamilyFile{b.PDF, b.Video}
		if d.FamilyFeatured == b.ID {
			d.FamilyFeatured = ""
		}
		d.Bulletins = slices.Delete(d.Bulletins, i, i+1)
	case "family.comment":
		if b.Hidden {
			return nil, missing()
		}
		input.Text = strings.TrimSpace(input.Text)
		if !bounded(input.Text, 2000) {
			return nil, bad("اكتب تعليقاً لا يتجاوز 2000 حرف")
		}
		b.Comments = append(b.Comments, FamilyComment{nonce(), a.ID, input.Text, timestamp()})
	case "family.removeComment":
		i := slices.IndexFunc(b.Comments, func(c FamilyComment) bool { return c.ID == input.Comment })
		if i < 0 {
			return nil, missing()
		}
		b.Comments = slices.Delete(b.Comments, i, i+1)
	}
	if action != "family.comment" {
		audit(d, a, action, input.ID)
	}
	return removed, nil
}

func (s *server) removeFamilyFiles(files ...*FamilyFile) {
	for _, f := range files {
		if f != nil {
			_ = os.Remove(filepath.Join(s.dataDir, f.ID))
		}
	}
}

func (s *server) saveFamily(w http.ResponseWriter, r *http.Request) error {
	d, err := s.read(r.Context())
	if err != nil {
		return err
	}
	if !programOpen(&d, "family") {
		return missing()
	}
	a, err := signedIn(&d, r)
	if err != nil {
		return err
	}
	if !can(a, "family.manage") {
		return forbidden()
	}
	r.Body = http.MaxBytesReader(w, r.Body, 72<<20)
	err = r.ParseMultipartForm(1 << 20)
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if err != nil {
		return bad("راجع الملفات. الحد الأقصى 20 ميجابايت للـ PDF و50 ميجابايت للفيديو")
	}

	intro := r.FormValue("mode") == "intro"
	id, title, summary := r.FormValue("id"), strings.TrimSpace(r.FormValue("title")), strings.TrimSpace(r.FormValue("summary"))
	if !intro && (!bounded(title, 160) || len([]rune(summary)) > 4000) {
		return bad("أدخل عنوان النشرة ووصفاً لا يتجاوز 4000 حرف")
	}
	var pdf, video *FamilyFile
	saved := false
	defer func() {
		if !saved {
			s.removeFamilyFiles(pdf, video)
		}
	}()
	if !intro {
		pdf, err = s.readFamilyUpload(r, "pdf", ".pdf", 20<<20)
		if err != nil {
			return err
		}
	}
	video, err = s.readFamilyUpload(r, "video", ".mp4", 50<<20)
	if err != nil {
		return err
	}
	var replaced []*FamilyFile
	err = s.write(r.Context(), func(d *Data) error {
		if !programOpen(d, "family") {
			return missing()
		}
		a, err := signedIn(d, r)
		if err != nil {
			return err
		}
		if !can(a, "family.manage") {
			return forbidden()
		}
		if intro {
			if video == nil && r.FormValue("removeVideo") != "on" {
				return bad("اختر فيديو MP4")
			}
			replaced = append(replaced, d.FamilyIntro)
			d.FamilyIntro = video
			audit(d, a, "family.intro", "")
			return nil
		}
		b := Bulletin{ID: nonce(), At: timestamp()}
		i := -1
		if id != "" {
			i = familyIndex(d, id)
			if i < 0 {
				return missing()
			}
			b = d.Bulletins[i]
		}
		if pdf != nil || r.FormValue("removePDF") == "on" {
			replaced = append(replaced, b.PDF)
			b.PDF = pdf
		}
		if video != nil || r.FormValue("removeVideo") == "on" {
			replaced = append(replaced, b.Video)
			b.Video = video
		}
		if b.PDF == nil && b.Video == nil {
			return bad("أضف نشرة PDF أو فيديو MP4")
		}
		b.Title, b.Summary = title, summary
		if i < 0 {
			d.Bulletins = append(d.Bulletins, b)
			announceBulletin(d, b)
		} else {
			d.Bulletins[i] = b
		}
		audit(d, a, "family.save", b.ID)
		return nil
	})
	if err != nil {
		return err
	}
	saved = true
	s.removeFamilyFiles(replaced...)
	respond(w, 200, map[string]bool{"ok": true})
	return nil
}

func (s *server) readFamilyUpload(r *http.Request, field, ext string, limit int64) (*FamilyFile, error) {
	headers := r.MultipartForm.File[field]
	if len(headers) == 0 {
		return nil, nil
	}
	h := headers[0]
	if len(headers) != 1 || h.Size < 12 || h.Size > limit || strings.ToLower(filepath.Ext(h.Filename)) != ext {
		return nil, bad("نوع الملف أو حجمه غير صالح")
	}
	source, err := h.Open()
	if err != nil {
		return nil, err
	}
	defer source.Close()
	var head [12]byte
	if _, err = io.ReadFull(source, head[:]); err != nil {
		return nil, bad("تعذر قراءة الملف")
	}
	if ext == ".pdf" && !bytes.HasPrefix(head[:], []byte("%PDF-")) || ext == ".mp4" && string(head[4:8]) != "ftyp" {
		return nil, bad("محتوى الملف لا يطابق نوعه")
	}
	if err = os.MkdirAll(s.dataDir, 0700); err != nil {
		return nil, err
	}
	f := &FamilyFile{ID: nonce(), Name: filepath.Base(h.Filename)}
	target, err := os.OpenFile(filepath.Join(s.dataDir, f.ID), os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(target, io.MultiReader(bytes.NewReader(head[:]), source))
	closeErr := target.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		s.removeFamilyFiles(f)
		return nil, err
	}
	return f, nil
}

func (s *server) serveFamilyFile(w http.ResponseWriter, r *http.Request, d *Data, a Account) error {
	if !can(a, "family.view") {
		return forbidden()
	}
	id := strings.TrimPrefix(r.URL.Path, "/files/family/")
	if f := d.FamilyIntro; f != nil && f.ID == id {
		return s.serveFamilyMedia(w, r, f, "video/mp4")
	}
	for _, b := range d.Bulletins {
		if b.Hidden && !can(a, "family.manage") {
			continue
		}
		if f := b.PDF; f != nil && f.ID == id {
			return s.serveFamilyMedia(w, r, f, "application/pdf")
		}
		if f := b.Video; f != nil && f.ID == id {
			return s.serveFamilyMedia(w, r, f, "video/mp4")
		}
	}
	return missing()
}

func (s *server) serveFamilyMedia(w http.ResponseWriter, r *http.Request, f *FamilyFile, mimeType string) error {
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": f.Name}))
	w.Header().Set("Cache-Control", "private, no-store")
	if mimeType == "application/pdf" {
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'self'; object-src 'none'")
	}
	var bundled string
	switch f.ID {
	case demoPDF:
		bundled = "demo-bulletin.pdf"
	case demoVideo:
		bundled = "web/media/05_Madar_Deraah_Welcome_Promo_1080p.mp4"
	default:
		http.ServeFile(w, r, filepath.Join(s.dataDir, f.ID))
		return nil
	}
	file, err := content.Open(bundled)
	if err != nil {
		return err
	}
	defer file.Close()
	http.ServeContent(w, r, f.Name, time.Time{}, file.(io.ReadSeeker))
	return nil
}
