package tiktok

import (
	"net/http"
	"time"

	"github.com/go-redis/redis"
	"github.com/jinzhu/gorm"
	"github.com/kelseyhightower/envconfig"
	"gitlab.com/Cacophony/Worker/plugins/common"
	"gitlab.com/Cacophony/Worker/plugins/tiktok/client"
	"gitlab.com/Cacophony/go-kit/interfaces"
	"gitlab.com/Cacophony/go-kit/localization"
	"gitlab.com/Cacophony/go-kit/state"
	"go.uber.org/zap"
)

const (
	checkInterval = 1 * time.Hour
	limit         = 50

	selectQuery = `
UPDATE tiktok_entries
SET last_check = NOW()
WHERE id IN (
  SELECT id
  FROM tiktok_entries
  WHERE deleted_at IS NULL
  AND tik_tok_username IN (
    SELECT tik_tok_username
    FROM tiktok_entries
    WHERE deleted_at IS NULL
    AND disable_post_feed = false
    GROUP BY tik_tok_username
  	HAVING MAX(last_check) < $1
    LIMIT $2
  )
  FOR UPDATE SKIP LOCKED
)
RETURNING
  id,
  created_at,
  tik_tok_username,
  guild_id,
  channel_or_user_id,
  bot_id,
  dm
;
`
)

type Plugin struct {
	logger       *zap.Logger
	state        *state.State
	db           *gorm.DB
	tokens       map[string]string
	redis        *redis.Client
	tiktokClient *client.Client
}

func (p *Plugin) Name() string {
	return "tiktok"
}

func (p *Plugin) Interval() time.Duration {
	return 1 * time.Minute
}

type config struct {
	TikTokAPIBaseURL string `envconfig:"TIKTOK_API_BASE_URL"`
}

func (p *Plugin) Start(params common.StartParameters) error {
	var config config
	err := envconfig.Process("", &config)
	if err != nil {
		return err
	}

	err = params.DB.AutoMigrate(Post{}).Error
	if err != nil {
		return err
	}

	p.logger = params.Logger
	p.db = params.DB
	p.state = params.State
	p.tokens = params.Tokens

	p.redis = params.Redis

	p.tiktokClient = client.NewClient(
		&http.Client{
			Timeout: 2 * time.Minute,
		},
		config.TikTokAPIBaseURL,
	)

	return nil
}

func (p *Plugin) Stop(params common.StopParameters) error {
	return nil
}

func (p *Plugin) Localizations() []interfaces.Localization {
	local, err := localization.NewFileSource("assets/translations/tiktok.en.toml", "en")
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
			&entry.TikTokUsername,
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
