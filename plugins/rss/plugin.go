package rss

import (
	"net/http"
	"time"

	"github.com/go-redis/redis"
	"github.com/jinzhu/gorm"
	"github.com/mmcdole/gofeed"
	"gitlab.com/Cacophony/Worker/plugins/common"
	"gitlab.com/Cacophony/go-kit/interfaces"
	"gitlab.com/Cacophony/go-kit/localization"
	"gitlab.com/Cacophony/go-kit/state"
	"go.uber.org/zap"
)

const (
	checkInterval = time.Minute * 15
	limit         = 100

	selectQuery = `
UPDATE rss_entries
SET last_check = NOW()
WHERE id IN (
  SELECT id
  FROM rss_entries
  WHERE deleted_at IS NULL
  AND feed_url IN (
    SELECT feed_url
    FROM rss_entries
    WHERE deleted_at IS NULL
    GROUP BY feed_url
  	HAVING MAX(last_check) < $1
    LIMIT $2
  )
  FOR UPDATE SKIP LOCKED
)
RETURNING
  id,
  created_at,
  guild_id,
  channel_id,
  feed_url,
  name,
  bot_id,
  dm
;
`
)

type Plugin struct {
	logger     *zap.Logger
	state      *state.State
	db         *gorm.DB
	tokens     map[string]string
	parser     *gofeed.Parser
	httpClient *http.Client
	redis      *redis.Client
}

func (p *Plugin) Name() string {
	return "rss"
}

func (p *Plugin) Interval() time.Duration {
	return 1 * time.Minute
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
	p.parser = gofeed.NewParser()
	p.httpClient = &http.Client{
		Timeout: time.Second * 60,
	}
	p.redis = params.Redis

	return nil
}

func (p *Plugin) Stop(params common.StopParameters) error {
	return nil
}

func (p *Plugin) Localizations() []interfaces.Localization {
	local, err := localization.NewFileSource("assets/translations/rss.en.toml", "en")
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
			&entry.GuildID,
			&entry.ChannelID,
			&entry.FeedURL,
			&entry.Name,
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
