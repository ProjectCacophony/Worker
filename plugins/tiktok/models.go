package tiktok

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
	DisablePostFeed bool `gorm:"default:false"`
	AddedBy         string
	BotID           string // only relevant for DMs

	TikTokUsername string

	LastCheck time.Time
	feed.Check
}

func (*Entry) TableName() string {
	return "tiktok_entries"
}

type Post struct {
	gorm.Model
	EntryID uint

	PostID     string
	MessageIDs pq.StringArray `gorm:"type:varchar[]"`
}

func (*Post) TableName() string {
	return "tiktok_posts"
}
