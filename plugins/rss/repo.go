package rss

import (
	"github.com/jinzhu/gorm"
)

func postAdd(
	db *gorm.DB,
	entryID uint,
	postGUID string,
	postLink string,
	messageIDs []string,
) error {
	return db.Create(&Post{
		Model:      gorm.Model{},
		EntryID:    entryID,
		PostGUID:   postGUID,
		PostLink:   postLink,
		MessageIDs: messageIDs,
	}).Error
}

func postFind(db *gorm.DB, where ...interface{}) (*Post, error) {
	var post Post

	err := db.First(&post, where...).Order("created_at DESC").Error
	if err != nil {
		return nil, err
	}
	return &post, err
}
