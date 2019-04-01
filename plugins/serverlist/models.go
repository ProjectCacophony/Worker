package serverlist

import (
	"time"

	"github.com/jinzhu/gorm"
)

// State model maintained by Processor
type State string

const (
	StateExpired State = "expired"
)

// Server model maintained by Processor
type Server struct {
	gorm.Model
	// Names         pq.StringArray `gorm:"type:varchar[]"`
	// Description   string
	InviteCode string
	GuildID    string
	// EditorUserIDs pq.StringArray `gorm:"type:varchar[]"`
	// Categories    []ServerCategory
	TotalMembers int
	State        State
	// Reason       string
	LastChecked time.Time
	BotID       string
	// Change        ServerChange `gorm:"embedded;embedded_prefix:change_"`
}

func (*Server) TableName() string {
	return "serverlist_servers"
}
