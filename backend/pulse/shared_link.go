package pulse

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

func ensureSharedLink(ctx context.Context, tx pgx.Tx, id, token string) (string, string, error) {
	var segment string
	if token != "" {
		err := tx.QueryRow(ctx, "SELECT segment_id FROM invites WHERE token_hash=$1 AND survey_id=$2", digest(token), id).Scan(&segment)
		return token, segment, err
	}
	token, segment = randomToken(32), newID()
	if _, err := tx.Exec(ctx, "INSERT INTO segments(id,survey_id,label,site,track) VALUES($1,$2,$3,'','all')", segment, id, marshal(LocalText{"جميع الموظفين", "All employees"})); err != nil {
		return "", "", err
	}
	if _, err := tx.Exec(ctx, "INSERT INTO invites(id,survey_id,segment_id,token_hash,token_prefix,mode) VALUES($1,$2,$3,$4,$5,'shared')", newID(), id, segment, digest(token), token[:6]); err != nil {
		return "", "", err
	}
	_, err := tx.Exec(ctx, "UPDATE surveys SET public_token=$1 WHERE id=$2", token, id)
	return token, segment, err
}

func (s *Server) sharedSurveyLink(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		fail(w, 500, "database_error")
		return
	}
	defer tx.Rollback(ctx)
	id := r.PathValue("id")
	var token, status string
	err = tx.QueryRow(ctx, "SELECT public_token,status FROM surveys WHERE id=$1 FOR UPDATE", id).Scan(&token, &status)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && status != "published" {
		fail(w, 409, "survey_unavailable")
		return
	}
	if err != nil {
		fail(w, 500, "database_error")
		return
	}
	token, _, err = ensureSharedLink(ctx, tx, id, token)
	if err != nil || tx.Commit(ctx) != nil {
		fail(w, 500, "database_error")
		return
	}
	writeJSON(w, 200, map[string]string{"path": "/pulse/r/" + token})
}
