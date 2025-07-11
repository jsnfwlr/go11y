CREATE TABLE IF NOT EXISTS remote_api_requests (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    url TEXT NOT NULL,
    method TEXT NOT NULL,
    request_headers JSONB NOT NULL,
    request_body TEXT,
    response_time_ms BIGINT NOT NULL,
    response_headers JSONB NOT NULL,
    response_body TEXT,
    status_code INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);
