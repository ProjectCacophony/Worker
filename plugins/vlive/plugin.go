package vlive

import (
	"net/http"
	"time"

	vlive_go "github.com/Seklfreak/vlive-go"
	"github.com/go-redis/redis"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"gitlab.com/Cacophony/Worker/plugins/common"
	"gitlab.com/Cacophony/go-kit/interfaces"
	"gitlab.com/Cacophony/go-kit/localization"
	"gitlab.com/Cacophony/go-kit/state"
	"go.uber.org/zap"
)

type Plugin struct {
	logger      *zap.Logger
	state       *state.State
	db          *gorm.DB
	tokens      map[string]string
	redis       *redis.Client
	vliveClient *vlive_go.VLive
}

func (p *Plugin) Name() string {
	return "vlive"
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

	p.redis = params.Redis

	p.vliveClient = vlive_go.NewVLive(
		&http.Client{
			Timeout: 1 * time.Minute,
		},
	)

	return nil
}

func (p *Plugin) Stop(params common.StopParameters) error {
	return nil
}

func (p *Plugin) Localizations() []interfaces.Localization {
	local, err := localization.NewFileSource("assets/translations/vlive.en.toml", "en")
	if err != nil {
		p.logger.Error("failed to load localisation", zap.Error(err))
	}

	return []interfaces.Localization{local}
}

func (p *Plugin) Run(run *common.Run) error {
	run.Logger().Info("run started")

	videos, err := p.vliveClient.Recents()
	if err != nil {
		return errors.Wrap(err, "failure getting recent videos")
	}

	for _, video := range videos {
		err = p.handleVideo(video)
		if err != nil {
			run.Logger().Error("failure handling video", zap.Error(err), zap.String("video_seq", video.Seq))
		}
	}

	return nil
}
