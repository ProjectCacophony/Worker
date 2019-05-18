package eventscheduler

import (
	"time"

	"github.com/jinzhu/gorm"
)

type Entry struct {
	gorm.Model
	Body        []byte
	PublishAt   time.Time
	PublishedAt *time.Time
}

func (*Entry) TableName() string {
	return "events_scheduled"
}
