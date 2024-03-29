package instagramfillid

import (
	"time"

	"github.com/jinzhu/gorm"
	"gitlab.com/Cacophony/go-kit/feed"
)

// Entry model maintained by Processor
type Entry struct {
	gorm.Model
	GuildID          string
	ChannelOrUserID  string // UserID in case of DMs
	DM               bool
	DisablePostFeed  bool `gorm:"default:false"`
	DisableStoryFeed bool `gorm:"default:false"`
	AddedBy          string
	BotID            string // only relevant for DMs

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
