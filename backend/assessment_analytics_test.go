package main

import "testing"

func TestSurveyAnalyticsSeparatesRatingScalesFromNPS(t *testing.T) {
	a := Assessment{
		Questions: []AssessmentQuestion{{Kind: "rating"}, {Kind: "rating", Scale: 10}, {Kind: "nps"}},
		Results:   map[string]AssessmentResult{},
	}
	for i, value := range []int{0, 6, 7, 8, 9, 9, 10, 10} {
		a.Results[string(rune('a'+i))] = AssessmentResult{Choices: []int{1 + i%5, 3 + i, value}}
	}
	got := summarizeSurvey(&a)
	if got.Submitted != 8 || got.NPS.Count != 8 || got.NPS.Detractors != 2 || got.NPS.Passives != 2 || got.NPS.Promoters != 4 || got.NPS.Score == nil || *got.NPS.Score != 25 {
		t.Fatalf("unexpected recommendation results: %+v", got.NPS)
	}
	five, ten := got.Questions[0], got.Questions[1]
	if len(five.Answers) != 5 || five.Median == nil || *five.Median != 2.5 || five.Answers[0].Share != 25 {
		t.Fatalf("unexpected results for legacy five-point scale: %+v", five)
	}
	if len(ten.Answers) != 10 || ten.Median == nil || *ten.Median != 6.5 || ten.Answers[9].Count != 1 {
		t.Fatalf("unexpected results for ten-point scale: %+v", ten)
	}
}

func TestSurveyAnalyticsWithNoRecommendationAnswers(t *testing.T) {
	a := Assessment{
		Questions: []AssessmentQuestion{{Kind: "rating", Scale: 10}},
		Results:   map[string]AssessmentResult{"legacy": {Choices: []int{10}}},
	}
	if got := summarizeSurvey(&a); got.NPS != nil {
		t.Fatalf("ordinary ratings must not produce NPS: %+v", got.NPS)
	}
	a.Questions = append(a.Questions, AssessmentQuestion{Kind: "nps"})
	if got := summarizeSurvey(&a); got.NPS == nil || got.NPS.Score != nil || got.NPS.Count != 0 {
		t.Fatalf("missing recommendations must not be counted as zero: %+v", got.NPS)
	}
}
