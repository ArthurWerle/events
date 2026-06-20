DO $$
    BEGIN
        IF NOT EXISTS (
            SELECT 1 FROM pg_type WHERE typname = 'status'
        ) THEN
            CREATE TYPE status AS ENUM ('pending', 'processing', 'done', 'failed');
        END IF;
    END;
$$;

CREATE TABLE IF NOT EXISTS events (
    id SERIAL PRIMARY KEY,
    payload TEXT,
    status status,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);