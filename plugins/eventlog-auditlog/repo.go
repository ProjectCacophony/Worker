package eventlogauditlog

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

func setAuthor(ctx context.Context, tx *sql.Tx, entryID uint, authorID string) error {
	if entryID == 0 {
		return errors.New("entryID cannot be empty")
	}
	if authorID == "" {
		return errors.New("authorID cannot be empty")
	}

	query := `
UPDATE eventlog_items
SET author_id = $1
WHERE id = $2
;
`

	_, err := tx.ExecContext(ctx, query, authorID, entryID)
	return err
}

// nolint: unparam
func addItemOption(ctx context.Context, tx *sql.Tx, entryID uint, key, previousValue, newValue, optionType, botID string) error {
	if entryID == 0 {
		return errors.New("entryID cannot be empty")
	}
	if previousValue == "" && newValue == "" {
		return errors.New("previousValue and newValue cannot be empty at the same time")
	}
	if optionType == "" {
		return errors.New("optionType cannot be empty")
	}

	query := `
INSERT INTO eventlog_item_options ("created_at", "updated_at", "item_id", "key", "previous_value", "new_value", "type", "author_id")
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
;
`

	now := time.Now().UTC()
	_, err := tx.ExecContext(ctx, query, now, now, entryID, key, previousValue, newValue, optionType, botID)
	return err
}
