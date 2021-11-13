package vlive

import (
	"github.com/jinzhu/gorm"
	"github.com/lib/pq"
)

// Entry model maintained by Processor
type Entry struct {
	gorm.Model
	GuildID         string
	ChannelOrUserID string // UserID in case of DMs
	DM              bool
	AddedBy         string
	BotID           string // only relevant for DMs

	VLiveChannelID string
}

func (*Entry) TableName() string {
	return "vlive_entries"
}

type Post struct {
	gorm.Model
	EntryID uint

	PostID     string
	MessageIDs pq.StringArray `gorm:"type:varchar[]"`
}

func (*Post) TableName() string {
	return "vlive_posts"
}
