package serverlist

import (
	"time"

	"gitlab.com/Cacophony/go-kit/amqp"

	"github.com/go-redis/redis"

	"github.com/jinzhu/gorm"
	"gitlab.com/Cacophony/Worker/plugins/common"
	"gitlab.com/Cacophony/go-kit/interfaces"
	"go.uber.org/zap"
)

const (
	checkInterval = time.Minute * 15
	limit         = 10

	selectQuery = `
UPDATE serverlist_servers
SET last_checked = NOW()
WHERE id IN (
  SELECT id
  FROM serverlist_servers
  WHERE deleted_at IS NULL
  AND state = 'public'
  AND last_checked < $1
  LIMIT $2
  FOR UPDATE SKIP LOCKED
)
RETURNING
  id,
  created_at,
  invite_code,
  guild_id,
  total_members,
  state,
  last_checked,
  bot_id
;
`
)

type Plugin struct {
	logger    *zap.Logger
	db        *gorm.DB
	redis     *redis.Client
	tokens    map[string]string
	publisher *amqp.Publisher
}

func (p *Plugin) Name() string {
	return "serverlist"
}

func (p *Plugin) Start(params common.StartParameters) error {

	p.logger = params.Logger
	p.db = params.DB
	p.redis = params.Redis
	p.tokens = params.Tokens
	p.publisher = params.Publisher

	return nil
}

func (p *Plugin) Stop(params common.StopParameters) error {
	return nil
}

func (p *Plugin) Localisations() []interfaces.Localisation {
	return nil
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
		time.Now().Add(-checkInterval), limit,
	)
	if err != nil {
		tx.Rollback()
		return err
	}

	defer tx.Commit()

	var entries []Server
	for rows.Next() {
		var entry Server

		err = rows.Scan(
			&entry.ID,
			&entry.CreatedAt,
			&entry.InviteCode,
			&entry.GuildID,
			&entry.TotalMembers,
			&entry.State,
			&entry.LastChecked,
			&entry.BotID,
		)
		if err != nil {
			return err
		}

		entries = append(entries, entry)
	}

	p.checkEntries(run, tx, entries)

	return nil
}
