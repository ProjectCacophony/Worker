package stockssymbols

import (
	"errors"

	"github.com/jinzhu/gorm"
)

func existsSymbol(db *gorm.DB, symbol string) (bool, error) {
	var exists int
	err := db.Model(Symbol{}).Where("symbol = ?", symbol).Count(&exists).Error
	return exists > 0, err
}

func upsertSymbol(db *gorm.DB, symbol *Symbol) error {
	if symbol == nil {
		return errors.New("symbol cannot be nil")
	}

	err := db.
		Where(&Symbol{
			Symbol: symbol.Symbol,
		}).
		Assign(symbol).
		FirstOrCreate(&Symbol{}).
		Error
	if err != nil {
		return err
	}

	return nil
}
