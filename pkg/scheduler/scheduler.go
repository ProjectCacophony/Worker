package scheduler

import (
	"context"
	"time"

	"gitlab.com/Cacophony/Worker/plugins"
	"gitlab.com/Cacophony/Worker/plugins/common"
	"gitlab.com/Cacophony/go-kit/featureflag"
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
	for _, plugin := range plugins.PluginList {
		go s.startPluginLoop(plugin)

		// some terrible way so not all plugins launch at the same time
		time.Sleep(1 * time.Second)
	}
}

func (s *Scheduler) startPluginLoop(plugin plugins.Plugin) {
	var err error
	var l *zap.Logger

	logger := s.logger.With(
		zap.String("plugin", plugin.Name()),
	)

	for {
		if !s.featureFlagger.IsEnabled(featureFlagPluginKey(plugin.Name()), true) {
			logger.Debug("skipping plugin as it is disabled by feature flags")
			continue
		}

		run := common.NewRun(plugin.Name())

		l = logger.With(zap.Time("launch", run.Launch))

		run.WithContext(context.Background())
		run.WithLogger(l)

		err = plugin.Run(run)
		if err != nil {
			l.Error("run execution failed",
				zap.Error(err),
			)
		}

		time.Sleep(plugin.Interval())
	}
}

func featureFlagPluginKey(pluginName string) string {
	return "plugin-" + pluginName
}
