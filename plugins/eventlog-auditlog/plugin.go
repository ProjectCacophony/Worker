package eventlogauditlog

import (
	"time"

	"github.com/getsentry/raven-go"
	"github.com/jinzhu/gorm"
	"gitlab.com/Cacophony/Worker/plugins/common"
	"gitlab.com/Cacophony/go-kit/events"
	"gitlab.com/Cacophony/go-kit/interfaces"
	"gitlab.com/Cacophony/go-kit/state"
	"go.uber.org/zap"
)

const (
	limit = 10

	selectQuery = `
UPDATE eventlog_items
SET waiting_for_audit_log_backfill = false
WHERE id IN (
  SELECT id
  FROM eventlog_items
  WHERE deleted_at IS NULL
  AND waiting_for_audit_log_backfill = true
  AND guild_id IN (
    SELECT guild_id
    FROM eventlog_items
    WHERE deleted_at IS NULL
    AND waiting_for_audit_log_backfill = true
    GROUP BY guild_id
    LIMIT $1
  )
  FOR UPDATE SKIP LOCKED
)
RETURNING
  id,
  created_at,
  guild_id,
  action_type,
  author_id,
  target_type,
  COALESCE(target_value, '')
;
`
)

type Plugin struct {
	logger    *zap.Logger
	state     *state.State
	db        *gorm.DB
	tokens    map[string]string
	publisher *events.Publisher
}

func (p *Plugin) Name() string {
	return "eventlog-auditlog"
}

func (p *Plugin) Interval() time.Duration {
	return 1 * time.Minute
}

func (p *Plugin) Start(params common.StartParameters) error {
	p.logger = params.Logger
	p.db = params.DB
	p.state = params.State
	p.tokens = params.Tokens
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
		run.Context(), selectQuery, limit,
	)
	if err != nil {
		tx.Rollback()
		return err
	}

	var entries []Item
	for rows.Next() {
		var entry Item

		err = rows.Scan(
			&entry.ID,
			&entry.CreatedAt,
			&entry.GuildID,
			&entry.ActionType,
			&entry.AuthorID,
			&entry.TargetType,
			&entry.TargetValue,
		)
		if err != nil {
			tx.Rollback()
			return err
		}

		entries = append(entries, entry)
	}

	bundles := p.bundleEntries(run, entries)

	updated := p.checkBundles(run, tx, bundles)

	tx.Commit()

	for _, item := range updated {
		event, err := events.New(events.CacophonyEventlogUpdate)
		if err != nil {
			run.Except(err)
			continue
		}
		event.EventlogUpdate = &events.EventlogUpdate{
			GuildID: item.GuildID,
			ItemID:  item.ID,
		}
		event.GuildID = item.GuildID

		err, recoverable := p.publisher.Publish(run.Context(), event)
		if err != nil && !recoverable {
			raven.CaptureError(err, nil)
			zap.L().Fatal(
				"received unrecoverable error while publishing \"eventlog update\" message",
				zap.Error(err),
			)
		}
	}

	return nil
}
