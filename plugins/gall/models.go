package gall

import (
	"time"

	"github.com/jinzhu/gorm"
	"github.com/lib/pq"
	"gitlab.com/Cacophony/go-kit/feed"
)

// Entry model maintained by Processor
type Entry struct {
	gorm.Model
	GuildID   string
	ChannelID string // UserID in case of DMs
	AddedBy   string
	BotID     string // only relevant for DMs

	BoardID      string
	MinorGallery bool
	Recommended  bool

	DM bool

	LastCheck time.Time
	feed.Check
}

func (*Entry) TableName() string {
	return "gall_entries"
}

type Post struct {
	gorm.Model
	EntryID uint

	PostID     string
	MessageIDs pq.StringArray `gorm:"type:varchar[]"`
}

func (*Post) TableName() string {
	return "gall_posts"
}
