-- name: StoreAPIRequest :exec
INSERT INTO remote_api_requests (
    url,
    method,
    request_headers,
    request_body,
    response_time_ms,
    response_headers,
    response_body,
    status_code
) VALUES (
    @url,
    @method,
    @request_headers,
    @request_body,
    @response_time_ms,
    @response_headers,
    @response_body,
    @status_code
);

-- name: GetAPIRequests :one
SELECT *
FROM remote_api_requests
ORDER BY created_at DESC
LIMIT 1;