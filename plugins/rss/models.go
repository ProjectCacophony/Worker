package rss

import (
	"time"

	"github.com/lib/pq"

	"github.com/jinzhu/gorm"
)

// Entry model maintained by Processor
type Entry struct {
	gorm.Model
	GuildID   string
	ChannelID string
	AddedBy   string

	Name    string
	URL     string
	FeedURL string

	LastCheck time.Time
}

func (*Entry) TableName() string {
	return "rss_entries"
}

type Post struct {
	gorm.Model
	EntryID uint

	PostGUID   string
	PostLink   string
	MessageIDs pq.StringArray `gorm:"type:varchar[]"`
}

func (*Post) TableName() string {
	return "rss_posts"
}
