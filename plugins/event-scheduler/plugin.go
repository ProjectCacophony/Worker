package eventscheduler

import (
	"github.com/jinzhu/gorm"
	"gitlab.com/Cacophony/Worker/plugins/common"
	"gitlab.com/Cacophony/go-kit/events"
	"gitlab.com/Cacophony/go-kit/interfaces"
	"go.uber.org/zap"
)

const (
	limit = 100

	selectQuery = `
UPDATE events_scheduled
SET published_at = NOW()
WHERE id IN (
  SELECT id
  FROM events_scheduled
  WHERE deleted_at IS NULL
  AND publish_at <= NOW()
  AND published_at IS NULL
  LIMIT $1
  FOR UPDATE SKIP LOCKED
)
RETURNING
  id,
  body
;
`
)

type Plugin struct {
	logger    *zap.Logger
	db        *gorm.DB
	publisher *events.Publisher
}

func (p *Plugin) Name() string {
	return "event-scheduler"
}

func (p *Plugin) Start(params common.StartParameters) error {
	err := params.DB.AutoMigrate(Entry{}).Error
	if err != nil {
		return err
	}

	p.logger = params.Logger
	p.db = params.DB
	p.publisher = params.Publisher

	return nil
}

func (p *Plugin) Stop(params common.StopParameters) error {
	return nil
}

func (p *Plugin) Localizations() []interfaces.Localization {
	return []interfaces.Localization{}
}

func (p *Plugin) Run(run *common.Run) (err error) {
	run.Logger().Info("run started")

	tx, err := p.db.DB().BeginTx(run.Context(), nil)
	if err != nil {
		tx.Rollback()
		return err
	}

	rows, err := tx.QueryContext(
		run.Context(), selectQuery,
		limit,
	)
	if err != nil {
		tx.Rollback()
		return err
	}

	defer tx.Commit()

	var entries []Entry
	for rows.Next() {
		var entry Entry

		err = rows.Scan(
			&entry.ID,
			&entry.Body,
		)
		if err != nil {
			return err
		}

		entries = append(entries, entry)
	}

	p.publishEvents(run, entries)

	return nil
}
