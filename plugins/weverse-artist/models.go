package weverseartist

import (
	"time"

	"github.com/lib/pq"
	"gitlab.com/Cacophony/go-kit/feed"

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

	WeverseChannelName string
	WeverseChannelID   int64

	ArtistFeedLastCheck time.Time
	ArtistFeedCheck     feed.Check `gorm:"embedded;embedded_prefix:artist_"`

	MediaFeedLastCheck time.Time
	MediaFeedCheck     feed.Check `gorm:"embedded;embedded_prefix:media_"`

	NoticeFeedLastCheck time.Time
	NoticeFeedCheck     feed.Check `gorm:"embedded;embedded_prefix:notice_"`
}

func (*Entry) TableName() string {
	return "weverse_entries"
}

type Post struct {
	gorm.Model
	EntryID uint

	PostType   string
	PostID     int64
	MessageIDs pq.StringArray `gorm:"type:varchar[]"`
}

func (*Post) TableName() string {
	return "weverse_posts"
}
