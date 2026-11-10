package pulse

import (
	"encoding/json"
	"testing"
)

func pint(v int) *int { return &v }
func TestENPSUsesAllRespondents(t *testing.T) {
	rows := []Response{{Enps: 10}, {Enps: 9}, {Enps: 8}, {Enps: 7}, {Enps: 0}}
	m := aggregate(rows)
	if m.ENPS == nil || *m.ENPS != 20 || *m.Count != 5 || *m.Passives != 2 {
		t.Fatalf("incorrect distribution: %+v", m)
	}
	if got := aggregate(rows[:1]); got.ENPS == nil || *got.ENPS != 100 {
		t.Fatalf("first anonymous response should be counted: %+v", got)
	}
}
func TestDefaultBankAndValidation(t *testing.T) {
	b, e := bundled.ReadFile("default-survey.json")
	if e != nil {
		t.Fatal(e)
	}
	var d Definition
	if json.Unmarshal(b, &d) != nil || d.validate() != nil {
		t.Fatal("invalid default bank")
	}
	n := 0
	answers := map[string]*int{}
	for _, c := range d.Categories {
		n += len(c.Questions)
		for _, q := range c.Questions {
			answers[q.ID] = pint(4)
		}
	}
	if len(d.Categories) != 10 || n != 26 {
		t.Fatalf("expected 10 drivers / 26 items, got %d/%d", len(d.Categories), n)
	}
	if validateAnswers(d, "commercial", answers) != nil {
		t.Fatal("valid response rejected")
	}
	answers["unknown"] = pint(2)
	if validateAnswers(d, "commercial", answers) == nil {
		t.Fatal("unknown question accepted")
	}
	delete(answers, "unknown")
	delete(answers, d.Categories[0].Questions[0].ID)
	if validateAnswers(d, "commercial", answers) == nil {
		t.Fatal("required answer skipped")
	}
}
func TestOptionalTrackAndBounds(t *testing.T) {
	d := Definition{Categories: []Category{{ID: "cat", Name: LocalText{"عام", "Core"}, Track: "all", Questions: []Question{{ID: "q1", Text: LocalText{"سؤال", "Question"}, Required: false}}}, {ID: "extra", Name: LocalText{"إضافي", "Extra"}, Track: "commercial", Questions: []Question{{ID: "q2", Text: LocalText{"آخر", "Another"}, Required: true}}}}}
	if validateAnswers(d, "administrative", map[string]*int{"q1": nil}) != nil {
		t.Fatal("optional null rejected")
	}
	if validateAnswers(d, "administrative", map[string]*int{"q2": pint(4)}) == nil {
		t.Fatal("cross-track answer accepted")
	}
	if validateAnswers(d, "commercial", map[string]*int{"q1": pint(6), "q2": pint(4)}) == nil {
		t.Fatal("out-of-range accepted")
	}
}
