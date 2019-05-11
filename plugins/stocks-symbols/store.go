package stockssymbols

import (
	"time"

	"github.com/pkg/errors"
	"gitlab.com/Cacophony/Worker/plugins/common"
	"go.uber.org/zap"
)

func (p *Plugin) cacheAllSymbols(run *common.Run) error {
	var err error

	for _, region := range regions {
		run.Logger().Info("caching symbols",
			zap.String("region", region),
		)

		err = p.cacheSymbols(run, region)
		if err != nil {
			return errors.Wrapf(err, "error caching symbols for region %s", region)
		}
	}

	return nil
}

func (p *Plugin) cacheSymbols(run *common.Run, region string) error {
	symbols, err := p.iexClient.RefDataSymbolsInternational(run.Context(), region)
	if err != nil {
		return errors.Wrapf(err, "error getting symbols from API for region %s", region)
	}

	var exists bool
	for _, symbolData := range symbols {
		exists, err = existsSymbol(p.db, symbolData.Symbol)
		if err != nil {
			return errors.Wrapf(err, "error querying db if symbol %s exists", symbolData.Symbol)
		}
		if exists {
			continue
		}

		symbol := &Symbol{
			Symbol:   symbolData.Symbol,
			Exchange: symbolData.Exchange,
			Name:     symbolData.Name,
			Date:     time.Time(symbolData.Date),
			Type:     symbolData.Type,
			IEXID:    symbolData.IEXID,
			Region:   symbolData.Region,
			Currency: symbolData.Currency,
		}

		err = upsertSymbol(p.db, symbol)
		if err != nil {
			return errors.Wrapf(err, "error upserting symbol %s", symbolData.Symbol)
		}
	}

	return nil
}
