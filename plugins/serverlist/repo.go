package serverlist

import (
	"context"
	"database/sql"
	"errors"
)

func setMemberCount(ctx context.Context, tx *sql.Tx, id uint, members int) error {
	if id == 0 {
		return errors.New("submitted invalid server id")
	}

	_, err := tx.ExecContext(ctx, `
UPDATE serverlist_servers
SET total_members = $2
WHERE id = $1
`, id, members)
	return err
}

func setExpired(ctx context.Context, tx *sql.Tx, id uint) error {
	if id == 0 {
		return errors.New("submitted invalid server id")
	}

	_, err := tx.ExecContext(ctx, `
UPDATE serverlist_servers
SET state = $2
WHERE id = $1
`, id, StateExpired)
	return err
}
