// Package cleaner provides functionality to purge API request and response records stored in a PostgreSQL database
// that were created by go11y's AddDBStorer transport middleware.
// Not all services using go11y's AddDBStorer transport middleware need to implement the cleaner, only those that pass
// PII to external services though a client using go11y's AddDBStorer transport middleware.
package cleaner

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Cleaner struct for cleaning up old records created by the storer
type Cleaner struct {
	pool       *pgxpool.Pool
	maxAgeDays string
}

const maxAge = "180 days" // roughly 6 months

// New creates a new Cleaner instance with a database connection pool
func New(ctx context.Context, dbConnStr, maxAgeDays string) (dbCleaner *Cleaner, fault error) {
	pool, err := pgxpool.New(ctx, dbConnStr)
	if err != nil {
		return nil, err
	}

	return &Cleaner{
		pool:       pool,
		maxAgeDays: maxAgeDays,
	}, nil
}

// Exec cleans the clears out db records created by the storer that are older than 180 days
func (s *Cleaner) Exec(ctx context.Context) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	sql := fmt.Sprintf(`DELETE FROM remote_api_requests WHERE created_at < (NOW() - interval '%s days');`, s.maxAgeDays)

	_, err = tx.Exec(ctx, sql)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

// Close closes the Cleaner's database connection
func (s *Cleaner) Close(ctx context.Context) {
	s.pool.Close()
}
