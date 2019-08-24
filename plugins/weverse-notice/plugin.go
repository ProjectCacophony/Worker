package weversenotice

import (
	"net/http"
	"time"

	"github.com/Seklfreak/geverse"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"

	"github.com/go-redis/redis"

	"gitlab.com/Cacophony/go-kit/localization"
	"gitlab.com/Cacophony/go-kit/state"

	"github.com/jinzhu/gorm"
	"gitlab.com/Cacophony/Worker/plugins/common"
	"gitlab.com/Cacophony/go-kit/interfaces"
	"go.uber.org/zap"
)

const (
	checkInterval = 10 * time.Minute
	limit         = 10
	postType      = "notice"

	selectQuery = `
UPDATE weverse_entries
SET notice_feed_last_check = NOW()
WHERE id IN (
  SELECT id
  FROM weverse_entries
  WHERE deleted_at IS NULL
  AND weverse_channel_id IN (
    SELECT weverse_channel_id
    FROM weverse_entries
    WHERE deleted_at IS NULL
    GROUP BY weverse_channel_id
  	HAVING MAX(notice_feed_last_check) < $1
    LIMIT $2
  )
  FOR UPDATE SKIP LOCKED
)
RETURNING
  id,
  created_at,
  weverse_channel_name,
  weverse_channel_id,
  guild_id,
  channel_or_user_id,
  bot_id,
  dm
;
`
)

type Plugin struct {
	logger  *zap.Logger
	state   *state.State
	db      *gorm.DB
	tokens  map[string]string
	redis   *redis.Client
	geverse *geverse.Geverse
}

func (p *Plugin) Name() string {
	return "weverse-notice"
}

func (p *Plugin) Interval() time.Duration {
	return 1 * time.Minute
}

type weverseConfig struct {
	WeverseToken string `envconfig:"WEVERSE_TOKEN"`
}

func (p *Plugin) Start(params common.StartParameters) error {
	var config weverseConfig
	err := envconfig.Process("", &config)
	if err != nil {
		return errors.Wrap(err, "failure loading weverse module config")
	}

	err = params.DB.AutoMigrate(Post{}).Error
	if err != nil {
		return err
	}

	p.logger = params.Logger
	p.db = params.DB
	p.state = params.State
	p.tokens = params.Tokens

	p.geverse = geverse.NewGeverse(
		&http.Client{
			Timeout: 2 * time.Minute,
		},
		config.WeverseToken,
	)

	p.redis = params.Redis

	return nil
}

func (p *Plugin) Stop(params common.StopParameters) error {
	return nil
}

func (p *Plugin) Localizations() []interfaces.Localization {
	local, err := localization.NewFileSource("assets/translations/weverse.en.toml", "en")
	if err != nil {
		p.logger.Error("failed to load localisation", zap.Error(err))
	}

	return []interfaces.Localization{local}
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

	var entries []Entry
	for rows.Next() {
		var entry Entry

		err = rows.Scan(
			&entry.ID,
			&entry.CreatedAt,
			&entry.WeverseChannelName,
			&entry.WeverseChannelID,
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
