package pulse

import (
	"encoding/json"
	"net/http"
)

const demoSurveyID = "pulse-demo-v1"

func (s *Server) demoSurvey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		fail(w, 500, "database_error")
		return
	}
	defer tx.Rollback(ctx)
	raw, err := bundled.ReadFile("default-survey.json")
	var definition Definition
	if err != nil || json.Unmarshal(raw, &definition) != nil {
		fail(w, 500, "invalid_survey")
		return
	}
	title := LocalText{"تجريبي · تجربة العمل في درعه", "Demo · Working at Deraah"}
	description := LocalText{"بيانات محاكاة لعرض شكل النتائج والمؤشر بعد انتهاء الاستبيان.", "Simulated answers demonstrating the results and recommendation score."}
	tag, err := tx.Exec(ctx, "INSERT INTO surveys(id,title,description,definition,status,expected_count,created_by) VALUES($1,$2,$3,$4,'closed',10,$5) ON CONFLICT(id) DO NOTHING", demoSurveyID, marshal(title), marshal(description), raw, actor(r).ID)
	if err != nil {
		fail(w, 500, "database_error")
		return
	}
	if tag.RowsAffected() == 1 {
		_, segment, err := ensureSharedLink(ctx, tx, demoSurveyID, "")
		if err != nil {
			fail(w, 500, "database_error")
			return
		}
		comments := []string{"اقتراح تجريبي بإتاحة جلسة شهرية لتبادل الخبرات.", "اقتراح تجريبي بتحسين وضوح تحديثات العمل."}
		for i, score := range []int{9, 10, 10, 9, 10, 7, 8, 6, 5, 3} {
			answers := map[string]*int{}
			for j, category := range definition.Categories {
				for k, question := range category.Questions {
					rating := 2 + (i+j+k)%4
					answers[question.ID] = &rating
				}
			}
			var comment string
			if i < len(comments) {
				comment = comments[i]
			}
			if err := storeResponse(ctx, tx, demoSurveyID, segment, score, answers, comment); err != nil {
				fail(w, 500, "database_error")
				return
			}
		}
	}
	if tx.Commit(ctx) != nil {
		fail(w, 500, "database_error")
		return
	}
	writeJSON(w, 200, map[string]string{"id": demoSurveyID})
}
