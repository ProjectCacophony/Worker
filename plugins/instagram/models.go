package instagram

import (
	"time"

	"gitlab.com/Cacophony/go-kit/feed"

	"github.com/lib/pq"

	"github.com/jinzhu/gorm"
)

// Entry model maintained by Processor
type Entry struct {
	gorm.Model
	GuildID         string
	ChannelOrUserID string // UserID in case of DMs
	DM              bool
	AddedBy         string
	BotID           string // only relevant for DMs

	InstagramUsername  string
	InstagramAccountID string

	LastCheck time.Time
	feed.Check

	StoriesLastCheck time.Time
	StoriesCheck     feed.Check `gorm:"embedded;embedded_prefix:stories_"`
}

func (*Entry) TableName() string {
	return "instagram_entries"
}

type Post struct {
	gorm.Model
	EntryID uint

	PostID     string
	MessageIDs pq.StringArray `gorm:"type:varchar[]"`
}

func (*Post) TableName() string {
	return "instagram_posts"
}
