package plugins

import (
	"github.com/go-redis/redis"
	"github.com/jinzhu/gorm"
	"gitlab.com/Cacophony/Worker/plugins/common"
	"gitlab.com/Cacophony/Worker/plugins/gall"
	"gitlab.com/Cacophony/Worker/plugins/instagram"
	"gitlab.com/Cacophony/Worker/plugins/rss"
	"gitlab.com/Cacophony/go-kit/featureflag"
	"gitlab.com/Cacophony/go-kit/interfaces"
	"gitlab.com/Cacophony/go-kit/state"
	"go.uber.org/zap"
)

type Plugin interface {
	Name() string

	// TODO: add context for deadline
	Start(common.StartParameters) error

	// TODO: add context for deadline
	Stop(common.StopParameters) error

	Localisations() []interfaces.Localisation

	Run(run *common.Run) error
}

// nolint: gochecknoglobals
var (
	PluginList = []Plugin{
		&gall.Plugin{},
		&rss.Plugin{},
		&instagram.Plugin{},
	}

	LocalisationsList []interfaces.Localisation
)

func StartPlugins(
	logger *zap.Logger,
	db *gorm.DB,
	redis *redis.Client,
	tokens map[string]string,
	state *state.State,
	featureFlagger *featureflag.FeatureFlagger,
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
		})
		if err != nil {
			logger.Error("failed to start plugin",
				zap.Error(err),
			)
		}
		// TODO: do not send run plugins that failed to start

		LocalisationsList = append(LocalisationsList, plugin.Localisations()...)
	}
}

func StopPlugins(
	logger *zap.Logger,
	db *gorm.DB,
	redis *redis.Client,
	tokens map[string]string,
	state *state.State,
	featureFlagger *featureflag.FeatureFlagger,
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
		})
		if err != nil {
			logger.Error("failed to stop plugin",
				zap.Error(err),
			)
		}
	}
}
