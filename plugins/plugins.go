package plugins

import (
	"time"

	"github.com/go-redis/redis"
	"github.com/jinzhu/gorm"
	"gitlab.com/Cacophony/Worker/plugins/common"
	eventscheduler "gitlab.com/Cacophony/Worker/plugins/event-scheduler"
	"gitlab.com/Cacophony/Worker/plugins/gall"
	"gitlab.com/Cacophony/Worker/plugins/instagram"
	instagramstories "gitlab.com/Cacophony/Worker/plugins/instagram-stories"
	"gitlab.com/Cacophony/Worker/plugins/patrons"
	"gitlab.com/Cacophony/Worker/plugins/rss"
	"gitlab.com/Cacophony/Worker/plugins/serverlist"
	stockssymbols "gitlab.com/Cacophony/Worker/plugins/stocks-symbols"
	weverseartist "gitlab.com/Cacophony/Worker/plugins/weverse-artist"
	weversemedia "gitlab.com/Cacophony/Worker/plugins/weverse-media"
	weversenotice "gitlab.com/Cacophony/Worker/plugins/weverse-notice"
	"gitlab.com/Cacophony/go-kit/events"
	"gitlab.com/Cacophony/go-kit/featureflag"
	"gitlab.com/Cacophony/go-kit/interfaces"
	"gitlab.com/Cacophony/go-kit/state"
	"go.uber.org/zap"
)

type Plugin interface {
	Name() string

	Interval() time.Duration

	// TODO: add context for deadline
	Start(common.StartParameters) error

	// TODO: add context for deadline
	Stop(common.StopParameters) error

	Localizations() []interfaces.Localization

	Run(run *common.Run) error
}

var (
	PluginList = []Plugin{
		// &instagramigtv.Plugin{},

		&gall.Plugin{},
		&rss.Plugin{},
		&instagram.Plugin{},
		&instagramstories.Plugin{},
		&serverlist.Plugin{},
		&stockssymbols.Plugin{},
		&eventscheduler.Plugin{},
		&patrons.Plugin{},
		&weverseartist.Plugin{},
		&weversemedia.Plugin{},
		&weversenotice.Plugin{},
	}

	LocalizationsList []interfaces.Localization
)

func StartPlugins(
	logger *zap.Logger,
	db *gorm.DB,
	redis *redis.Client,
	tokens map[string]string,
	state *state.State,
	featureFlagger *featureflag.FeatureFlagger,
	publisher *events.Publisher,
) {
	var err error
	for _, plugin := range PluginList {
		err = plugin.Start(common.StartParameters{
			Logger:         logger,
			DB:             db,
			Redis:          redis,
			Tokens:         tokens,
			State:          state,
			FeatureFlagger: featureFlagger,
			Publisher:      publisher,
		})
		if err != nil {
			logger.Error("failed to start plugin",
				zap.Error(err),
			)
		}
		// TODO: do not send run plugins that failed to start

		LocalizationsList = append(LocalizationsList, plugin.Localizations()...)
	}
}

func StopPlugins(
	logger *zap.Logger,
	db *gorm.DB,
	redis *redis.Client,
	tokens map[string]string,
	state *state.State,
	featureFlagger *featureflag.FeatureFlagger,
	publisher *events.Publisher,
) {
	var err error
	for _, plugin := range PluginList {
		err = plugin.Stop(common.StopParameters{
			Logger:         logger,
			DB:             db,
			Redis:          redis,
			Tokens:         tokens,
			State:          state,
			FeatureFlagger: featureFlagger,
			Publisher:      publisher,
		})
		if err != nil {
			logger.Error("failed to stop plugin",
				zap.Error(err),
			)
		}
	}
}
