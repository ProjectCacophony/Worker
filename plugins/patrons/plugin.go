package patrons

import (
	"net/http"
	"time"

	"github.com/go-redis/redis"
	"github.com/jinzhu/gorm"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"gitlab.com/Cacophony/Worker/plugins/common"
	"gitlab.com/Cacophony/go-kit/interfaces"
	"go.uber.org/zap"
)

const (
	apiBase       = "https://www.patreon.com/"
	checkInterval = time.Hour * 24
)

type Plugin struct {
	logger *zap.Logger
	db     *gorm.DB
	redis  *redis.Client
	client *PatreonAPI
}

func (p *Plugin) Name() string {
	return "patrons"
}

func (p *Plugin) Interval() time.Duration {
	return 1 * time.Hour
}

func (p *Plugin) Start(params common.StartParameters) error {
	err := params.DB.AutoMigrate(Patron{}).Error
	if err != nil {
		return err
	}

	p.logger = params.Logger
	p.db = params.DB
	p.redis = params.Redis

	var config Config
	err = envconfig.Process("", &config)
	if err != nil {
		return nil
	}

	p.client = NewPatreonAPI(
		&http.Client{
			Timeout: 1 * time.Minute,
		},
		config.PatreonCreatorsAccessToken,
		config.PatreonCampaignID,
	)

	return nil
}

func (p *Plugin) Stop(params common.StopParameters) error {
	return nil
}

func (p *Plugin) Localizations() []interfaces.Localization {
	return nil
}

func (p *Plugin) Run(run *common.Run) (err error) {
	run.Logger().Info("run started")

	lock := p.getRunLock()
	locked, err := lock.LockWithContext(run.Context())
	if err != nil {
		return errors.Wrap(err, "error acquring lock")
	}
	if !locked {
		run.Logger().Info("skipped run, another run is already in progress")
		return nil
	}
	defer lock.Unlock()

	shouldRun, err := p.shouldRun()
	if err != nil {
		return errors.Wrap(err, "error finding out if run should happen")
	}
	if !shouldRun {
		run.Logger().Debug("skipped run, previous run recently enough",
			zap.Duration("check_interval", checkInterval),
		)
		return nil
	}

	patrons, err := p.getPatrons(run)
	if err != nil {
		return errors.Wrap(err, "error gathering patrons")
	}

	err = p.upsertPatrons(patrons)
	if err != nil {
		return errors.Wrap(err, "error upserting patrons")
	}

	run.Logger().Info("stored or updated patrons", zap.Int("amount", len(patrons)))

	err = p.setRun()
	if err != nil {
		return errors.Wrap(err, "error setting last run time")
	}

	run.Logger().Debug("completed",
		zap.Duration("took", time.Since(run.Launch)),
	)

	return nil
}
