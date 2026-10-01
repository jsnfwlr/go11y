// Package storer provides functionality to store API request and response details in a PostgreSQL database for use by
// go11y's AddDBStorer transport middleware.
package storer

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StoreRequest struct for storing API request and response details
type StoreRequest struct {
	pool            *pgxpool.Pool
	URL             string      `db:"url" json:"url"`
	Method          string      `db:"method" json:"method"`
	RequestHeaders  []byte      `db:"request_headers" json:"request_headers"`
	RequestBody     pgtype.Text `db:"request_body" json:"request_body"`
	ResponseTimeMs  int64       `db:"response_time_ms" json:"response_time_ms"`
	ResponseHeaders []byte      `db:"response_headers" json:"response_headers"`
	ResponseBody    pgtype.Text `db:"response_body" json:"response_body"`
	StatusCode      int32       `db:"status_code" json:"status_code"`
}

// New creates a new StoreRequest instance with a database connection pool
func New(ctx context.Context, dbConnStr string) (dbStore *StoreRequest, fault error) {
	pool, err := pgxpool.New(ctx, dbConnStr)
	if err != nil {
		return nil, err
	}

	return &StoreRequest{
		pool: pool,
	}, nil
}

// NewWithPool returns a new StoreRequest instance with the provided database connection pool
func NewWithPool(pool *pgxpool.Pool) (dbStore *StoreRequest, fault error) {
	return &StoreRequest{
		pool: pool,
	}, nil
}

// Exec executes the database insert for the StoreRequest
func (s *StoreRequest) Exec(ctx context.Context) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	sql := `INSERT INTO remote_api_requests (
	url,
	method,
	request_headers,
	request_body,
	response_time_ms,
	response_headers,
	response_body,
	status_code
) VALUES (
	$1,
	$2,
	$3,
	$4,
	$5,
	$6,
	$7,
	$8
);`

	_, err = tx.Exec(ctx, sql, s.URL, s.Method, s.RequestHeaders, s.RequestBody, s.ResponseTimeMs, s.ResponseHeaders, s.ResponseBody, s.StatusCode)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

// SetURL sets the URL field of the StoreRequest
func (s *StoreRequest) SetURL(input string) {
	s.URL = input
}

// SetMethod sets the Method field of the StoreRequest
func (s *StoreRequest) SetMethod(input string) {
	s.Method = input
}

// SetRequestHeaders sets the RequestHeaders field of the StoreRequest
func (s *StoreRequest) SetRequestHeaders(input []byte) {
	s.RequestHeaders = input
}

// SetRequestBody sets the RequestBody field of the StoreRequest
func (s *StoreRequest) SetRequestBody(input pgtype.Text) {
	s.RequestBody = input
}

// SetResponseTimeMS sets the ResponseTimeMs field of the StoreRequest
func (s *StoreRequest) SetResponseTimeMS(input int64) {
	s.ResponseTimeMs = input
}

// SetResponseHeaders sets the ResponseHeaders field of the StoreRequest
func (s *StoreRequest) SetResponseHeaders(input []byte) {
	s.ResponseHeaders = input
}

// SetResponseBody sets the ResponseBody field of the StoreRequest
func (s *StoreRequest) SetResponseBody(input pgtype.Text) {
	s.ResponseBody = input
}

// SetStatusCode sets the StatusCode field of the StoreRequest
func (s *StoreRequest) SetStatusCode(input int32) {
	s.StatusCode = input
}
