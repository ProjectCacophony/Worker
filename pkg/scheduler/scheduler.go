package scheduler

import (
	"context"
	"time"

	"gitlab.com/Cacophony/go-kit/featureflag"

	"gitlab.com/Cacophony/Worker/plugins"
	"gitlab.com/Cacophony/Worker/plugins/common"
	"go.uber.org/zap"
)

type Scheduler struct {
	logger         *zap.Logger
	featureFlagger *featureflag.FeatureFlagger
}

func NewScheduler(
	logger *zap.Logger,
	featureFlagger *featureflag.FeatureFlagger,
) *Scheduler {
	return &Scheduler{
		logger:         logger,
		featureFlagger: featureFlagger,
	}
}

func (s *Scheduler) Start() {
	var err error

	for {
		for _, plugin := range plugins.PluginList {
			if !s.featureFlagger.IsEnabled(featureFlagPluginKey(plugin.Name()), true) {
				s.logger.Debug("skipping plugin as it is disabled by feature flags",
					zap.String("plugin_name", plugin.Name()),
				)
				continue
			}

			run := common.NewRun(plugin.Name())

			logger := s.logger.With(
				zap.String("plugin", plugin.Name()),
				zap.String("launch", run.Launch.String()),
			)

			run.WithContext(context.Background())
			run.WithLogger(logger)

			err = plugin.Run(run)
			if err != nil {
				logger.Error("run execution failed",
					zap.Error(err),
				)
			}

		}

		time.Sleep(60 * time.Second)
	}
}

func featureFlagPluginKey(pluginName string) string {
	return "plugin-" + pluginName
}
