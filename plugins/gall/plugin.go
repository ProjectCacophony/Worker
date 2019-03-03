package gall

import (
	"net/http"
	"time"

	"gitlab.com/Cacophony/go-kit/state"

	"github.com/Seklfreak/ginside"

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
UPDATE gall_entries
SET last_check = NOW()
WHERE id IN (
  SELECT id FROM gall_entries
  WHERE last_check < $1
  ORDER BY last_check ASC
  FOR UPDATE SKIP LOCKED
  LIMIT $2
)
RETURNING
  id,
  created_at,
  board_id,
  minor_gallery,
  recommended,
  guild_id,
  channel_id
;
`
)

type Plugin struct {
	logger *zap.Logger
	state  *state.State
	db     *gorm.DB
	gall   *ginside.GInside
	tokens map[string]string
}

func (p *Plugin) Name() string {
	return "gall"
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

	p.gall = ginside.NewGInside(&http.Client{
		Timeout: 60 * time.Second,
	})

	return nil
}

func (p *Plugin) Stop(params common.StopParameters) error {
	return nil
}

func (p *Plugin) Localisations() []interfaces.Localisation {
	local, err := localisation.NewFileSource("assets/translations/gall.en.toml", "en")
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
			&entry.BoardID,
			&entry.MinorGallery,
			&entry.Recommended,
			&entry.GuildID,
			&entry.ChannelID,
		)
		if err != nil {
			return err
		}

		entries = append(entries, entry)
	}

	bundles := p.bundleEntries(run, entries)

	p.checkBundles(run, tx, bundles)

	return nil
}