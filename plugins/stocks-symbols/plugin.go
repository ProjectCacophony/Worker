package stockssymbols

import (
	"net/http"
	"time"

	"github.com/go-redis/redis"
	"github.com/jinzhu/gorm"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"gitlab.com/Cacophony/Worker/plugins/common"
	"gitlab.com/Cacophony/go-kit/external/iexcloud"
	"gitlab.com/Cacophony/go-kit/interfaces"
	"go.uber.org/zap"
)

const checkInterval = time.Hour * 24

// TODO: generate list automatically from exchanges endpoint
var regions = []string{
	"AE", "BE", "DE", "FR", "GB", "IE", "NL", "PT", "CA", "MX", "US",
}

type Plugin struct {
	logger    *zap.Logger
	db        *gorm.DB
	redis     *redis.Client
	iexClient *iexcloud.IEX
}

func (p *Plugin) Name() string {
	return "stocks-symbols"
}

func (p *Plugin) Interval() time.Duration {
	return 1 * time.Hour
}

func (p *Plugin) Start(params common.StartParameters) error {
	err := params.DB.AutoMigrate(Symbol{}).Error
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

	if config.IEXAPISecret == "" {
		return errors.New("stocks plugin configuration missing")
	}

	p.iexClient = iexcloud.NewIEX(
		&http.Client{
			Timeout: time.Second * 60,
		},
		config.IEXAPISecret,
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
	if p.iexClient == nil {
		run.Logger().Info("skipped run, iex client is not initialised")
		return nil
	}

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

	err = p.cacheAllSymbols(run)
	if err != nil {
		return err
	}

	err = p.setRun()
	if err != nil {
		return errors.Wrap(err, "error setting last run time")
	}

	run.Logger().Debug("completed",
		zap.Duration("took", time.Since(run.Launch)),
	)

	return nil
}
