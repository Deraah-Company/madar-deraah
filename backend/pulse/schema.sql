CREATE TABLE IF NOT EXISTS users (
 id text PRIMARY KEY, email text NOT NULL UNIQUE, name text NOT NULL,
 password_hash text NOT NULL, role text NOT NULL CHECK(role IN ('admin','analyst','viewer','case_manager')),
 active boolean NOT NULL DEFAULT true, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS surveys (
 id text PRIMARY KEY, title jsonb NOT NULL, description jsonb NOT NULL,
 definition jsonb NOT NULL, status text NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','published','closed')),
 privacy_min integer NOT NULL DEFAULT 5 CHECK(privacy_min BETWEEN 5 AND 50),
 expected_count integer NOT NULL DEFAULT 0 CHECK(expected_count >= 0),
 excellent integer NOT NULL DEFAULT 50, good integer NOT NULL DEFAULT 10,
 opens_at timestamptz, closes_at timestamptz,
 created_by text NOT NULL REFERENCES users(id), created_at timestamptz NOT NULL DEFAULT now(),
 CHECK(excellent > good AND excellent <= 100 AND good >= -100),
 CHECK(opens_at IS NULL OR closes_at IS NULL OR closes_at > opens_at)
);
CREATE TABLE IF NOT EXISTS segments (
 id text PRIMARY KEY, survey_id text NOT NULL REFERENCES surveys(id), label jsonb NOT NULL,
 site text NOT NULL, track text NOT NULL CHECK(track IN ('commercial','administrative')),
 UNIQUE(id,survey_id)
);
CREATE TABLE IF NOT EXISTS invites (
 id text PRIMARY KEY, survey_id text NOT NULL REFERENCES surveys(id), segment_id text NOT NULL,
 token_hash text NOT NULL UNIQUE, token_prefix text NOT NULL, mode text NOT NULL CHECK(mode IN ('shared','single')),
 active boolean NOT NULL DEFAULT true, used boolean NOT NULL DEFAULT false,
 created_at timestamptz NOT NULL DEFAULT now(), FOREIGN KEY(segment_id,survey_id) REFERENCES segments(id,survey_id)
);
CREATE TABLE IF NOT EXISTS response_claims (
 survey_id text NOT NULL REFERENCES surveys(id), claim_hash text NOT NULL,
 PRIMARY KEY(survey_id,claim_hash)
);
CREATE TABLE IF NOT EXISTS responses (
 id text PRIMARY KEY, survey_id text NOT NULL REFERENCES surveys(id), segment_id text NOT NULL,
 enps integer NOT NULL CHECK(enps BETWEEN 0 AND 10), answers jsonb NOT NULL,
 tenure text NOT NULL DEFAULT '', submitted_on date NOT NULL DEFAULT current_date,
 FOREIGN KEY(segment_id,survey_id) REFERENCES segments(id,survey_id)
);
CREATE INDEX IF NOT EXISTS response_survey ON responses(survey_id,segment_id);
CREATE TABLE IF NOT EXISTS comments (
 id text PRIMARY KEY, response_id text NOT NULL REFERENCES responses(id) ON DELETE CASCADE,
 body text NOT NULL, status text NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','approved','rejected')),
 reviewed_by text REFERENCES users(id)
);
CREATE TABLE IF NOT EXISTS action_plans (
 id text PRIMARY KEY, survey_id text NOT NULL REFERENCES surveys(id), domain text NOT NULL,
 title text NOT NULL, owner_id text NOT NULL REFERENCES users(id), due_date date NOT NULL,
 status text NOT NULL CHECK(status IN ('planned','in_progress','done')), notes text NOT NULL DEFAULT '',
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS audit_log (
 id bigserial PRIMARY KEY, actor_id text REFERENCES users(id), action text NOT NULL,
 entity_type text NOT NULL, entity_id text NOT NULL DEFAULT '', created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS audit_created ON audit_log(created_at DESC);

CREATE TABLE IF NOT EXISTS survey_reports (survey_id text PRIMARY KEY REFERENCES surveys(id),report jsonb NOT NULL,created_at timestamptz NOT NULL DEFAULT now());
ALTER TABLE surveys ADD COLUMN IF NOT EXISTS employee_visible boolean NOT NULL DEFAULT false;
ALTER TABLE surveys ADD COLUMN IF NOT EXISTS public_token text NOT NULL DEFAULT '';
ALTER TABLE segments DROP CONSTRAINT IF EXISTS segments_track_check;
ALTER TABLE segments ADD CONSTRAINT segments_track_check CHECK(track IN ('all','commercial','administrative'));
ALTER TABLE survey_reports ADD COLUMN IF NOT EXISTS aggregate_only boolean NOT NULL DEFAULT false;
DELETE FROM survey_reports sr
 WHERE NOT sr.aggregate_only
 AND (sr.report->>'response_count')::integer =
     (SELECT count(*) FROM responses r WHERE r.survey_id=sr.survey_id);
