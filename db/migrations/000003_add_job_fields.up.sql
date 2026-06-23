ALTER TABLE events ADD COLUMN job_type TEXT NOT NULL DEFAULT '';
ALTER TABLE events ADD COLUMN callback_url TEXT NOT NULL DEFAULT '';

CREATE TABLE event_executions (
    id           SERIAL PRIMARY KEY,
    event_id     INTEGER NOT NULL REFERENCES events(id),
    attempted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    status_code  INTEGER,
    error        TEXT,
    duration_ms  INTEGER
);
