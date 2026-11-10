package main

type surveyAnswer struct {
	Value int     `json:"value"`
	Count int     `json:"count"`
	Share float64 `json:"share"`
}

type surveyQuestionSummary struct {
	Answers []surveyAnswer `json:"answers"`
	Count   int            `json:"count"`
	Median  *float64       `json:"median"`
}

type npsSummary struct {
	Count      int      `json:"count"`
	Promoters  int      `json:"promoters"`
	Passives   int      `json:"passives"`
	Detractors int      `json:"detractors"`
	Score      *float64 `json:"score"`
}

type surveyAnalytics struct {
	Submitted int                     `json:"submitted"`
	NPS       *npsSummary             `json:"nps,omitempty"`
	Questions []surveyQuestionSummary `json:"questions"`
}

func surveyRange(q AssessmentQuestion) (int, int) {
	if q.Kind == "nps" {
		return 0, 10
	}
	if q.Scale == 10 {
		return 1, 10
	}
	return 1, 5
}

func hasRatingQuestion(a *Assessment) bool {
	for _, q := range a.Questions {
		if q.Kind != "nps" {
			return true
		}
	}
	return false
}

func summarizeSurvey(a *Assessment) surveyAnalytics {
	summary := surveyAnalytics{Submitted: len(a.Results), Questions: make([]surveyQuestionSummary, len(a.Questions))}
	for index, q := range a.Questions {
		minimum, maximum := surveyRange(q)
		answers := make([]surveyAnswer, maximum-minimum+1)
		for i := range answers {
			answers[i].Value = minimum + i
		}
		count := 0
		for _, result := range a.Results {
			if index >= len(result.Choices) {
				continue
			}
			value := result.Choices[index]
			if value >= minimum && value <= maximum {
				answers[value-minimum].Count++
				count++
			}
		}
		item := surveyQuestionSummary{Answers: answers, Count: count}
		cumulative, middleSum := 0, 0
		for i := range answers {
			answer := &answers[i]
			if count == 0 {
				continue
			}
			answer.Share = round(100 * float64(answer.Count) / float64(count))
			for _, middle := range [2]int{(count - 1) / 2, count / 2} {
				if cumulative <= middle && middle < cumulative+answer.Count {
					middleSum += answer.Value
				}
			}
			cumulative += answer.Count
		}
		if q.Kind == "nps" {
			nps := npsSummary{Count: count}
			for _, answer := range answers {
				switch {
				case answer.Value >= 9:
					nps.Promoters += answer.Count
				case answer.Value >= 7:
					nps.Passives += answer.Count
				default:
					nps.Detractors += answer.Count
				}
			}
			if count > 0 {
				score := round(100 * float64(nps.Promoters-nps.Detractors) / float64(count))
				nps.Score = &score
			}
			summary.NPS = &nps
		} else if count > 0 {
			median := float64(middleSum) / 2
			item.Median = &median
		}
		summary.Questions[index] = item
	}
	return summary
}
