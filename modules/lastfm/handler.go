package lastfm

import (
	"github.com/sirupsen/logrus"
	"gitlab.com/project-d-collab/dhelpers"
	"gitlab.com/project-d-collab/dhelpers/cache"
)

// Module is a struct for the module
type Module struct{}

// GetJobs defines all jobs
func (m *Module) GetJobs() []dhelpers.Job {
	return []dhelpers.Job{
		{
			Name:     "LastFm:ServerStats",
			Cron:     "@every 6h",
			Job:      JobServerStats,
			AtLaunch: true,
		},
	}
}

// GetTranslationFiles defines all translation files for the module
func (m *Module) GetTranslationFiles() []string {
	return []string{}
}

// Init is called on bot startup
func (m *Module) Init() {
}

// Uninit is called on normal bot shutdown
func (m *Module) Uninit() {
}

func logger() *logrus.Entry {
	return cache.GetLogger().WithField("module", "lastfm")
}
