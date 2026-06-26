package postgres

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// toTS wraps a time.Time into a non-null pgtype.Timestamptz.
func toTS(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// toTSPtr wraps an optional time into a pgtype.Timestamptz (null when nil).
func toTSPtr(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

// fromTSPtr converts a pgtype.Timestamptz back into an optional time.
func fromTSPtr(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time
	return &t
}
