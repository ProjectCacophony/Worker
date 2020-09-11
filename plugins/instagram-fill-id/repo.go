package instagramfillid

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"github.com/jinzhu/gorm"
	"gitlab.com/Cacophony/go-kit/feed"
)

func findIDByUsername(db *gorm.DB, username string) (string, error) {
	var entry Entry

	err := db.Take(&entry,
		"instagram_account_id <> '' AND instagram_username = ?",
		username,
	).Error
	return entry.InstagramAccountID, err
}

func checkSetError(ctx context.Context, tx *sql.Tx, message string, entries ...Entry) error {
	var ids []string // nolint: prealloc
	for _, entry := range entries {
		if entry.ID == 0 {
			continue
		}

		ids = append(ids, strconv.FormatUint(uint64(entry.ID), 10))
	}

	if len(ids) == 0 {
		return nil
	}

	_, err := tx.ExecContext(ctx, `
UPDATE instagram_entries
SET check_status = $1, check_message = $2
WHERE id in (`+strings.Join(ids, ",")+`)
`, feed.ErrorStatus, message)
	return err
}

func setInstagramID(ctx context.Context, tx *sql.Tx, entryID uint, newID string) error {
	if entryID == 0 || newID == "" {
		return errors.New("invalid information passed")
	}

	_, err := tx.ExecContext(ctx, `
UPDATE instagram_entries
SET instagram_account_id = $2
WHERE id = $1
`, entryID, newID)
	return err
}

func deleteEntry(ctx context.Context, tx *sql.Tx, entryID uint) error {
	if entryID == 0 {
		return errors.New("invalid information passed")
	}

	_, err := tx.ExecContext(ctx, `
UPDATE instagram_entries
SET deleted_at = now()
WHERE id = $1
;
`, entryID)
	return err
}
