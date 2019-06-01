package patrons

import (
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

func upsertPatron(db *gorm.DB, patron *Patron) error {
	if patron == nil {
		return errors.New("patron cannot be nil")
	}

	err := db.
		Where(&Patron{
			PatreonUserID: patron.PatreonUserID,
		}).
		Assign(patron).
		FirstOrCreate(&Patron{}).
		Error
	if err != nil {
		return err
	}

	return nil
}
