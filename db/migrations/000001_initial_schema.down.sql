DO $$
    BEGIN
        IF NOT EXISTS (
            SELECT 1 FROM pg_type WHERE typname = 'status'
        ) THEN
            DROP TYPE status;
        END IF;
    END;
$$;

DROP TABLE IF EXISTS events;