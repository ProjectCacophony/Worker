package stockssymbols

import (
	"time"

	"github.com/jinzhu/gorm"
)

type Symbol struct {
	gorm.Model
	Symbol   string
	Exchange string
	Name     string
	Date     time.Time
	Type     string
	IEXID    string
	Region   string
	Currency string
}

func (*Symbol) TableName() string {
	return "stocks_symbols"
}
