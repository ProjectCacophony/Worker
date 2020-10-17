package vlive

import (
	"errors"

	"github.com/jinzhu/gorm"
)

func findEntriesForChannel(db *gorm.DB, channelID string) ([]*Entry, error) {
	if channelID == "" {
		return nil, errors.New("channelID cannot be empty")
	}

	var entries []*Entry
	err := db.Find(&entries, "v_live_channel_id = ?", channelID).Error
	return entries, err
}
