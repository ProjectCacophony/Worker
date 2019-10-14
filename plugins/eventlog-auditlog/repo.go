package eventlogauditlog

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

func addReason(ctx context.Context, tx *sql.Tx, entryID uint, reason string) error {
	if entryID == 0 {
		return errors.New("entryID cannot be empty")
	}
	if reason == "" {
		return errors.New("reason cannot be empty")
	}

	query := `
UPDATE eventlog_items
SET reasons = array_append(reasons, $1)
WHERE id = $2
;
`

	_, err := tx.ExecContext(ctx, query, reason, entryID)
	return err
}

func addItemOption(ctx context.Context, tx *sql.Tx, entryID uint, key, previousValue, newValue, optionType string) error {
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
INSERT INTO eventlog_item_options ("created_at", "updated_at", "item_id", "key", "previous_value", "new_value", "type")
VALUES ($1, $2, $3, $4, $5, $6, $7)
;
`

	now := time.Now().UTC()
	_, err := tx.ExecContext(ctx, query, now, now, entryID, key, previousValue, newValue, optionType)
	return err
}
