// nolint: dupl
package instagramigtv

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"github.com/jinzhu/gorm"
	"gitlab.com/Cacophony/go-kit/feed"
)

func postAdd(
	db *gorm.DB,
	entryID uint,
	postID string,
	messageIDs []string,
) error {
	return db.Create(&Post{
		EntryID:    entryID,
		PostID:     postID,
		MessageIDs: messageIDs,
	}).Error
}

func postFind(db *gorm.DB, where ...interface{}) (*Post, error) {
	var post Post

	err := db.First(&post, where...).Order("created_at DESC").Error
	if err != nil {
		return nil, err
	}
	return &post, err
}

func checkSet(ctx context.Context, tx *sql.Tx, status feed.Status, message string, entries ...Entry) error {
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
SET igtv_check_status = $2, igtv_check_message = $3
WHERE id in ($1)
`, strings.Join(ids, ","), status, message)
	return err
}
