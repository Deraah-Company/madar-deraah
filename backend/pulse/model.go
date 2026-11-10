package pulse

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

type LocalText struct {
	Ar string `json:"ar"`
	En string `json:"en"`
}
type Question struct {
	ID       string    `json:"id"`
	Text     LocalText `json:"text"`
	Required bool      `json:"required"`
}
type Category struct {
	ID        string     `json:"id"`
	Name      LocalText  `json:"name"`
	Track     string     `json:"track"`
	Questions []Question `json:"questions"`
}
type Definition struct {
	Categories []Category `json:"categories"`
}
type User struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Role   string `json:"role"`
	Active bool   `json:"active"`
}
type Survey struct {
	ID              string     `json:"id"`
	Title           LocalText  `json:"title"`
	Description     LocalText  `json:"description"`
	Definition      Definition `json:"definition"`
	Status          string     `json:"status"`
	Expected        int        `json:"expected_count"`
	Excellent       int        `json:"excellent"`
	Good            int        `json:"good"`
	OpensAt         *string    `json:"opens_at"`
	ClosesAt        *string    `json:"closes_at"`
	Responses       int        `json:"response_count"`
	CreatedAt       string     `json:"created_at"`
	EmployeeVisible bool       `json:"employee_visible"`
}

func randomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func newID() string          { return randomToken(18) }
func digest(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }

func marshal(v any) []byte {
	b, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	return b
}
func (t LocalText) valid(max int) bool {
	return strings.TrimSpace(t.Ar) != "" && strings.TrimSpace(t.En) != "" && utf8.RuneCountInString(t.Ar) <= max && utf8.RuneCountInString(t.En) <= max && !strings.Contains(t.Ar, "???")
}
func (d Definition) validate() error {
	if len(d.Categories) < 1 || len(d.Categories) > 20 {
		return errors.New("categories_count")
	}
	seen := map[string]bool{}
	core := 0
	total := 0
	extras := map[string]int{}
	for _, c := range d.Categories {
		if c.ID == "" || len(c.ID) > 64 || seen[c.ID] || !c.Name.valid(160) || len(c.Questions) < 1 || len(c.Questions) > 12 {
			return errors.New("invalid_category")
		}
		seen[c.ID] = true
		if c.Track != "all" && c.Track != "commercial" && c.Track != "administrative" {
			return errors.New("invalid_track")
		}
		if c.Track == "all" {
			core++
		} else {
			extras[c.Track] += len(c.Questions)
		}
		for _, q := range c.Questions {
			total++
			if q.ID == "" || len(q.ID) > 64 || seen[q.ID] || !q.Text.valid(800) {
				return errors.New("invalid_question")
			}
			seen[q.ID] = true
		}
	}
	if core == 0 || total > 100 || extras["commercial"] > 4 || extras["administrative"] > 4 {
		return errors.New("question_limits")
	}
	return nil
}
func validateAnswers(d Definition, track string, answers map[string]*int) error {
	allowed := map[string]Question{}
	for _, c := range d.Categories {
		if track == "all" || c.Track == "all" || c.Track == track {
			for _, q := range c.Questions {
				allowed[q.ID] = q
			}
		}
	}
	for id, v := range answers {
		if _, ok := allowed[id]; !ok {
			return fmt.Errorf("unknown_question")
		}
		if v != nil && (*v < 1 || *v > 5) {
			return errors.New("invalid_answer")
		}
	}
	for id, q := range allowed {
		v, ok := answers[id]
		if q.Required && (!ok || v == nil) {
			return errors.New("required_answer")
		}
	}
	return nil
}
