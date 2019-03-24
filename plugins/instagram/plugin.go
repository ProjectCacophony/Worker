// nolint: dupl
package instagram

import (
	"net/http"
	"time"

	"github.com/Seklfreak/ginsta"

	"github.com/go-redis/redis"

	"gitlab.com/Cacophony/go-kit/state"

	"github.com/jinzhu/gorm"
	"gitlab.com/Cacophony/Worker/plugins/common"
	"gitlab.com/Cacophony/go-kit/interfaces"
	"gitlab.com/Cacophony/go-kit/localisation"
	"go.uber.org/zap"
)

const (
	checkInterval = time.Minute * 10
	limit         = 10

	selectQuery = `
UPDATE instagram_entries
SET last_check = NOW()
WHERE id IN (
  SELECT id
  FROM instagram_entries
  WHERE deleted_at IS NULL
  AND instagram_account_id IN (
    SELECT instagram_account_id
    FROM instagram_entries
    WHERE deleted_at IS NULL
    GROUP BY instagram_account_id
  	HAVING MAX(last_check) < $1
    LIMIT $2
  )
  FOR UPDATE SKIP LOCKED
)
RETURNING
  id,
  created_at,
  instagram_account_id,
  instagram_username,
  guild_id,
  channel_or_user_id,
  bot_id,
  dm
;
`
)

type Plugin struct {
	logger *zap.Logger
	state  *state.State
	db     *gorm.DB
	tokens map[string]string
	redis  *redis.Client
	ginsta *ginsta.Ginsta
}

func (p *Plugin) Name() string {
	return "instagram"
}

func (p *Plugin) Start(params common.StartParameters) error {
	err := params.DB.AutoMigrate(Post{}).Error
	if err != nil {
		return err
	}

	p.logger = params.Logger
	p.db = params.DB
	p.state = params.State
	p.tokens = params.Tokens

	p.ginsta = ginsta.NewGinsta(
		&http.Client{
			Timeout: 2 * time.Minute,
		},
		nil,
	)

	p.redis = params.Redis

	return nil
}

func (p *Plugin) Stop(params common.StopParameters) error {
	return nil
}

func (p *Plugin) Localisations() []interfaces.Localisation {
	local, err := localisation.NewFileSource("assets/translations/instagram.en.toml", "en")
	if err != nil {
		p.logger.Error("failed to load localisation", zap.Error(err))
	}

	return []interfaces.Localisation{local}
}

func (p *Plugin) Run(run *common.Run) (err error) {
	run.Logger().Info("run started")

	tx, err := p.db.DB().BeginTx(run.Context(), nil)
	if err != nil {
		tx.Rollback() // nolint: errcheck
		return err
	}

	rows, err := tx.QueryContext(
		run.Context(), selectQuery,
		time.Now().Add(-checkInterval), limit,
	)
	if err != nil {
		tx.Rollback() // nolint: errcheck
		return err
	}

	defer tx.Commit() // nolint: errcheck

	var entries []Entry
	for rows.Next() {
		var entry Entry

		err = rows.Scan(
			&entry.ID,
			&entry.CreatedAt,
			&entry.InstagramAccountID,
			&entry.InstagramUsername,
			&entry.GuildID,
			&entry.ChannelOrUserID,
			&entry.BotID,
			&entry.DM,
		)
		if err != nil {
			return err
		}

		entries = append(entries, entry)
	}

	bundles := p.bundleEntries(run, tx, entries)

	p.checkBundles(run, tx, bundles)

	return nil
}
